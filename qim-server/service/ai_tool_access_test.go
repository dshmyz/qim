package service

import (
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
)

func TestMessageRetrieverLimitsGlobalSearchToCallerConversations(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	database.DB = db

	alice := &model.User{Username: "alice", Nickname: "Alice", Type: "user"}
	bob := &model.User{Username: "bob", Nickname: "Bob", Type: "user"}
	requireNoError(t, db.Create(alice).Error)
	requireNoError(t, db.Create(bob).Error)

	aliceConv := &model.Conversation{Type: "group"}
	bobConv := &model.Conversation{Type: "group"}
	requireNoError(t, db.Create(aliceConv).Error)
	requireNoError(t, db.Create(bobConv).Error)
	requireNoError(t, db.Create(&model.ConversationMember{ConversationID: aliceConv.ID, UserID: alice.ID, Role: "member"}).Error)
	requireNoError(t, db.Create(&model.ConversationMember{ConversationID: bobConv.ID, UserID: bob.ID, Role: "member"}).Error)
	requireNoError(t, db.Create(&model.Message{ConversationID: aliceConv.ID, SenderID: alice.ID, Type: "text", Content: "shared needle owned by alice"}).Error)
	requireNoError(t, db.Create(&model.Message{ConversationID: bobConv.ID, SenderID: bob.ID, Type: "text", Content: "shared needle owned by bob"}).Error)

	docs, err := NewMessageRetriever(0, alice.ID, 10).Retrieve(t.Context(), "shared needle")
	requireNoError(t, err)
	if len(docs) != 1 {
		t.Fatalf("want only caller-visible message, got %d docs", len(docs))
	}
	if strings.Contains(docs[0].Content, "bob") {
		t.Fatalf("global search leaked another user's conversation: %q", docs[0].Content)
	}
}

func TestMessageRetrieverCanFilterCallerConversation(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	database.DB = db

	user := &model.User{Username: "member", Nickname: "Member", Type: "user"}
	requireNoError(t, db.Create(user).Error)
	conv := &model.Conversation{Type: "group"}
	requireNoError(t, db.Create(conv).Error)
	requireNoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID, Role: "member"}).Error)
	requireNoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "needle in selected conversation"}).Error)

	docs, err := NewMessageRetriever(conv.ID, user.ID, 10).Retrieve(t.Context(), "needle")
	requireNoError(t, err)
	if len(docs) != 1 {
		t.Fatalf("want one visible message, got %d", len(docs))
	}
}

func TestSummarizeConversationToolRejectsNonMemberConversationOverride(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	database.DB = db

	user := &model.User{Username: "reader", Nickname: "Reader", Type: "user"}
	outsiderConv := &model.Conversation{Type: "group"}
	requireNoError(t, db.Create(user).Error)
	requireNoError(t, db.Create(outsiderConv).Error)

	tool := NewSummarizeConversationTool(&SummaryGraph{})
	_, err := tool.Execute(map[string]interface{}{"conversation_id": "1"}, &ai.CallerContext{
		UserID:         user.ID,
		ConversationID: outsiderConv.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "无权") {
		t.Fatalf("want access denied before summary graph execution, got %v", err)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
