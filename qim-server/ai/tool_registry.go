package ai

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// Tool 定义 AI 可调用工具的进程内接口。
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(params map[string]interface{}, ctx *CallerContext) (interface{}, error)
}

// CallerContext 调用者上下文（用于权限控制）
type CallerContext struct {
	UserID         uint
	Username       string
	Role           string
	GroupID        uint
	GroupRole      string
	ConversationID uint     // 当前会话 ID，工具可据此获取上下文
	AllowedTools   []string // 允许使用的工具名列表，为空则允许全部
}

// ToolRegistry 是 QIM server 进程内的 AI 工具注册表。
// 它不是标准 MCP server，也不负责启动 HTTP 端点；对外 MCP 网关由 cmd/qim-mcp 提供。
type ToolRegistry struct {
	tools        map[string]Tool
	enabledTools map[string]bool // 工具启用状态
	mu           sync.RWMutex
}

// NewToolRegistry 创建进程内 AI 工具注册表。
func NewToolRegistry(aiService *AIService) *ToolRegistry {
	registry := &ToolRegistry{
		tools:        make(map[string]Tool),
		enabledTools: make(map[string]bool),
	}

	if aiService != nil {
		// 运维工具（intelligent_troubleshooting 等 5 个）已移除：后端有实现但前端未接入、
		// 群聊助手白名单显式排除，属于死代码。如需恢复运维能力，参考 git 历史中的 ops_tools.go。
	}

	return registry
}

// RegisterTool 注册工具。
// 返回 error：新工具名与某已注册工具名仅大小写不同（如 send_message vs Send_Message）时
// 拒绝注册——canonicalKey 的大小写不敏感回退扫描依赖 map 迭代顺序，两个仅大小写不同的键
// 会让 GetTool/EnableTool 等的大小写不敏感查找结果不确定。同一名字重复注册仍允许（覆盖，
// 幂等，供外部 MCP 断线重连时重注册同名工具）。
func (r *ToolRegistry) RegisterTool(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := tool.Name()
	lower := strings.ToLower(name)
	for key := range r.tools {
		if key != name && strings.ToLower(key) == lower {
			return fmt.Errorf("tool %q 与既有工具 %q 仅大小写不同，拒绝注册（避免大小写不敏感查找不确定）", name, key)
		}
	}
	r.tools[name] = tool
	r.enabledTools[name] = true // 默认启用
	logger.WithModule("ToolRegistry").Info("Registered tool", "tool", name)
	return nil
}

// RemoveTool 按名字移除工具（含其启用状态）。用于外部 MCP 连接在运行时被删除/
// 禁用后把其工具从注册表摘除，避免残留脏工具。
func (r *ToolRegistry) RemoveTool(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.canonicalKey(name)
	if key != "" {
		delete(r.tools, key)
		delete(r.enabledTools, key)
	}
}

// canonicalKey 返回与 name 大小写不敏感匹配的实际注册键；未命中返回 ""。
// 工具名可能被 LLM 改写大小写（如 send_message → Send_Message）。白名单匹配已统一 ToLower
// （见 ai_service.isToolAllowed），执行路径也必须大小写不敏感，否则出现"白名单放行、
// 执行仍报 tool not found"的不一致。注意保留注册时的原始大小写（外部 MCP 工具名可能含大写），
// 不强制改小写。锁由调用方持有。
func (r *ToolRegistry) canonicalKey(name string) string {
	if _, ok := r.tools[name]; ok {
		return name
	}
	lower := strings.ToLower(name)
	for key := range r.tools {
		if strings.ToLower(key) == lower {
			return key
		}
	}
	return ""
}

// GetTool 获取工具（大小写不敏感）
func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := r.canonicalKey(name)
	if key == "" {
		return nil, false
	}
	tool, ok := r.tools[key]
	return tool, ok
}

// ListTools 列出所有工具
func (r *ToolRegistry) ListTools() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(r.tools))
	for name, tool := range r.tools {
		enabled := true
		if e, ok := r.enabledTools[name]; ok {
			enabled = e
		}
		tools = append(tools, map[string]interface{}{
			"name":        name,
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
			"enabled":     enabled,
		})
	}

	return tools
}

// EnableTool 启用工具（大小写不敏感）
func (r *ToolRegistry) EnableTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.canonicalKey(name)
	if key == "" {
		return fmt.Errorf("tool not found: %s", name)
	}
	r.enabledTools[key] = true
	return nil
}

// DisableTool 禁用工具（大小写不敏感）
func (r *ToolRegistry) DisableTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.canonicalKey(name)
	if key == "" {
		return fmt.Errorf("tool not found: %s", name)
	}
	r.enabledTools[key] = false
	return nil
}

// IsToolEnabled 检查工具是否启用（大小写不敏感）
func (r *ToolRegistry) IsToolEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := r.canonicalKey(name)
	if key == "" {
		return true // 默认启用
	}
	if e, ok := r.enabledTools[key]; ok {
		return e
	}
	return true // 默认启用
}

// ExecuteTool 执行工具
func (r *ToolRegistry) ExecuteTool(name string, params map[string]interface{}, ctx *CallerContext) (interface{}, error) {
	if !r.IsToolEnabled(name) {
		return nil, fmt.Errorf("tool disabled: %s", name)
	}
	tool, ok := r.GetTool(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(params, ctx)
}
