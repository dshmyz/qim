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

// TestSendMessage_OrphanMemberRejected 孤儿成员行（member 行存在但 conversations 无对应行）
// 发送消息被拒（ErrMessageForbidden）。锁定的语义：SendMessage 用 INNER JOIN
// 校验成员 + 取会话类型，孤儿成员行 JOIN 不到 → 拒绝，绝不创建指向不存在会话的悬挂消息。
// 该语义是对历史行为（单独按 member 行查询会放行、在已删除会话上写消息）的防御性改进，
// 用测试锁定以防未来重构意外放开。
func TestSendMessage_OrphanMemberRejected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Conversation{}, &model.ConversationMember{}, &model.Message{},
	))
	database.DB = db

	user := model.User{Username: "orphan-user", Nickname: "孤儿成员"}
	require.NoError(t, db.Create(&user).Error)

	// 只建成员行，不建 conversations 行（会话已删除/残留，conversation_id 指向不存在的会话）
	require.NoError(t, db.Create(&model.ConversationMember{
		ConversationID: 999,
		UserID:         user.ID,
		Role:           "member",
		JoinedAt:       time.Now(),
	}).Error)

	svc := NewMessageService(db, nil, nil)
	_, err = svc.SendMessage(999, user.ID, "text", "hello", nil)
	assert.ErrorIs(t, err, ErrMessageForbidden)
}
