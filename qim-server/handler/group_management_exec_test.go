package handler

import (
	"context"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// mockToolProvider 模拟 LLM provider：第 1 次 ChatWithTools 返回 group_management
// tool call，第 2 次（工具执行结果回灌后）返回最终文本。其余方法 no-op。
type mockToolProvider struct {
	toolCalls []ai.ToolCall
	calls     int
}

func (m *mockToolProvider) Name() string { return "mock" }
func (m *mockToolProvider) Chat(messages []ai.Message) (string, error) {
	return "ok", nil
}
func (m *mockToolProvider) ChatStream(messages []ai.Message, onChunk func(ai.StreamChunk) error) error {
	return nil
}
func (m *mockToolProvider) ChatStreamWithContext(ctx context.Context, messages []ai.Message, onChunk func(ai.StreamChunk) error) error {
	return nil
}
func (m *mockToolProvider) Embedding(text string) ([]float32, error) { return nil, nil }
func (m *mockToolProvider) ChatWithTools(messages []ai.Message, tools []ai.ToolDef) (*ai.ChatResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &ai.ChatResponse{Content: "好的，我来移除", ToolCalls: m.toolCalls}, nil
	}
	return &ai.ChatResponse{Content: "已将 victim 移出群聊"}, nil
}
func (m *mockToolProvider) IsConfigured() bool            { return true }
func (m *mockToolProvider) WithModel(model string) ai.Provider { return m }

// TestExecuteWithToolsMockLLM_KicksMember 用 mock provider 验证完整代管链路：
// SmartReplyGraph.ExecuteWithTools -> EinoChatModel.Generate ->
// AIService.GetCompletionWithTools（注入 MCP group_management 工具）->
// LLM 返回 tool call -> mcpServer.ExecuteTool -> GroupManagementTool 真实移除成员。
// 不依赖真实 LLM API。这条测试保护 @AI 代管成员的核心代码路径。
func TestExecuteWithToolsMockLLM_KicksMember(t *testing.T) {
	// 空 config 建 aiService（pool 空），注入 mock provider
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("mock", &mockToolProvider{
		toolCalls: []ai.ToolCall{{
			ID:   "call_1",
			Name: "group_management",
			Arguments: map[string]interface{}{
				"action":           "remove_member",
				"group_identifier": "测试群",
				"user_identifier":  "victim",
			},
		}},
	})
	mcpServer := ai.NewMCPServer(false, aiSvc)
	RegisterAdminTools(mcpServer)
	aiSvc.SetMCPServer(mcpServer)

	// 内存 DB + 种子
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Conversation{}, &model.ConversationMember{}, &model.Group{}))
	database.DB = db

	owner := model.User{Username: "owner", Nickname: "群主"}
	victim := model.User{Username: "victim", Nickname: "被踢者"}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&victim).Error)
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: owner.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "owner", JoinedAt: time.Now()}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: victim.ID, Role: "member", JoinedAt: time.Now()}).Error)
	require.True(t, memberExists(db, conv.ID, victim.ID), "前置：被踢者在群")

	// 用 owner 身份发起（群主权限，工具的 isOwner/admin 校验通过）
	graph := service.NewSmartReplyGraph(aiSvc, db, nil, nil, nil, nil)
	input := &service.SmartReplyContext{
		Message:         "把被踢者移出群",
		OriginalContent: "把被踢者移出群",
		UserID:          owner.ID,
		ConversationID:  conv.ID,
		IsAIMention:     true,
		AssistantName:   "AI助手",
	}

	reply, err := graph.ExecuteWithTools(context.Background(), input)
	require.NoError(t, err)
	assert.NotEmpty(t, reply, "AI 应返回非空回复")
	t.Logf("AI 回复: %q", reply)

	assert.False(t, memberExists(db, conv.ID, victim.ID),
		"被踢者应已被 ExecuteWithTools 经 group_management 工具真实移除")
}

// TestExecuteWithToolsMockLLM_RejectsPlainMember 普通成员身份发起时，
// 工具的群主/管理员校验拒绝执行，被踢者仍在群（callerCtx 用真实 input.UserID）。
func TestExecuteWithToolsMockLLM_RejectsPlainMember(t *testing.T) {
	aiSvc := ai.NewAIService(&ai.AIConfig{})
	aiSvc.SetProviderForTesting("mock", &mockToolProvider{
		toolCalls: []ai.ToolCall{{
			ID:   "call_1",
			Name: "group_management",
			Arguments: map[string]interface{}{
				"action":           "remove_member",
				"group_identifier": "测试群",
				"user_identifier":  "victim",
			},
		}},
	})
	mcpServer := ai.NewMCPServer(false, aiSvc)
	RegisterAdminTools(mcpServer)
	aiSvc.SetMCPServer(mcpServer)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Conversation{}, &model.ConversationMember{}, &model.Group{}))
	database.DB = db

	owner := model.User{Username: "owner", Nickname: "群主"}
	plain := model.User{Username: "plain", Nickname: "普通人"}
	victim := model.User{Username: "victim", Nickname: "被踢者"}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&plain).Error)
	require.NoError(t, db.Create(&victim).Error)
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: owner.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "owner", JoinedAt: time.Now()}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: plain.ID, Role: "member", JoinedAt: time.Now()}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: victim.ID, Role: "member", JoinedAt: time.Now()}).Error)

	// 普通成员发起（input.UserID = plain.ID），工具应拒绝
	graph := service.NewSmartReplyGraph(aiSvc, db, nil, nil, nil, nil)
	input := &service.SmartReplyContext{
		Message:         "把被踢者移出群",
		OriginalContent: "把被踢者移出群",
		UserID:          plain.ID,
		ConversationID:  conv.ID,
		IsAIMention:     true,
		AssistantName:   "AI助手",
	}

	_, err = graph.ExecuteWithTools(context.Background(), input)
	// 工具执行返回权限不足 error，GetCompletionWithTools 会返回该 error
	require.Error(t, err, "普通成员发起时工具应返回权限不足错误")
	assert.Contains(t, err.Error(), "权限不足")
	assert.True(t, memberExists(db, conv.ID, victim.ID), "权限不足时被踢者不应被移除")
}
