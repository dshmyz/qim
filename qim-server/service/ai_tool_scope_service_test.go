package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolScopeServiceDefaultsWhenNotOverridden(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	svc := NewToolScopeService(db)

	// 未覆盖 → 代码默认
	assert.Equal(t, SidebarAllowedTools, svc.ScopeTools(ToolScopeSidebar))
	assert.Equal(t, botAllowedTools, svc.ScopeTools(ToolScopeBotDM))
	assert.Equal(t, groupAssistantToolWhitelist, svc.ScopeTools(ToolScopeGroup))
	assert.False(t, svc.IsOverridden(ToolScopeSidebar))

	// nil 接收者安全：未注入服务的场景回退默认
	var nilSvc *ToolScopeService
	assert.Equal(t, SidebarAllowedTools, nilSvc.ScopeTools(ToolScopeSidebar))
}

func TestToolScopeServiceOverrideHotEffective(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	svc := NewToolScopeService(db)

	require.NoError(t, svc.SetScopeTools(ToolScopeSidebar, []string{"search_knowledge", "list_tasks"}))

	// 即时生效（缓存同步刷新）
	assert.Equal(t, []string{"search_knowledge", "list_tasks"}, svc.ScopeTools(ToolScopeSidebar))
	assert.True(t, svc.IsOverridden(ToolScopeSidebar))

	// 其他作用域不受影响
	assert.Equal(t, botAllowedTools, svc.ScopeTools(ToolScopeBotDM))

	// 新实例（模拟重启）读库仍生效
	fresh := NewToolScopeService(svc.db)
	assert.Equal(t, []string{"search_knowledge", "list_tasks"}, fresh.ScopeTools(ToolScopeSidebar))
}

func TestToolScopeServiceReset(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	svc := NewToolScopeService(db)
	require.NoError(t, svc.SetScopeTools(ToolScopeGroup, []string{"search_messages"}))
	require.True(t, svc.IsOverridden(ToolScopeGroup))

	require.NoError(t, svc.ResetScope(ToolScopeGroup))
	assert.Equal(t, groupAssistantToolWhitelist, svc.ScopeTools(ToolScopeGroup))
	assert.False(t, svc.IsOverridden(ToolScopeGroup))
}

func TestToolScopeServiceInvalidScope(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	svc := NewToolScopeService(db)
	err := svc.SetScopeTools("hacky", []string{"search_files"})
	assert.Error(t, err)
	assert.False(t, svc.IsOverridden("hacky"))
}

// 群作用域即使被 admin 覆盖为包含 send_message，群 @AI 实际放行也必须剔除它：
// 群路径 callerCtx 无 ConfirmTools，send_message 会走无确认直发分支（群助手获得
// 静默代发能力，超出群管理工具边界）。过滤点在 groupAssistantAllowedTools（唯一消费口）。
func TestGroupAssistantAllowedToolsExcludesSendMessage(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	svc := NewToolScopeService(db)

	// admin 显式把 send_message 配进群作用域（SetScopeTools 不校验具体工具名）
	require.NoError(t, svc.SetScopeTools(ToolScopeGroup, []string{
		"search_messages", "send_message", "group_summary",
	}))

	g := &SmartReplyGraph{toolScopes: svc}
	allowed := g.groupAssistantAllowedTools()
	assert.NotContains(t, allowed, "send_message")
	assert.Contains(t, allowed, "search_messages")
	assert.Contains(t, allowed, "group_summary")

	// 大小写变体同样剔除（与工具注册表大小写不敏感查找对齐）
	require.NoError(t, svc.SetScopeTools(ToolScopeGroup, []string{"SEND_MESSAGE", "search_messages"}))
	allowed = g.groupAssistantAllowedTools()
	assert.NotContains(t, allowed, "SEND_MESSAGE")
	assert.NotContains(t, allowed, "send_message")
	assert.Contains(t, allowed, "search_messages")
}
