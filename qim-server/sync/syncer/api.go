package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/orgsync"
)

type APIConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	AuthToken   string            `json:"auth_token"`
	Timeout     int               `json:"timeout"`
}

type APISyncer struct {
	config *APIConfig
	client *http.Client
	dbID   uint
}

func NewAPISyncer(model *model.OrgSyncConfig) (*APISyncer, error) {
	var cfg APIConfig
	if err := json.Unmarshal([]byte(model.Config), &cfg); err != nil {
		return nil, fmt.Errorf("解析API配置失败: %w", err)
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("API配置缺少url")
	}
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30
	}

	return &APISyncer{
		config: &cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		dbID:   model.ID,
	}, nil
}

func (s *APISyncer) Name() string {
	return "api"
}

func (s *APISyncer) Fetch(ctx context.Context, configStr string) (*orgsync.OrgData, error) {
	req, err := http.NewRequestWithContext(ctx, s.config.Method, s.config.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	if s.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.AuthToken)
	}
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API返回异常状态码: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var apiResp struct {
		Departments []apiDepartment `json:"departments"`
		Users       []apiUser       `json:"users"`
		Groups      []apiGroup      `json:"groups"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析API响应失败: %w", err)
	}

	data := &orgsync.OrgData{
		Departments: make([]orgsync.DepartmentInfo, 0, len(apiResp.Departments)),
		Users:       make([]orgsync.UserInfo, 0, len(apiResp.Users)),
		Groups:      make([]orgsync.GroupInfo, 0, len(apiResp.Groups)),
	}

	for _, d := range apiResp.Departments {
		data.Departments = append(data.Departments, orgsync.DepartmentInfo{
			ID:       d.ID,
			Name:     d.Name,
			ParentID: d.ParentID,
			Level:    d.Level,
		})
	}

	for _, u := range apiResp.Users {
		data.Users = append(data.Users, orgsync.UserInfo{
			ID:           u.ID,
			Username:     u.Username,
			Nickname:     u.Nickname,
			Email:        u.Email,
			Phone:        u.Phone,
			Avatar:       u.Avatar,
			DepartmentID: u.DepartmentID,
			Position:     u.Position,
		})
		if u.DepartmentID != "" {
			data.UserDeptRelations = append(data.UserDeptRelations, orgsync.UserDeptRelation{
				UserID:       u.ID,
				DepartmentID: u.DepartmentID,
			})
		}
	}

	for _, g := range apiResp.Groups {
		data.Groups = append(data.Groups, orgsync.GroupInfo{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
		})
	}

	return data, nil
}

func (s *APISyncer) Sync(ctx context.Context, data *orgsync.OrgData) (*orgsync.SyncResult, error) {
	return &orgsync.SyncResult{
		Success: true,
		Message: "API数据已获取，由引擎执行本地同步",
	}, nil
}

type apiDepartment struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
	Level    int    `json:"level"`
}

type apiUser struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Avatar       string `json:"avatar"`
	DepartmentID string `json:"department_id"`
	Position     string `json:"position"`
}

type apiGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
