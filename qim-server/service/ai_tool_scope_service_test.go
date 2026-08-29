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
