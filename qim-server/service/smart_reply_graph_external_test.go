package service

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// toolPlan 构造「带工具」回合计划：工具集与外部指引都取自 graph 自身，
// 用于验证 preparedHistory 落定的工具边界（提示词声明与实际放行永不背离）。
// externalGuide 为 true 时模拟外部 MCP 已启用的普通提问路径；false 模拟管理指令/auto
// 等不注入外部工具指引的路径。
func toolPlan(g *SmartReplyGraph, externalGuide bool) ReplyTurnPlan {
	return PlanGroupTurn(TurnInput{
		Intent:               &ai.MessageIntent{Type: "query"},
		Question:             "我周三有什么会",
		GroupTools:           g.GroupTools(),
		ExternalToolsEnabled: externalGuide && g.HasExternalTools(),
		VisionDisableTools:   true,
		QualityGateEnabled:   true,
	})
}

// hasExternalToolGuides 判断历史消息里是否含外部工具的触发/输出组织指引。
func hasExternalToolGuides(history []*schema.Message) (trigger, output bool) {
	for _, m := range history {
		if m.Role != schema.System || m.Content == "" {
			continue
		}
		if strings.Contains(m.Content, "你有若干外部工具可调用") {
			trigger = true
		}
		if strings.Contains(m.Content, "当你调用外部工具得到结果后") {
			output = true
		}
	}
	return trigger, output
}

// TestPreparedHistory_UnionToolsAndExternalGuide 回归：后台开启外部 MCP 后，带工具路径的
// 白名单必须是「内置群工具 ∪ 外部 MCP 工具」。修复前普通 @AI 提问走 ToolsetExternal，
// 只拿到 mcp_* 工具，search_knowledge/list_tasks 等内置能力被静默顶掉——开关一开能力反而变少。
// 同时校验外部工具指引只在 input.ExternalToolGuide 置位时追加（管理指令/auto 路径不置位，
// 行为与修复前一致）。
func TestPreparedHistory_UnionToolsAndExternalGuide(t *testing.T) {
	db := setupServiceTestDB(t)
	// 网关的 group_enabled 门控读 system_configs
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	// buildHistoryMessages 读 database.GetDB() 取群历史（不走 g.db），须把全局指到本测试库
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)

	g := &SmartReplyGraph{
		db:         db,
		toolScopes: NewToolScopeService(db),
		mcpGateway: buildGateway(t, db, true, true),
	}
	require.True(t, g.HasExternalTools(), "前置：外部 MCP 已开启且有已注册工具")

	t.Run("白名单=内置∪外部", func(t *testing.T) {
		input := &SmartReplyContext{ConversationID: conv.ID, UserID: 1, Message: "我周三有什么会"}
		_, _, err := g.preparedHistory(input, toolPlan(g, true))
		require.NoError(t, err)

		assert.Contains(t, input.AllowedTools, "search_knowledge", "内置查询工具应保留")
		assert.Contains(t, input.AllowedTools, "list_tasks", "内置查询工具应保留")
		assert.Contains(t, input.AllowedTools, "mcp_demo_calculator", "外部 MCP 工具应并入")
		assert.NotContains(t, input.AllowedTools, "send_message", "群入口仍强制剔除 send_message")
		assert.True(t, input.HasTools)
	})

	t.Run("未置位指引=不追加（管理指令/auto 路径）", func(t *testing.T) {
		input := &SmartReplyContext{ConversationID: conv.ID, UserID: 1, Message: "踢掉张三"}
		history, _, err := g.preparedHistory(input, toolPlan(g, false))
		require.NoError(t, err)

		trigger, output := hasExternalToolGuides(history)
		assert.False(t, trigger, "普通带工具路径不应追加外部工具触发指引")
		assert.False(t, output, "普通带工具路径不应追加外部工具输出指引")
	})

	t.Run("置位指引=追加两条（@AI 普通提问且有外部工具）", func(t *testing.T) {
		input := &SmartReplyContext{
			ConversationID: conv.ID, UserID: 1, Message: "算一下 3.5 × 7",
			ExternalToolGuide: true,
		}
		history, _, err := g.preparedHistory(input, toolPlan(g, true))
		require.NoError(t, err)

		trigger, output := hasExternalToolGuides(history)
		assert.True(t, trigger, "应追加外部工具触发指引")
		assert.True(t, output, "应追加外部工具输出组织指引")
		// 指引只是附加，工具白名单不受影响
		assert.Contains(t, input.AllowedTools, "search_knowledge")
	})
}

// TestBuildSystemPromptCapabilityBoundary 校验「能力与工具」能力边界按 HasTools 分支注入，
// 且【边界与约束】否定边界恒定存在（无工具时也注入，避免模型虚构执行）。
func TestBuildSystemPromptCapabilityBoundary(t *testing.T) {
	sg := &SmartReplyGraph{}
	base := &SmartReplyContext{Message: "hi"}

	t.Run("无工具=诚实说明边界", func(t *testing.T) {
		p := sg.buildSystemPrompt(base)
		assert.Contains(t, p, "【能力与工具】")
		assert.Contains(t, p, "当前未提供实时查询或执行工具", "无工具分支应说明能力边界")
		assert.Contains(t, p, "不要假装你已经执行了某些操作", "无工具分支应禁止虚构执行")
	})

	t.Run("有工具=声明可调用", func(t *testing.T) {
		withTools := &SmartReplyContext{Message: "hi", HasTools: true}
		p := sg.buildSystemPrompt(withTools)
		assert.Contains(t, p, "你可以调用系统提供的工具来获取实时数据或执行操作")
		assert.NotContains(t, p, "当前未提供实时查询或执行工具", "有工具分支不应出现无工具说明")
	})

	t.Run("边界与约束恒定存在", func(t *testing.T) {
		for _, in := range []*SmartReplyContext{base, {Message: "hi", HasTools: true}} {
			p := sg.buildSystemPrompt(in)
			assert.Contains(t, p, "【边界与约束】")
			assert.Contains(t, p, "不编造事实")
			assert.Contains(t, p, "隐私克制")
		}
	})

	t.Run("时间注入恒定存在", func(t *testing.T) {
		p := sg.buildSystemPrompt(base)
		if !strings.Contains(p, "【当前时间】") {
			t.Errorf("system prompt 应携带当前时间: %q", p)
		}
	})

	t.Run("产品知识恒定注入（不再依赖关键词启发式）", func(t *testing.T) {
		p := sg.buildSystemPrompt(base)
		assert.Contains(t, p, "【产品使用参考】", "产品知识应恒注入（改前按 isProductQuestion 关键词门控）")
		assert.Contains(t, p, "以下为系统已知的产品使用说明", "产品知识应以参考性措辞框定")
	})

	t.Run("输出格式规则恒定存在", func(t *testing.T) {
		p := sg.buildSystemPrompt(base)
		assert.Contains(t, p, "直接给出结论")
		assert.Contains(t, p, "自然语气简要标注来源")
	})

	t.Run("主动性分层声明恒定存在", func(t *testing.T) {
		// 群聊=建议层：敢建议、不擅自执行；与侧边栏高主动分工。各分支（自定义/普通、有无工具）都应声明。
		for _, in := range []*SmartReplyContext{base, {Message: "hi", HasTools: true}} {
			p := sg.buildSystemPrompt(in)
			assert.Contains(t, p, "你的回复会对全群成员可见", "群聊应声明回复可见范围")
			assert.Contains(t, p, "未经用户明确要求，不要擅自执行", "群聊应声明不擅自执行操作")
			assert.Contains(t, p, "用户明确提出要求", "用户明确要求时可直接执行，避免误伤合法指令（如建群待办）")
			assert.Contains(t, p, "无需再回头征求确认", "明确要求时应直接执行，不应反复确认导致任务提取畏手畏脚")
		}
	})
}

// capStubTool 简洁能力自述用的桩工具。
type capStubTool struct{ name string }

func (t capStubTool) Name() string                       { return t.name }
func (t capStubTool) Description() string                { return t.name + " 描述" }
func (t capStubTool) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (t capStubTool) Execute(map[string]interface{}, *ai.CallerContext) (interface{}, error) {
	return nil, nil
}

// TestBuildSystemPromptCapabilityInjection 校验能力自述会依 AllowedTools 动态注入真实工具名。
func TestBuildSystemPromptCapabilityInjection(t *testing.T) {
	registry := ai.NewToolRegistry(nil)
	registry.RegisterTool(capStubTool{name: "search_knowledge"})
	registry.RegisterTool(capStubTool{name: "create_user_task"})

	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetToolRegistry(registry)
	sg := &SmartReplyGraph{aiService: aiSvc}

	t.Run("无工具=只注入静态能力，不声明工具", func(t *testing.T) {
		p := sg.buildSystemPrompt(&SmartReplyContext{Message: "hi"})
		assert.Contains(t, p, "你具备以下能力")
		assert.NotContains(t, p, "search_knowledge", "无工具不应声称工具")
		assert.NotContains(t, p, "create_user_task")
	})

	t.Run("有工具=按白名单注入真实工具名", func(t *testing.T) {
		withTools := &SmartReplyContext{Message: "hi", HasTools: true, AllowedTools: []string{"search_knowledge"}}
		p := sg.buildSystemPrompt(withTools)
		assert.Contains(t, p, "你具备以下能力")
		assert.Contains(t, p, "search_knowledge", "白名单内工具应注入")
		assert.NotContains(t, p, "create_user_task", "白名单外工具不应注入")
	})
}
