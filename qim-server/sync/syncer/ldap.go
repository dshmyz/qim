package syncer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"qim-server/model"
	"qim-server/orgsync"

	"github.com/go-ldap/ldap/v3"
)

type LDAPConfig struct {
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	UseSSL            bool              `json:"use_ssl"`
	BaseDN            string            `json:"base_dn"`
	BindDN            string            `json:"bind_dn"`
	BindPassword      string            `json:"bind_password"`
	DepartmentFilter  string            `json:"department_filter"`
	UserFilter        string            `json:"user_filter"`
	AttributeMapping  map[string]string `json:"attribute_mapping"`
}

type LDAPSyncer struct {
	config *LDAPConfig
	dbID   uint
}

func NewLDAPSyncer(model *model.OrgSyncConfig) (*LDAPSyncer, error) {
	var cfg LDAPConfig
	if err := json.Unmarshal([]byte(model.Config), &cfg); err != nil {
		return nil, fmt.Errorf("解析LDAP配置失败: %w", err)
	}
	if cfg.Host == "" {
		return nil, fmt.Errorf("LDAP配置缺少host")
	}
	if cfg.Port == 0 {
		cfg.Port = 389
	}
	if cfg.DepartmentFilter == "" {
		cfg.DepartmentFilter = "(objectClass=organizationalUnit)"
	}
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(objectClass=inetOrgPerson)"
	}
	if cfg.AttributeMapping == nil {
		cfg.AttributeMapping = map[string]string{
			"username": "uid",
			"nickname": "cn",
			"email":    "mail",
		}
	}

	return &LDAPSyncer{
		config: &cfg,
		dbID:   model.ID,
	}, nil
}

func (s *LDAPSyncer) Name() string {
	return "ldap"
}

func (s *LDAPSyncer) Fetch(ctx context.Context, configStr string) (*orgsync.OrgData, error) {
	var conn *ldap.Conn
	var err error

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	if s.config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{
			ServerName: s.config.Host,
			InsecureSkipVerify: false,
		})
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("连接LDAP服务器失败: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(s.config.BindDN, s.config.BindPassword); err != nil {
		return nil, fmt.Errorf("LDAP绑定失败: %w", err)
	}

	departments, err := s.fetchDepartments(conn)
	if err != nil {
		return nil, fmt.Errorf("获取部门数据失败: %w", err)
	}

	users, err := s.fetchUsers(conn)
	if err != nil {
		return nil, fmt.Errorf("获取用户数据失败: %w", err)
	}

	userDeptRels, err := s.fetchUserDepartmentRelations(conn, users)
	if err != nil {
		return nil, fmt.Errorf("获取用户部门关系失败: %w", err)
	}

	result := &orgsync.OrgData{
		Departments:     departments,
		Users:           users,
		UserDeptRelations: userDeptRels,
	}

	return result, nil
}

func (s *LDAPSyncer) Sync(ctx context.Context, data *orgsync.OrgData) (*orgsync.SyncResult, error) {
	return &orgsync.SyncResult{
		Success: true,
		Message: "LDAP数据已获取，由引擎执行本地同步",
	}, nil
}

func (s *LDAPSyncer) fetchDepartments(conn *ldap.Conn) ([]orgsync.DepartmentInfo, error) {
	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		s.config.DepartmentFilter,
		[]string{"dn", "ou", "departmentNumber"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	departments := make([]orgsync.DepartmentInfo, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		name := entry.GetAttributeValue("ou")
		if name == "" {
			continue
		}

		level := 0
		parentID := ""
		dnParts := parseDN(entry.DN)
		for i, part := range dnParts {
			if part == name {
				level = i
				break
			}
		}

		departments = append(departments, orgsync.DepartmentInfo{
			ID:       entry.DN,
			Name:     name,
			ParentID: parentID,
			Level:    level,
		})
	}

	return departments, nil
}

func (s *LDAPSyncer) fetchUsers(conn *ldap.Conn) ([]orgsync.UserInfo, error) {
	attrMap := s.config.AttributeMapping
	attrs := make([]string, 0, len(attrMap)+2)
	for _, v := range attrMap {
		attrs = append(attrs, v)
	}
	attrs = append(attrs, "dn", "departmentNumber")

	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		s.config.UserFilter,
		attrs,
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	users := make([]orgsync.UserInfo, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		username := entry.GetAttributeValue(attrMap["username"])
		if username == "" {
			continue
		}

		userDeptID := ""
		if deptNumbers := entry.GetAttributeValues("departmentNumber"); len(deptNumbers) > 0 {
			userDeptID = deptNumbers[0]
		}

		user := orgsync.UserInfo{
			ID:       entry.DN,
			Username: username,
			Nickname: entry.GetAttributeValue(attrMap["nickname"]),
			Email:    entry.GetAttributeValue(attrMap["email"]),
			Phone:    entry.GetAttributeValue(attrMap["phone"]),
			DepartmentID: userDeptID,
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *LDAPSyncer) fetchUserDepartmentRelations(conn *ldap.Conn, users []orgsync.UserInfo) ([]orgsync.UserDeptRelation, error) {
	relations := make([]orgsync.UserDeptRelation, 0)

	for _, user := range users {
		if user.DepartmentID != "" {
			relations = append(relations, orgsync.UserDeptRelation{
				UserID:       user.ID,
				DepartmentID: user.DepartmentID,
				IsLeader:     false,
			})
		}

		if userDeptEntries := getUserDepartmentFromDN(user.ID); userDeptEntries != "" {
			relations = append(relations, orgsync.UserDeptRelation{
				UserID:       user.ID,
				DepartmentID: userDeptEntries,
				IsLeader:     false,
			})
		}
	}

	return relations, nil
}

func getUserDepartmentFromDN(dn string) string {
	parts := parseDN(dn)
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return ""
}

func parseDN(dn string) []string {
	parts := make([]string, 0)
	current := ""
	escaped := false
	for _, c := range dn {
		if escaped {
			current += string(c)
			escaped = false
		} else if c == '\\' {
			escaped = true
		} else if c == ',' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
