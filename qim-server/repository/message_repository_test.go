package repository

import (
	"context"
	"testing"

	"qim-server/model"

	"github.com/stretchr/testify/assert"
)

func TestMessageRepository_FindByConversationID(t *testing.T) {
	db := setupConvTestDB(t)
	db.AutoMigrate(&model.Message{})
	repo := NewMessageRepository(db)
	ctx := context.Background()

	conv := &model.Conversation{Type: "single"}
	db.Create(conv)
	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	for i := 1; i <= 5; i++ {
		msg := &model.Message{
			ConversationID: conv.ID,
			SenderID:       user.ID,
			Type:           "text",
			Content:        "test message",
		}
		db.Create(msg)
	}

	messages, err := repo.FindByConversationID(ctx, conv.ID, 3, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
}

func TestMessageRepository_RecallMessage(t *testing.T) {
	db := setupConvTestDB(t)
	db.AutoMigrate(&model.Message{})
	repo := NewMessageRepository(db)
	ctx := context.Background()

	conv := &model.Conversation{Type: "single"}
	db.Create(conv)
	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	msg := &model.Message{
		ConversationID: conv.ID,
		SenderID:       user.ID,
		Type:           "text",
		Content:        "test message",
	}
	db.Create(msg)

	err := repo.RecallMessage(ctx, msg.ID)
	assert.NoError(t, err)

	var recalled model.Message
	db.First(&recalled, msg.ID)
	assert.True(t, recalled.IsRecalled)
}

func TestMessageRepository_FindLatestByConversationID(t *testing.T) {
	db := setupConvTestDB(t)
	db.AutoMigrate(&model.Message{})
	repo := NewMessageRepository(db)
	ctx := context.Background()

	conv := &model.Conversation{Type: "single"}
	db.Create(conv)
	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	msg1 := &model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "first"}
	msg2 := &model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "latest"}
	db.Create(msg1)
	db.Create(msg2)

	latest, err := repo.FindLatestByConversationID(ctx, conv.ID)
	assert.NoError(t, err)
	assert.Equal(t, "latest", latest.Content)
}
