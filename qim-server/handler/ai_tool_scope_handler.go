package handler

import (
	"errors"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

// 工具面作用域的展示元信息（顺序即 admin 界面展示顺序）。
var toolScopeMeta = []struct {
	Scope string
	Name  string
	Desc  string
}{
	{service.ToolScopeSidebar, "侧边栏 AI", "AI 助手侧边栏元对话可调用的工具"},
	{service.ToolScopeBotDM, "AI 助手 Bot", "与 AI 助手 1:1 私聊可调用的工具（按对话者本人数据隔离）"},
	{service.ToolScopeGroup, "群 @AI", "群里 @AI 触发时可调用的内置工具（外部 MCP 工具另行动态追加）"},
}

// ListToolScopes GET /admin/ai/tool-scopes
// 返回各作用域的生效工具面、默认值、覆盖状态，以及注册表中的全部工具名（供前端下拉）。
func (h *AIHandler) ListToolScopes(c *gin.Context) {
	registered := []string{}
	if h.toolRegistry != nil {
		for _, t := range h.toolRegistry.ListTools() {
			if name, ok := t["name"].(string); ok {
				registered = append(registered, name)
			}
		}
	}

	scopes := make([]gin.H, 0, len(toolScopeMeta))
	for _, m := range toolScopeMeta {
		overridden := h.toolScopes.IsOverridden(m.Scope)
		tools := h.toolScopes.ScopeTools(m.Scope)
		scopes = append(scopes, gin.H{
			"scope":         m.Scope,
			"name":          m.Name,
			"desc":          m.Desc,
			"tools":         tools,
			"default_tools": service.DefaultScopeTools(m.Scope),
			"overridden":    overridden,
		})
	}

	response.Success(c, gin.H{
		"scopes":           scopes,
		"registered_tools": registered,
	})
}

// UpdateToolScopeRequest PUT /admin/ai/tool-scopes/:scope 的请求体。
type UpdateToolScopeRequest struct {
	// Tools 覆盖后的工具名列表；reset=true 时忽略。
	Tools []string `json:"tools"`
	// Reset 恢复代码默认（删除覆盖配置）。
	Reset bool `json:"reset"`
}

// UpdateToolScope PUT /admin/ai/tool-scopes/:scope
// 覆盖或重置某作用域的工具面。工具名校验注册表（大小写不敏感命中后按注册名规范化），
// 防止配置出永远无法命中的工具名。修改即时生效（缓存同步刷新）。
func (h *AIHandler) UpdateToolScope(c *gin.Context) {
	if h.toolScopes == nil {
		response.InternalServerError(c, "工具面配置服务不可用")
		return
	}
	scope := c.Param("scope")
	if !service.ValidToolScope(scope) {
		response.BadRequest(c, "非法的工具面作用域: "+scope)
		return
	}

	var req UpdateToolScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Reset {
		if err := h.toolScopes.ResetScope(scope); err != nil {
			response.InternalServerError(c, "恢复默认失败: "+err.Error())
			return
		}
		response.Success(c, gin.H{"scope": scope, "tools": h.toolScopes.ScopeTools(scope), "overridden": false})
		return
	}

	if h.toolRegistry == nil {
		response.InternalServerError(c, "AI工具注册表未初始化")
		return
	}

	// 规范化：大小写不敏感匹配注册名 + 去重保序
	normalized := make([]string, 0, len(req.Tools))
	seen := map[string]bool{}
	for _, name := range req.Tools {
		tool, ok := h.toolRegistry.GetTool(strings.TrimSpace(name))
		if !ok || tool == nil {
			response.BadRequest(c, "工具不存在: "+name)
			return
		}
		if !seen[tool.Name()] {
			seen[tool.Name()] = true
			normalized = append(normalized, tool.Name())
		}
	}
	if len(normalized) == 0 {
		response.BadRequest(c, "工具列表不能为空（如需恢复默认请传 reset=true）")
		return
	}

	if err := h.toolScopes.SetScopeTools(scope, normalized); err != nil {
		if errors.Is(err, service.ErrToolScopeInvalid) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "保存失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"scope": scope, "tools": h.toolScopes.ScopeTools(scope), "overridden": true})
}
