package repository

import (
	"context"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"

	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupConvTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestConversationRepository_FindByUserID(t *testing.T) {
	db := setupConvTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)

	conv := &model.Conversation{Type: "single"}
	db.Create(conv)

	db.Create(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         user.ID,
		Role:           "member",
	})

	convs, err := repo.FindByUserID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Len(t, convs, 1)
}

func TestConversationRepository_AddMember(t *testing.T) {
	db := setupConvTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)
	conv := &model.Conversation{Type: "group"}
	db.Create(conv)

	err := repo.AddMember(ctx, conv.ID, user.ID, "member")
	assert.NoError(t, err)

	var member model.ConversationMember
	err = db.Where("conversation_id = ? AND user_id = ?", conv.ID, user.ID).First(&member).Error
	assert.NoError(t, err)
	assert.Equal(t, "member", member.Role)
}

func TestConversationRepository_RemoveMember(t *testing.T) {
	db := setupConvTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", PasswordHash: "hash"}
	db.Create(user)
	conv := &model.Conversation{Type: "group"}
	db.Create(conv)

	repo.AddMember(ctx, conv.ID, user.ID, "member")

	err := repo.RemoveMember(ctx, conv.ID, user.ID)
	assert.NoError(t, err)

	var count int64
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conv.ID, user.ID).
		Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestConversationRepository_FindSingleConversation(t *testing.T) {
	db := setupConvTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user1 := &model.User{Username: "user1", PasswordHash: "hash"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash"}
	db.Create(user1)
	db.Create(user2)

	conv := &model.Conversation{Type: "single"}
	db.Create(conv)

	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user1.ID, Role: "member"})
	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user2.ID, Role: "member"})

	found, err := repo.FindSingleConversation(ctx, user1.ID, user2.ID)
	assert.NoError(t, err)
	assert.Equal(t, conv.ID, found.ID)
}

func TestConversationRepository_FindSingleConversation_SelfChatDoesNotMatchTwoMemberConversation(t *testing.T) {
	db := setupConvTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user1 := &model.User{Username: "user1", PasswordHash: "hash"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash"}
	db.Create(user1)
	db.Create(user2)

	twoMemberConv := &model.Conversation{Type: "single"}
	db.Create(twoMemberConv)
	db.Create(&model.ConversationMember{ConversationID: twoMemberConv.ID, UserID: user1.ID, Role: "member"})
	db.Create(&model.ConversationMember{ConversationID: twoMemberConv.ID, UserID: user2.ID, Role: "member"})

	selfConv := &model.Conversation{Type: "single"}
	db.Create(selfConv)
	db.Create(&model.ConversationMember{ConversationID: selfConv.ID, UserID: user1.ID, Role: "member"})

	found, err := repo.FindSingleConversation(ctx, user1.ID, user1.ID)
	assert.NoError(t, err)
	assert.Equal(t, selfConv.ID, found.ID)
}
