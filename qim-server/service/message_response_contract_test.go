package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// BuildMessageResponse 是 HTTP 响应、WS 广播、AI 消息广播的唯一载荷构建入口。
// 本测试钉住输出字段集（契约快照）：字段增减必须同步修改前端
// qim-client/src/composables/useMainMessageHandlers.ts 的 processMessage。
func TestBuildMessageResponse_FieldContractSnapshot(t *testing.T) {
	msg := model.Message{
		ID:              1,
		ConversationID:  10,
		SenderID:        2,
		Type:            "text",
		Content:         "hello",
		IsRecalled:      false,
		IsRead:          false,
		Origin:          "user",
		CreatedAt:       time.Now(),
		Sender:          model.User{ID: 2, Nickname: "张三", Type: "user"},
		QuotedMessage:   &model.Message{ID: 5, Content: "quoted"},
		QuotedMessageID: uintPtr(5),
	}

	resp := BuildMessageResponse(msg, MessageResponseOptions{BroadcastWS: true})

	// 契约字段集：前端 processMessage 逐一消费，缺失的字段会被 || 默认值静默吞掉。
	// 新增字段必须加在这里 + 前端 processMessage。
	for _, key := range []string{
		"id", "conversation_id", "sender_id", "sender_type", "type", "content",
		"quoted_message_id", "is_recalled", "is_read", "is_avatar_reply", "is_ai_message",
		"is_streaming", "ai_assistant_name", "origin", "recalled_at", "created_at",
		"sender", "quoted_message", "mention_user_ids", "extra",
	} {
		_, ok := resp[key]
		require.Truef(t, ok, "契约字段 %s 缺失——需同步补进 BuildMessageResponse 与前端 processMessage", key)
	}
}

// WS 广播场景：is_read 固定 false（避免 A 已读后 B 收到的广播显示已读），
// 不输出 is_at_mention（前端按 mention_user_ids 自己算）。
func TestBuildMessageResponse_BroadcastWS(t *testing.T) {
	msg := model.Message{ID: 1, SenderID: 2, IsRead: true, Origin: "user", Sender: model.User{ID: 2}}

	resp := BuildMessageResponse(msg, MessageResponseOptions{MentionUserIDs: []uint{3}, BroadcastWS: true})

	assert.False(t, resp["is_read"].(bool), "WS 广播 is_read 应固定 false")
	assert.Equal(t, []uint{3}, resp["mention_user_ids"])
	_, hasAtMention := resp["is_at_mention"]
	assert.False(t, hasAtMention, "WS 广播不应输出 is_at_mention（per-user 字段）")
}

// HTTP 场景：per-user 已读（发送者自己恒已读）+ is_at_mention 计算。
func TestBuildMessageResponse_HTTPPerUser(t *testing.T) {
	msg := model.Message{ID: 1, SenderID: 2, IsRead: false, Origin: "user", Sender: model.User{ID: 2}}

	// 自己发的消息：有回执上下文时恒已读
	resp := BuildMessageResponse(msg, MessageResponseOptions{
		CurrentUserID: 2,
		AllMemberIDs:  []uint{2, 3},
		UserReadSet:   map[uint]bool{},
	})
	assert.True(t, resp["is_read"].(bool), "发送者自己的消息应恒已读")

	// 他人发的消息：按回执判定
	resp2 := BuildMessageResponse(msg, MessageResponseOptions{
		CurrentUserID: 3,
		AllMemberIDs:  []uint{2, 3},
		UserReadSet:   map[uint]bool{},
	})
	assert.False(t, resp2["is_read"].(bool), "无回执时应未读")

	// 无回执上下文（如 SendMessage 响应）：回退 msg.IsRead
	resp3 := BuildMessageResponse(msg, MessageResponseOptions{CurrentUserID: 2, AllMemberIDs: []uint{2, 3}})
	assert.Equal(t, msg.IsRead, resp3["is_read"], "无回执上下文时回退 msg.IsRead")
}

// 分身消息应透出 avatar_name；Extra 解析的 knowledge_sources/sources 应进入顶层。
func TestBuildMessageResponse_AvatarAndExtra(t *testing.T) {
	oldDB := database.GetDB()
	defer func() { database.DB = oldDB }()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.AvatarConfig{UserID: 2, Name: "小助手"}).Error)
	database.DB = db

	msg := model.Message{
		ID:       1,
		SenderID: 2,
		Origin:   "avatar",
		Sender:   model.User{ID: 2, Type: "bot"},
		Extra:    `{"sources":[{"source":"notes","title":"会议纪要","score":0.9}]}`,
	}

	resp := BuildMessageResponse(msg, MessageResponseOptions{BroadcastWS: true})

	assert.True(t, resp["is_avatar_reply"].(bool))
	assert.True(t, resp["is_ai_message"].(bool))
	assert.Equal(t, "小助手", resp["avatar_name"], "分身消息应透出 avatar_name")
	sources, ok := resp["sources"].([]interface{})
	require.True(t, ok, "Extra 中的 sources 应解析到顶层")
	require.Len(t, sources, 1)
}

func uintPtr(v uint) *uint {
	return &v
}
