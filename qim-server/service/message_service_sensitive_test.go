package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSendMessage_SensitiveCheck_MarkdownType 验证 SendMessage 对 markdown 类型消息也执行敏感词检查。
// markdown 类型含敏感词应被拦截，不含敏感词应正常发送。
func TestSendMessage_SensitiveCheck_MarkdownType(t *testing.T) {
	db := setupServiceTestDB(t)

	// 准备：创建用户与会话（参考 TestMessageService_SendMessage 的内联构造方式）
	user1 := &model.User{Username: "u-markdown-1", PasswordHash: "hash", Nickname: "Markdown Sender"}
	user2 := &model.User{Username: "u-markdown-2", PasswordHash: "hash", Nickname: "Markdown Receiver"}
	require.NoError(t, db.Create(user1).Error)
	require.NoError(t, db.Create(user2).Error)

	conv, err := NewConversationService(db).CreateSingleConversation(user1.ID, user2.ID)
	require.NoError(t, err)

	// 直接构造 MessageService，注入敏感词缓存，避免触发 db 查询 sensitive_words 表。
	// sensitiveWordLoaded=true 使 CheckSensitiveContent 直接使用缓存，不调用 loadSensitiveWords。
	svc := &MessageService{
		db:                  db,
		sensitiveWordCache:  []model.SensitiveWord{{Word: "禁用词", Enabled: true}},
		sensitiveWordLoaded: true,
	}

	// markdown 类型消息含敏感词，应被拦截
	_, err = svc.SendMessage(conv.ID, user1.ID, "markdown", "```go\n// 禁用词\nfmt.Println(1)\n```", nil)
	assert.ErrorIs(t, err, ErrSensitiveWordBlocked)

	// markdown 类型消息不含敏感词，正常发送
	msg, err := svc.SendMessage(conv.ID, user1.ID, "markdown", "```go\nfmt.Println(1)\n```", nil)
	require.NoError(t, err)
	assert.Equal(t, "markdown", msg.Type)
	assert.Equal(t, "```go\nfmt.Println(1)\n```", msg.Content)
}
