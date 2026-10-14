package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupExtractNotification 初始化 GlobalHub：createTodo 给负责人发通知时
// ws.GlobalHub.SendToUser 需要非 nil（与 misc_system_message_version_test 同法）。
func setupExtractNotification(t *testing.T, db *gorm.DB) {
	t.Helper()
	hub := ws.NewHub(db, "test-secret", "http")
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = nil })
}

// extractCaptureProvider 提取器测试用 Provider 桩：Chat 捕获收到的消息并返回固定 JSON。
// WithModel 必须返回自身：带 model 路由时 GetCompletion 走 WithModel().Chat()，
// 返回内嵌裸 mockToolProvider 会丢掉捕获（与 describeCaptureProvider 同一处理）。
type extractCaptureProvider struct {
	mockToolProvider
	reply        string
	lastMessages []ai.Message
}

func (p *extractCaptureProvider) Chat(messages []ai.Message) (string, error) {
	p.lastMessages = messages
	return p.reply, nil
}

func (p *extractCaptureProvider) WithModel(model string) ai.Provider { return p }

// newExtractTestAI 构造打桩 AI：TaskTypeAnalysis 路由到 mock provider，按 reply 返回固定结果。
func newExtractTestAI(t *testing.T, reply string) (*ai.AIService, *extractCaptureProvider) {
	t.Helper()
	aiSvc := ai.NewAIService(&ai.AIConfig{
		Router: ai.RouterConfig{
			DefaultTask: ai.TaskTypeAnalysis,
			Routes: map[ai.TaskType]ai.Route{
				ai.TaskTypeAnalysis: {Provider: "mock", Model: "analysis"},
			},
		},
	})
	capProv := &extractCaptureProvider{reply: reply}
	aiSvc.SetProviderForTesting("mock", capProv)
	return aiSvc, capProv
}

// TestExtractAndCreateTodos_DecodesMentionTokensBeforeAI 问题1取证：
// 消息落库存的是前端写入的 @{mention:id|urlencoded姓名} 原文，直接送 AI 会让
// 「识别 @的人」规则对着百分号乱码工作，负责人识别不出来并静默回退给发送者。
func TestExtractAndCreateTodos_DecodesMentionTokensBeforeAI(t *testing.T) {
	db := setupTodoTestDB(t)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "zhangsan", PasswordHash: "x", Nickname: "张三"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "lisi", PasswordHash: "x", Nickname: "李四"}).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: 1, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: 2, Role: "member"}).Error)

	aiSvc, capProv := newExtractTestAI(t, "[]")
	e := NewTodoExtractor(aiSvc)

	// 前端发送的原文形态：@单人 token，中文名经 URL 编码（client encodeMentionToken）
	e.ExtractAndCreateTodos("@{mention:2|%E6%9D%8E%E5%9B%9B} 明天下午把报告发我", 1, conv.ID)

	require.NotNil(t, capProv.lastMessages, "Provider 应收到调用")
	userMsg := ""
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			userMsg = m.Content
		}
	}
	assert.Contains(t, userMsg, "@李四", "送 AI 前应把 mention token 解码为 @姓名")
	assert.NotContains(t, userMsg, "@{mention:", "不应把原始 token 送进 prompt")
	assert.NotContains(t, userMsg, "%E6%9D%8E", "不应把 URL 编码的姓名送进 prompt")
}

// TestCreateTodo_AssigneeLookupScopedToGroupMembers 问题2取证：
// 负责人查找当前是全库 nickname/username 精确匹配——群外存在同名且主键更小的
// 用户时，First() 会命中群外那个，任务跨群派给无关的人并触发通知。
func TestCreateTodo_AssigneeLookupScopedToGroupMembers(t *testing.T) {
	db := setupTodoTestDB(t)
	setupExtractNotification(t, db)
	// 群外同名用户（ID 更小：当前实现 First() 按主键序会命中它）
	require.NoError(t, db.Create(&model.User{ID: 10, Username: "out", PasswordHash: "x", Nickname: "李四"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 11, Username: "sender", PasswordHash: "x", Nickname: "张三"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 12, Username: "in", PasswordHash: "x", Nickname: "李四"}).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	for _, uid := range []uint{11, 12} {
		require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: uid, Role: "member"}).Error)
	}

	e := NewTodoExtractor(nil)
	e.createTodo(ExtractedTodo{Title: "交季度报告", Assignee: "李四"}, 11, conv.ID)

	var tasks []model.Task
	require.NoError(t, db.Find(&tasks).Error)
	require.Len(t, tasks, 1, "应只创建一条待办")
	assert.Equal(t, uint(12), tasks[0].UserID,
		"负责人应落在群内同名成员身上，而不是主键更小的群外同名用户")
}

// TestCreateTodo_AllExcludesBotMembers 问题3取证：assignee=all 遍历会话成员时
// 未排除 bot 虚拟用户（拉 bot 进群写 ConversationMember，已删 bot 的虚拟用户也残留），
// 会生成没有任何人能看到的幽灵任务。
func TestCreateTodo_AllExcludesBotMembers(t *testing.T) {
	db := setupTodoTestDB(t)
	setupExtractNotification(t, db)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "sender", PasswordHash: "x", Nickname: "张三"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "member", PasswordHash: "x", Nickname: "王五"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 3, Username: "bot_v", PasswordHash: "x", Nickname: "助手机器人", Type: "bot"}).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	for _, uid := range []uint{1, 2, 3} {
		require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: uid, Role: "member"}).Error)
	}

	e := NewTodoExtractor(nil)
	e.createTodo(ExtractedTodo{Title: "全员周报", Assignee: "all"}, 1, conv.ID)

	var tasks []model.Task
	require.NoError(t, db.Find(&tasks).Error)
	require.Len(t, tasks, 1, "assignee=all 应只给人类成员创建，bot 虚拟用户不该有任务")
	require.NotEmpty(t, tasks)
	assert.Equal(t, uint(2), tasks[0].UserID, "任务应落在人类成员身上")
}
