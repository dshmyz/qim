package service

import (
	"encoding/json"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeNoteSearcher 测试用 NoteSearcher 替身，记录调用参数并可控返回。
type fakeNoteSearcher struct {
	called       bool
	calledUserID uint
	calledQuery  string
	calledTopK   int
	results      []SearchResult
	err          error
}

func (f *fakeNoteSearcher) SearchNotes(userID uint, query string, topK int) ([]SearchResult, error) {
	f.called = true
	f.calledUserID = userID
	f.calledQuery = query
	f.calledTopK = topK
	return f.results, f.err
}

// TestBotConfig_UseCreatorNotes_ParsesFromJSON 验证 BotConfig 能从 JSON 解析 use_creator_notes 字段。
func TestBotConfig_UseCreatorNotes_ParsesFromJSON(t *testing.T) {
	cfg := ParseBotConfig(`{"mode":"internal_ai","use_creator_notes":true}`)
	assert.True(t, cfg.UseCreatorNotes)

	cfg2 := ParseBotConfig(`{"mode":"internal_ai"}`)
	assert.False(t, cfg2.UseCreatorNotes, "未设置时默认 false")
}

// TestHandleBotMessage_UseCreatorNotesFalse_DoesNotCallSearchNotes 开关关闭时不调用 SearchNotes。
func TestHandleBotMessage_UseCreatorNotesFalse_DoesNotCallSearchNotes(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_a", Nickname: "BotA", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_a", Nickname: "UA", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name: "BotA", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: human.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":false}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi", Origin: "user"})

	fake := &fakeNoteSearcher{}
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.SetNoteSearcher(fake)
	msgSvc.handleBotMessage(human.ID, conv.ID, "text", "hi")

	assert.False(t, fake.called, "开关关闭时不应调用 SearchNotes")
}

// TestHandleBotMessage_UseCreatorNotesTrue_CallsSearchNotesWithCreatorID
// 开关开启时必须以 bot.CreatorID 为 scope 调用 SearchNotes。
func TestHandleBotMessage_UseCreatorNotesTrue_CallsSearchNotesWithCreatorID(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_b", Nickname: "BotB", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_b", Nickname: "UB", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name: "BotB", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: human.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":true}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "帮我查一下昨天的会议纪要", Origin: "user"})

	fake := &fakeNoteSearcher{
		results: []SearchResult{{Content: "昨日会议：项目进度同步", Score: 0.9}},
	}
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.SetNoteSearcher(fake)
	msgSvc.handleBotMessage(human.ID, conv.ID, "text", "帮我查一下昨天的会议纪要")

	assert.True(t, fake.called, "开关开启时应调用 SearchNotes")
	assert.Equal(t, human.ID, fake.calledUserID, "scope 必须是创建者 ID，防止越权读他人笔记")
	assert.Equal(t, "帮我查一下昨天的会议纪要", fake.calledQuery)
	assert.Equal(t, 3, fake.calledTopK)
}

// TestHandleBotMessage_UseCreatorNotesTrue_NonCreator_DoesNotInjectNotes
// 隐私修复：他人（非创建者）和 bot 对话时，即使 UseCreatorNotes 开关开启，也不应注入创建者笔记，
// 避免泄漏创建者私有数据。创建者笔记仅对创建者本人与自己的 bot 对话可见。
func TestHandleBotMessage_UseCreatorNotesTrue_NonCreator_DoesNotInjectNotes(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_priv", Nickname: "BotPriv", Type: "bot"}
	db.Create(vUser)
	creator := &model.User{Username: "creator", Nickname: "Creator", Type: "user"}
	db.Create(creator)
	other := &model.User{Username: "other", Nickname: "Other", Type: "user"}
	db.Create(other)

	bot := &model.Bot{
		Name: "BotPriv", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: creator.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":true}`,
	}
	db.Create(bot)

	// other（非创建者）与 bot 建立会话
	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, other.ID)
	assert.NoError(t, err)
	db.Create(&model.Message{ConversationID: conv.ID, SenderID: other.ID, Type: "text", Content: "查一下笔记", Origin: "user"})

	fake := &fakeNoteSearcher{
		results: []SearchResult{{Content: "创建者的私密笔记", Score: 0.95}},
	}
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.SetNoteSearcher(fake)
	// other（非创建者）和 bot 对话
	msgSvc.handleBotMessage(other.ID, conv.ID, "text", "查一下笔记")

	assert.False(t, fake.called, "非创建者和 bot 对话时不应注入创建者笔记（隐私）")
}

// TestHandleBotMessage_NoteSearcherNil_NoPanic 没有向量库时静默降级，不 panic。
func TestHandleBotMessage_NoteSearcherNil_NoPanic(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_c", Nickname: "BotC", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_c", Nickname: "UC", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name: "BotC", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: human.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":true}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi", Origin: "user"})

	// noteSearcher=nil：开关虽开，但应静默降级（不调用、不 panic、仍能写一条回复消息）
	msgSvc := NewMessageService(db, nil, nil)
	// 故意不 SetNoteSearcher，保留 nil

	assert.NotPanics(t, func() {
		msgSvc.handleBotMessage(human.ID, conv.ID, "text", "hi")
	})

	// 仍应写入 bot 回复
	var replyCount int64
	db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).Count(&replyCount)
	assert.Equal(t, int64(1), replyCount, "降级时仍应写入兜底回复")
}

// TestHandleBotMessage_SearchNotesError_StillReplies 检索失败时静默降级，仍能回复。
func TestHandleBotMessage_SearchNotesError_StillReplies(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_d", Nickname: "BotD", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_d", Nickname: "UD", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name: "BotD", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: human.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":true}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi", Origin: "user"})

	fake := &fakeNoteSearcher{err: assert.AnError}
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.SetNoteSearcher(fake)

	assert.NotPanics(t, func() {
		msgSvc.handleBotMessage(human.ID, conv.ID, "text", "hi")
	})

	// 检索失败仍应写入兜底回复
	var replyCount int64
	db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).Count(&replyCount)
	assert.Equal(t, int64(1), replyCount, "检索失败时应降级为兜底回复")
}

// TestHandleBotMessage_SearchNotesResults_InjectedAsSystemContext
// 命中结果时，应在 aiMessages 头部插入 system message（笔记片段作为上下文）。
// 这里只验证：调用 SearchNotes 且未 panic + 写入回复（完整注入内容验证留给集成测试）。
func TestHandleBotMessage_SearchNotesResults_InjectedAsSystemContext(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_e", Nickname: "BotE", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_e", Nickname: "UE", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name: "BotE", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: human.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":true}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "讲一下 Q3 规划", Origin: "user"})

	fake := &fakeNoteSearcher{
		results: []SearchResult{
			{Content: "Q3 规划：完成知识库重构", Score: 0.92, Metadata: map[string]string{"title": "Q3 规划"}},
			{Content: "Q3 计划上线 MCP 接入", Score: 0.85, Metadata: map[string]string{"title": "Q3 计划"}},
		},
	}
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.SetNoteSearcher(fake)

	assert.NotPanics(t, func() {
		msgSvc.handleBotMessage(human.ID, conv.ID, "text", "讲一下 Q3 规划")
	})

	assert.True(t, fake.called)
	assert.Equal(t, 3, fake.calledTopK)
	// 命中结果时应写入回复
	var replyCount int64
	db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).Count(&replyCount)
	assert.Equal(t, int64(1), replyCount)
}

// TestHandleBotMessage_HitNotes_WritesKnowledgeSourcesToExtra
// 命中笔记时，Bot 回复消息的 Extra 字段应包含 knowledge_sources（标题/分数），
// 供前端折叠「知识来源」标签渲染。
func TestHandleBotMessage_HitNotes_WritesKnowledgeSourcesToExtra(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_f", Nickname: "BotF", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_f", Nickname: "UF", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name: "BotF", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: human.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":true}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "Q3 规划", Origin: "user"})

	fake := &fakeNoteSearcher{
		results: []SearchResult{
			{Content: "Q3 规划正文", Score: 0.92, Metadata: map[string]string{"title": "Q3 规划"}},
		},
	}
	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.SetNoteSearcher(fake)
	msgSvc.handleBotMessage(human.ID, conv.ID, "text", "Q3 规划")

	// 拉取 bot 回复，校验 Extra 落库
	var reply model.Message
	require.NoError(t, db.Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).First(&reply).Error)
	assert.NotEmpty(t, reply.Extra, "命中笔记时 Extra 不应为空")

	var extra map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(reply.Extra), &extra))
	ks, ok := extra["knowledge_sources"].([]interface{})
	require.True(t, ok, "knowledge_sources 应为数组")
	require.Len(t, ks, 1)
	first := ks[0].(map[string]interface{})
	assert.Equal(t, "Q3 规划", first["title"])
	assert.InDelta(t, 0.92, first["score"], 1e-9)
}

// TestHandleBotMessage_NoHit_ExtraEmpty 未命中笔记时 Extra 应保持空，
// 前端据此跳过「知识来源」标签渲染。
func TestHandleBotMessage_NoHit_ExtraEmpty(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	vUser := &model.User{Username: "bot_g", Nickname: "BotG", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "u_g", Nickname: "UG", Type: "user"}
	db.Create(human)

	bot := &model.Bot{
		Name: "BotG", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &vUser.ID, CreatorID: human.ID,
		Config: `{"mode":"internal_ai","use_creator_notes":false}`,
	}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi", Origin: "user"})

	msgSvc := NewMessageService(db, nil, nil)
	msgSvc.handleBotMessage(human.ID, conv.ID, "text", "hi")

	var reply model.Message
	require.NoError(t, db.Where("conversation_id = ? AND sender_id = ?", conv.ID, vUser.ID).First(&reply).Error)
	assert.Empty(t, reply.Extra, "未开启知识库时 Extra 应为空")
}

// TestBuildMessageResponse_PopulatesKnowledgeSourcesFromExtra
// buildMessageResponse 应把 Extra 中的 knowledge_sources 解析并放入响应体顶层，
// 供前端直接消费（无需再解析 Extra 字符串）。
func TestBuildMessageResponse_PopulatesKnowledgeSourcesFromExtra(t *testing.T) {
	msgSvc := NewMessageService(nil, nil, nil)

	t.Run("Extra 包含 knowledge_sources", func(t *testing.T) {
		msg := model.Message{
			Extra: `{"knowledge_sources":[{"title":"Q3 规划","score":0.92}]}`,
		}
		resp := msgSvc.buildMessageResponse(msg, nil)
		ks, ok := resp["knowledge_sources"].([]interface{})
		assert.True(t, ok)
		require.Len(t, ks, 1)
		first := ks[0].(map[string]interface{})
		assert.Equal(t, "Q3 规划", first["title"])
	})

	t.Run("Extra 为空时不输出 knowledge_sources", func(t *testing.T) {
		msg := model.Message{Extra: ""}
		resp := msgSvc.buildMessageResponse(msg, nil)
		_, ok := resp["knowledge_sources"]
		assert.False(t, ok, "Extra 为空时响应不应包含 knowledge_sources")
	})

	t.Run("Extra 为损坏 JSON 时安全降级", func(t *testing.T) {
		msg := model.Message{Extra: "{not-json"}
		resp := msgSvc.buildMessageResponse(msg, nil)
		_, ok := resp["knowledge_sources"]
		assert.False(t, ok, "损坏 JSON 时不应 panic，也不应输出 knowledge_sources")
	})
}

// TestBuildMessageResponse_PopulatesAvatarSourcesFromExtra
// buildMessageResponse 应把 Extra 中的 sources（分身命中知识来源）解析并放入响应体顶层，
// 供前端渲染「依据」徽章，且在刷新/REST 回放后仍可见（与广播下发的 sources 一致）。
func TestBuildMessageResponse_PopulatesAvatarSourcesFromExtra(t *testing.T) {
	msgSvc := NewMessageService(nil, nil, nil)

	t.Run("Extra 包含 sources", func(t *testing.T) {
		msg := model.Message{
			Extra: `{"sources":[{"type":"note","title":"会议纪要","snippet":"..."}]}`,
		}
		resp := msgSvc.buildMessageResponse(msg, nil)
		srcs, ok := resp["sources"].([]interface{})
		assert.True(t, ok)
		require.Len(t, srcs, 1)
		first := srcs[0].(map[string]interface{})
		assert.Equal(t, "note", first["type"])
		assert.Equal(t, "会议纪要", first["title"])
	})

	t.Run("Extra 无 sources 时不输出", func(t *testing.T) {
		msg := model.Message{Extra: `{"tool_calls":[]}`}
		resp := msgSvc.buildMessageResponse(msg, nil)
		_, ok := resp["sources"]
		assert.False(t, ok, "Extra 无 sources 时响应不应包含 sources")
	})
}
