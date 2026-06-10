package syncer

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/orgsync"

	"github.com/go-ldap/ldap/v3"
)

type LDAPConfig struct {
	Server           string            `json:"server"`
	Port             int               `json:"port"`
	UseTLS           bool              `json:"use_tls"`
	BaseDN           string            `json:"base_dn"`
	BindDN           string            `json:"bind_dn"`
	BindPassword     string            `json:"bind_password"`
	DepartmentFilter string            `json:"department_filter"`
	UserFilter       string            `json:"user_filter"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	IncludeBaseDN    *bool             `json:"include_base_dn"`
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
	if cfg.Server == "" {
		return nil, fmt.Errorf("LDAP配置缺少server")
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

	addr := fmt.Sprintf("%s:%d", s.config.Server, s.config.Port)

	if s.config.UseTLS {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{
			ServerName:         s.config.Server,
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

	departments, dnToUUID, err := s.fetchDepartments(conn)
	if err != nil {
		return nil, fmt.Errorf("获取部门数据失败: %w", err)
	}

	users, err := s.fetchUsers(conn, dnToUUID)
	if err != nil {
		return nil, fmt.Errorf("获取用户数据失败: %w", err)
	}

	userDeptRels, err := s.fetchUserDepartmentRelations(users, dnToUUID)
	if err != nil {
		return nil, fmt.Errorf("获取用户部门关系失败: %w", err)
	}

	result := &orgsync.OrgData{
		Departments:       departments,
		Users:             users,
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

func (s *LDAPSyncer) fetchDepartments(conn *ldap.Conn) ([]orgsync.DepartmentInfo, map[string]string, error) {
	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		s.config.DepartmentFilter,
		[]string{"dn", "ou", "departmentNumber", "entryUUID"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, nil, err
	}

	// 构建 DN → entryUUID 映射，用于解析父部门关系
	dnToUUID := make(map[string]string, len(sr.Entries))
	for _, entry := range sr.Entries {
		uuid := entry.GetAttributeValue("entryUUID")
		if uuid == "" {
			uuid = entry.DN // 回退到 DN
		}
		dnToUUID[entry.DN] = uuid
	}

	departments := s.departmentsFromEntries(sr.Entries)

	return departments, dnToUUID, nil
}

func (s *LDAPSyncer) departmentsFromEntries(entries []*ldap.Entry) []orgsync.DepartmentInfo {
	dnToUUID := make(map[string]string, len(entries))
	for _, entry := range entries {
		uuid := entry.GetAttributeValue("entryUUID")
		if uuid == "" {
			uuid = entry.DN
		}
		dnToUUID[entry.DN] = uuid
	}

	departments := make([]orgsync.DepartmentInfo, 0, len(entries))
	includeBaseDN := s.includeBaseDNDepartment()
	excludedBaseLevelOffset := 0
	if !includeBaseDN {
		excludedBaseLevelOffset = orgRDNCount(parseDN(s.config.BaseDN))
	}
	for _, entry := range entries {
		if !includeBaseDN && sameDN(entry.DN, s.config.BaseDN) {
			continue
		}

		name := entry.GetAttributeValue("ou")
		if name == "" {
			continue
		}

		level := 0
		parentID := ""
		dnParts := parseDN(entry.DN)

		// 计算层级：跳过 DC 部分，从 OU/CN 开始计数
		ouIndex := 0
		for _, part := range dnParts {
			if len(part) >= 3 && (part[:3] == "ou=" || part[:3] == "OU=" ||
				(len(part) >= 3 && part[:3] == "cn=") || (len(part) >= 3 && part[:3] == "CN=")) {
				ouIndex++
			}
		}
		level = ouIndex - 1
		level -= excludedBaseLevelOffset
		if level < 0 {
			level = 0
		}

		// 从 DN 解析父部门的 entryUUID
		if len(dnParts) > 1 {
			parentDN := joinDN(dnParts[1:])
			if includeBaseDN || !sameDN(parentDN, s.config.BaseDN) {
				parentID = dnToUUID[parentDN]
			}
		}

		departments = append(departments, orgsync.DepartmentInfo{
			ID:       dnToUUID[entry.DN],
			Name:     name,
			ParentID: parentID,
			Level:    level,
		})
	}

	return departments
}

// joinDN 将 DN 各部分重新拼接为 DN 字符串
func joinDN(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ","
		}
		result += part
	}
	return result
}

func (s *LDAPSyncer) fetchUsers(conn *ldap.Conn, dnToUUID map[string]string) ([]orgsync.UserInfo, error) {
	attrMap := s.config.AttributeMapping
	attrs := make([]string, 0, len(attrMap)+3)
	for _, v := range attrMap {
		attrs = append(attrs, v)
	}
	attrs = append(attrs, "dn", "departmentNumber", "entryUUID")

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

	// 将用户 DN 也加入映射
	for _, entry := range sr.Entries {
		uuid := entry.GetAttributeValue("entryUUID")
		if uuid == "" {
			uuid = entry.DN
		}
		dnToUUID[entry.DN] = uuid
	}

	users := make([]orgsync.UserInfo, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		username := entry.GetAttributeValue(attrMap["username"])
		if username == "" {
			continue
		}

		userDeptID := s.resolveUserDepartmentID(entry, dnToUUID)

		user := orgsync.UserInfo{
			ID:           dnToUUID[entry.DN],
			Username:     username,
			Nickname:     entry.GetAttributeValue(attrMap["nickname"]),
			Email:        entry.GetAttributeValue(attrMap["email"]),
			Phone:        entry.GetAttributeValue(attrMap["phone"]),
			DepartmentID: userDeptID,
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *LDAPSyncer) fetchUserDepartmentRelations(users []orgsync.UserInfo, dnToUUID map[string]string) ([]orgsync.UserDeptRelation, error) {
	relations := make([]orgsync.UserDeptRelation, 0)

	for _, user := range users {
		if user.DepartmentID != "" {
			relations = append(relations, orgsync.UserDeptRelation{
				UserID:       user.ID,
				DepartmentID: user.DepartmentID,
				IsLeader:     false,
			})
		}

	}

	return relations, nil
}

func (s *LDAPSyncer) resolveUserDepartmentID(entry *ldap.Entry, dnToUUID map[string]string) string {
	if parentDN := getParentDN(entry.DN); parentDN != "" {
		if s.includeBaseDNDepartment() || !sameDN(parentDN, s.config.BaseDN) {
			if deptUUID, ok := dnToUUID[parentDN]; ok {
				return deptUUID
			}
		}
	}

	if deptNumbers := entry.GetAttributeValues("departmentNumber"); len(deptNumbers) > 0 {
		return deptNumbers[0]
	}

	return ""
}

func (s *LDAPSyncer) includeBaseDNDepartment() bool {
	return s.config.IncludeBaseDN == nil || *s.config.IncludeBaseDN
}

func sameDN(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func orgRDNCount(parts []string) int {
	count := 0
	for _, part := range parts {
		if len(part) >= 3 && (part[:3] == "ou=" || part[:3] == "OU=" ||
			part[:3] == "cn=" || part[:3] == "CN=") {
			count++
		}
	}
	return count
}

// getParentDN 从 DN 中去掉第一段 RDN，返回父节点 DN
// 例如: cn=张三,ou=技术部,dc=example,dc=com → ou=技术部,dc=example,dc=com
func getParentDN(dn string) string {
	parts := parseDN(dn)
	if len(parts) >= 2 {
		return joinDN(parts[1:])
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
