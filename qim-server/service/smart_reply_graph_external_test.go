package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// buildGateway 构造一个 MCPClientGateway：in-memory config + 手动注册一个外部工具。
func buildGateway(t *testing.T, db *gorm.DB, groupEnabled bool, withTool bool) *MCPClientGateway {
	t.Helper()
	configSvc := NewSystemConfigService(db)
	gw := NewMCPClientGateway(configSvc, ai.NewToolRegistry(nil))

	if groupEnabled {
		setGatewayConfig(t, db, externalMCPGroupEnabledKey, "true")
	}
	if withTool {
		gw.registered["mcp_demo_calculator"] = true
	}
	return gw
}

func TestSmartReplyGraph_HasExternalTools(t *testing.T) {
	db := setupGatewayTestDB(t)

	t.Run("无网关=false", func(t *testing.T) {
		sg := &SmartReplyGraph{}
		assert.False(t, sg.HasExternalTools(), "nil 网关不应放行")
	})

	t.Run("有网关但未开启group=false", func(t *testing.T) {
		sg := &SmartReplyGraph{mcpGateway: buildGateway(t, db, false, true)}
		assert.False(t, sg.HasExternalTools(), "未开 external_mcp:group_enabled 不应放行")
	})

	t.Run("开启但无外部工具=false", func(t *testing.T) {
		sg := &SmartReplyGraph{mcpGateway: buildGateway(t, db, true, false)}
		assert.False(t, sg.HasExternalTools(), "无已注册外部工具不应放行")
	})

	t.Run("开启且有外部工具=true", func(t *testing.T) {
		sg := &SmartReplyGraph{mcpGateway: buildGateway(t, db, true, true)}
		assert.True(t, sg.HasExternalTools(), "开启且已有 mcp_* 工具应放行")
	})
}
