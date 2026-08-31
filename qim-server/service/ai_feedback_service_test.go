package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIFeedbackSetAndRevoke(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AIMessageFeedback{}))

	user := &model.User{Username: "feuer", Nickname: "反馈用户"}
	require.NoError(t, db.Create(user).Error)
	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)
	msg := &model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "markdown", Content: "AI 回答"}
	require.NoError(t, db.Create(msg).Error)

	svc := NewAIFeedbackService(db)

	// 点赞 → 查询回读
	require.NoError(t, svc.SetFeedback(user.ID, msg.ID, 1))
	rating, err := svc.GetFeedback(user.ID, msg.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, rating)

	// 改踩 → upsert 覆盖
	require.NoError(t, svc.SetFeedback(user.ID, msg.ID, -1))
	rating, _ = svc.GetFeedback(user.ID, msg.ID)
	assert.Equal(t, -1, rating)

	var count int64
	db.Model(&model.AIMessageFeedback{}).Count(&count)
	assert.EqualValues(t, 1, count, "改评应覆盖而非新增")

	// 撤销（rating=0）→ 删行
	require.NoError(t, svc.SetFeedback(user.ID, msg.ID, 0))
	rating, _ = svc.GetFeedback(user.ID, msg.ID)
	assert.Equal(t, 0, rating)

	// 非法 rating
	require.Error(t, svc.SetFeedback(user.ID, msg.ID, 5))
}

func TestAIFeedbackMembershipGuard(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AIMessageFeedback{}))

	insider := &model.User{Username: "in"}
	outsider := &model.User{Username: "out"}
	require.NoError(t, db.Create(insider).Error)
	require.NoError(t, db.Create(outsider).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: insider.ID}).Error)
	msg := &model.Message{ConversationID: conv.ID, SenderID: insider.ID, Type: "text", Content: "内部讨论"}
	require.NoError(t, db.Create(msg).Error)

	svc := NewAIFeedbackService(db)
	require.ErrorIs(t, svc.SetFeedback(outsider.ID, msg.ID, 1), ErrFeedbackForbidden, "非会话成员不可反馈")
	require.NoError(t, svc.SetFeedback(insider.ID, msg.ID, 1))
}

func TestAIFeedbackBatch(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AIMessageFeedback{}))

	user := &model.User{Username: "batchu", Nickname: "批量用户"}
	require.NoError(t, db.Create(user).Error)
	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)

	msgs := make([]*model.Message, 3)
	for i := range msgs {
		msgs[i] = &model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "markdown", Content: "回答"}
		require.NoError(t, db.Create(msgs[i]).Error)
	}

	svc := NewAIFeedbackService(db)
	require.NoError(t, svc.SetFeedback(user.ID, msgs[0].ID, 1))
	require.NoError(t, svc.SetFeedback(user.ID, msgs[1].ID, -1))
	// msgs[2] 无反馈

	ratings, err := svc.GetFeedbackBatch(user.ID, []uint{msgs[0].ID, msgs[1].ID, msgs[2].ID})
	require.NoError(t, err)
	assert.Equal(t, 1, ratings[msgs[0].ID])
	assert.Equal(t, -1, ratings[msgs[1].ID])
	_, has := ratings[msgs[2].ID]
	assert.False(t, has, "未反馈的消息不在 map 中")

	// 空入参
	empty, err := svc.GetFeedbackBatch(user.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
