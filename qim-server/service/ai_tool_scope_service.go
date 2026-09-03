package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/dshmyz/qim/qim-server/model"
	pkgErr "github.com/dshmyz/qim/qim-server/pkg/errors"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"gorm.io/gorm"
)

// ai_tool_scope_service.go — 各 AI 入口工具面（白名单）的运行时可配置层。
// 存储复用 system_configs 表（config_key = ai.tools.scope.<scope>，value = JSON 字符串数组，
// type=json）。未配置时回退 ai_tool_scopes.go 中的代码默认值——默认行为不变，
// admin 显式覆盖才生效；Set/Reset 即时刷新缓存（与 AiThresholdService「改完即生效」同语义）。

// 工具面作用域标识。
const (
	ToolScopeSidebar = "sidebar" // 侧边栏 AI 元对话
	ToolScopeBotDM   = "bot_dm"  // 专属机器人 1:1 会话
	ToolScopeGroup   = "group"   // 群 @AI（外部 MCP 工具在此之上动态追加）
)

// configKey 返回作用域对应的 system_configs 键。
func toolScopeConfigKey(scope string) string {
	return "ai.tools.scope." + scope
}

// defaultScopeTools 返回作用域的代码默认白名单（ai_tool_scopes.go 单一来源）。
func defaultScopeTools(scope string) []string {
	switch scope {
	case ToolScopeSidebar:
		return SidebarAllowedTools
	case ToolScopeBotDM:
		return botAllowedTools
	case ToolScopeGroup:
		return groupAssistantToolWhitelist
	}
	return nil
}

// DefaultScopeTools 返回作用域的代码默认白名单（供 admin 界面回显默认值）。
func DefaultScopeTools(scope string) []string {
	return append([]string(nil), defaultScopeTools(scope)...)
}

// ValidToolScope 校验作用域标识是否合法。
func ValidToolScope(scope string) bool {
	return defaultScopeTools(scope) != nil
}

// ToolScopeService 工具面作用域配置服务。
type ToolScopeService struct {
	db    *gorm.DB
	mu    sync.RWMutex
	cache map[string]*[]string // scope -> 配置值指针；nil 值项 = 已查过但未覆盖（用默认）
}

// NewToolScopeService 创建工具面配置服务。
func NewToolScopeService(db *gorm.DB) *ToolScopeService {
	return &ToolScopeService{db: db, cache: make(map[string]*[]string)}
}

// ScopeTools 返回作用域的生效工具列表（大小写规范以配置为准）。
// nil 接收者安全：服务未注入（如部分测试场景）时直接返回代码默认值。
func (s *ToolScopeService) ScopeTools(scope string) []string {
	if s == nil {
		return append([]string(nil), defaultScopeTools(scope)...)
	}
	s.mu.RLock()
	cached, ok := s.cache[scope]
	s.mu.RUnlock()
	// ok=true 即已查过：cached=nil 表示「已查过但未覆盖（用默认）」，直接返回默认，
	// 不能因 cached==nil 就回源重查（否则未覆盖作用域每次都多一次 DB 查询）。
	if ok {
		if cached != nil {
			return append([]string(nil), (*cached)...)
		}
		return append([]string(nil), defaultScopeTools(scope)...)
	}

	loaded := s.loadFromDB(scope)
	s.mu.Lock()
	s.cache[scope] = loaded
	s.mu.Unlock()
	if loaded != nil {
		return append([]string(nil), (*loaded)...)
	}
	return append([]string(nil), defaultScopeTools(scope)...)
}

// IsOverridden 作用域是否被 admin 显式覆盖（供 admin 界面回显）。
func (s *ToolScopeService) IsOverridden(scope string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	cached, ok := s.cache[scope]
	s.mu.RUnlock()
	if ok {
		return cached != nil
	}
	loaded := s.loadFromDB(scope)
	s.mu.Lock()
	s.cache[scope] = loaded
	s.mu.Unlock()
	return loaded != nil
}

// SetScopeTools 覆盖作用域工具面。即写 system_configs 即刷缓存。
func (s *ToolScopeService) SetScopeTools(scope string, tools []string) error {
	if !ValidToolScope(scope) {
		return fmt.Errorf("%w: %q", ErrToolScopeInvalid, scope)
	}
	if s.db == nil {
		return ErrToolScopeServiceNotReady
	}
	value, err := json.Marshal(tools)
	if err != nil {
		return fmt.Errorf("序列化工具列表失败: %w", err)
	}

	cfg := model.SystemConfig{
		ConfigKey: toolScopeConfigKey(scope),
		Value:     string(value),
		Type:      "json",
		Desc:      "AI 工具面覆盖配置（scope=" + scope + "），删除该行即恢复代码默认",
	}
	// upsert：存在则更新 value，否则创建
	if err := s.db.Where("config_key = ?", cfg.ConfigKey).
		Assign(model.SystemConfig{Value: cfg.Value, Type: "json"}).
		FirstOrCreate(&cfg).Error; err != nil {
		return fmt.Errorf("写入工具面配置失败: %w", err)
	}

	s.mu.Lock()
	s.cache[scope] = &tools
	s.mu.Unlock()
	return nil
}

// ResetScope 恢复代码默认（删除覆盖行 + 清缓存）。
func (s *ToolScopeService) ResetScope(scope string) error {
	if !ValidToolScope(scope) {
		return fmt.Errorf("%w: %q", ErrToolScopeInvalid, scope)
	}
	if s.db == nil {
		return ErrToolScopeServiceNotReady
	}
	if err := s.db.Where("config_key = ?", toolScopeConfigKey(scope)).
		Delete(&model.SystemConfig{}).Error; err != nil {
		return fmt.Errorf("删除工具面配置失败: %w", err)
	}
	s.mu.Lock()
	delete(s.cache, scope)
	s.mu.Unlock()
	return nil
}

// loadFromDB 读取覆盖配置。返回 nil 表示未覆盖（非错误）。
func (s *ToolScopeService) loadFromDB(scope string) *[]string {
	if s.db == nil {
		return nil
	}
	var cfg model.SystemConfig
	if err := s.db.Where("config_key = ?", toolScopeConfigKey(scope)).First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		// 读失败按未覆盖降级并告警，不阻断对话主链路
		logger.WithModule("ToolScope").Error("读取工具面配置失败，回退默认", "scope", scope, "error", err)
		return nil
	}
	var tools []string
	if err := json.Unmarshal([]byte(cfg.Value), &tools); err != nil {
		logger.WithModule("ToolScope").Error("工具面配置解析失败，回退默认", "scope", scope, "value", cfg.Value, "error", err)
		return nil
	}
	return &tools
}

// 工具面配置服务的哨兵错误。
var (
	ErrToolScopeInvalid         = pkgErr.BadRequestError("非法的工具面作用域")
	ErrToolScopeServiceNotReady = pkgErr.InternalError("工具面配置服务不可用")
)
