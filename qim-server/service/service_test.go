package service

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	err = db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
		&model.Group{},
		&model.Message{},
		&model.MessageReadReceipt{},
		&model.Notification{},
		&model.Bot{},
		&model.BotConversation{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestUserService_GetUser(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}
	db.Create(user)

	found, err := svc.GetUser(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", found.Username)

	_, err = svc.GetUser(99999)
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_GetUserByUsername(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}
	db.Create(user)

	found, err := svc.GetUserByUsername("testuser")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)

	_, err = svc.GetUserByUsername("notexist")
	assert.Error(t, err)
}

func TestUserService_SearchUsers(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	users := []*model.User{
		{Username: "zhangsan", Nickname: "张三", PasswordHash: "hash"},
		{Username: "lisi", Nickname: "李四", PasswordHash: "hash"},
		{Username: "wangwu", Nickname: "王五", PasswordHash: "hash"},
	}
	for _, u := range users {
		db.Create(u)
	}

	results, err := svc.SearchUsers("张", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "张三", results[0].Nickname)
}

func TestUserService_GetDefaultAIAssistantCreatesBotUser(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user, err := svc.GetDefaultAIAssistant()

	require.NoError(t, err)
	assert.Equal(t, "bot", user.Type)
	assert.Equal(t, "bot_ai_assistant", user.Username)

	var bot model.Bot
	require.NoError(t, db.Where("virtual_user_id = ?", user.ID).First(&bot).Error)
	assert.Equal(t, "assistant", bot.Type)
}

func TestUserService_EnsureGroupAIAssistantCreatesBotUser(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user, err := svc.EnsureGroupAIAssistant(42, "群助手")

	require.NoError(t, err)
	assert.Equal(t, "bot", user.Type)
	assert.Equal(t, "bot_group_assistant_42", user.Username)

	var bot model.Bot
	require.NoError(t, db.Where("virtual_user_id = ?", user.ID).First(&bot).Error)
	assert.Equal(t, "assistant", bot.Type)
}

func TestConversationService_SearchGroupsByName_QuotesGroupsTable(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)
	assert.Equal(t, "JOIN `groups` ON `groups`.conversation_id = conversations.id", groupSearchJoinClause())

	user := &model.User{Username: "searcher", PasswordHash: "hash", Nickname: "搜索者"}
	require.NoError(t, db.Create(user).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.Group{
		ConversationID: conv.ID,
		GroupType:      "group",
		Name:           "项目组",
		CreatorID:      user.ID,
	}).Error)

	results, err := svc.SearchGroupsByName("项目", user.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, conv.ID, results[0].ConversationID)
}

func TestUserService_UpdateUserStatus(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
		Status:       "online",
	}
	db.Create(user)

	err := svc.UpdateUserStatus(user.ID, "offline")
	assert.NoError(t, err)

	updated, err := svc.GetUser(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "offline", updated.Status)
}

func TestUserService_UpdateUser(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Old Name",
	}
	db.Create(user)

	updated, err := svc.UpdateUser(user.ID, map[string]interface{}{
		"nickname": "New Name",
	})
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updated.Nickname)
}

func TestUserService_IsUsernameExists(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}
	db.Create(user)

	exists, err := svc.IsUsernameExists("testuser")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = svc.IsUsernameExists("notexist")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestConversationService_CreateSingleConversation(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	conv, err := svc.CreateSingleConversation(user1.ID, user2.ID)
	assert.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, "single", conv.Type)

	isMember, err := svc.IsConversationMember(conv.ID, user1.ID)
	assert.NoError(t, err)
	assert.True(t, isMember)

	isMember, err = svc.IsConversationMember(conv.ID, user2.ID)
	assert.NoError(t, err)
	assert.True(t, isMember)
}

func TestConversationService_CreateSingleConversation_Duplicate(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	conv1, err := svc.CreateSingleConversation(user1.ID, user2.ID)
	assert.NoError(t, err)

	conv2, err := svc.CreateSingleConversation(user1.ID, user2.ID)
	assert.NoError(t, err)
	assert.Equal(t, conv1.ID, conv2.ID)
}

func TestConversationService_GetConversation(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	conv, _ := svc.CreateSingleConversation(user1.ID, user2.ID)

	found, err := svc.GetConversation(conv.ID)
	assert.NoError(t, err)
	assert.Equal(t, conv.ID, found.ID)

	_, err = svc.GetConversation(99999)
	assert.Error(t, err)
}

func TestConversationService_GetConversationWithAccessCheck(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	user3 := &model.User{Username: "user3", PasswordHash: "hash", Nickname: "User 3"}
	db.Create(user1)
	db.Create(user2)
	db.Create(user3)

	conv, _ := svc.CreateSingleConversation(user1.ID, user2.ID)

	found, err := svc.GetConversationWithAccessCheck(conv.ID, user1.ID)
	assert.NoError(t, err)
	assert.Equal(t, conv.ID, found.ID)

	_, err = svc.GetConversationWithAccessCheck(conv.ID, user3.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrConversationForbidden, err)
}

func TestConversationService_SetConversationMute(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	conv, _ := svc.CreateSingleConversation(user1.ID, user2.ID)

	member, err := svc.SetConversationMute(conv.ID, user1.ID, true)
	assert.NoError(t, err)
	assert.True(t, member.Muted)

	member, err = svc.SetConversationMute(conv.ID, user1.ID, false)
	assert.NoError(t, err)
	assert.False(t, member.Muted)
}

func TestConversationService_GetConversationMembers(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	conv, _ := svc.CreateSingleConversation(user1.ID, user2.ID)

	members, err := svc.GetConversationMembers(conv.ID)
	assert.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestConversationService_UpdateMemberRole(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	conv, _ := svc.CreateSingleConversation(user1.ID, user2.ID)

	err := svc.UpdateMemberRole(conv.ID, user2.ID, "admin")
	assert.NoError(t, err)

	var member model.ConversationMember
	db.Where("conversation_id = ? AND user_id = ?", conv.ID, user2.ID).First(&member)
	assert.Equal(t, "admin", member.Role)
}

func TestConversationService_CreateGroupConversation(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	user3 := &model.User{Username: "user3", PasswordHash: "hash", Nickname: "User 3"}
	db.Create(user1)
	db.Create(user2)
	db.Create(user3)

	conv, err := svc.CreateGroupConversation("Test Group", user1.ID, []uint{user2.ID, user3.ID}, "")
	assert.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, "group", conv.Type)

	var count int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", conv.ID).Count(&count)
	assert.Equal(t, int64(3), count)
}

func TestConversationService_UpdateConversation(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewConversationService(db)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	conv, _ := svc.CreateSingleConversation(user1.ID, user2.ID)

	err := svc.UpdateConversation(conv.ID, map[string]interface{}{
		"last_message_at": "2024-01-01 00:00:00",
	})
	assert.NoError(t, err)

	err = svc.UpdateConversation(99999, map[string]interface{}{})
	assert.Error(t, err)
}

func TestUserService_GetUserRoles(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewUserService(db)

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}
	db.Create(user)

	db.Create(&model.UserRole{UserID: user.ID, Role: "system_admin"})
	db.Create(&model.UserRole{UserID: user.ID, Role: "system_publisher"})

	roles, err := svc.GetUserRoles(user.ID)
	assert.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.Contains(t, roles, "system_admin")
	assert.Contains(t, roles, "system_publisher")

	roles, err = svc.GetUserRoles(99999)
	assert.NoError(t, err)
	assert.Len(t, roles, 0)
}

func TestMessageService_SendMessage(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	msg, err := svc.SendMessage(conv.ID, user1.ID, "text", "Hello World", nil)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, "text", msg.Type)
	assert.Equal(t, "Hello World", msg.Content)
	assert.Equal(t, user1.ID, msg.SenderID)

	var updatedConv model.Conversation
	db.First(&updatedConv, conv.ID)
	assert.NotNil(t, updatedConv.LastMessageID)
	assert.Equal(t, msg.ID, *updatedConv.LastMessageID)
}

func TestMessageService_SendMessageTriggersHubCallbackOnce(t *testing.T) {
	db := setupServiceTestDB(t)
	hub := ws.NewHub(db, "test-secret")
	callbackCalls := make(chan []uint, 2)
	hub.OnMessageSent = func(_ uint, _ uint, _ string, mentionUserIDs []uint) {
		callbackCalls <- mentionUserIDs
	}
	svc := NewMessageService(db, hub, nil)

	sender := &model.User{Username: "callback-sender", PasswordHash: "hash", Nickname: "Sender"}
	recipient := &model.User{Username: "callback-recipient", PasswordHash: "hash", Nickname: "Recipient"}
	db.Create(sender)
	db.Create(recipient)
	conv, err := NewConversationService(db).CreateSingleConversation(sender.ID, recipient.ID)
	assert.NoError(t, err)

	_, err = svc.SendMessage(conv.ID, sender.ID, "text", "@{mention:"+fmt.Sprint(recipient.ID)+"|Recipient}", nil)
	assert.NoError(t, err)

	select {
	case mentionUserIDs := <-callbackCalls:
		assert.Equal(t, []uint{recipient.ID}, mentionUserIDs)
	case <-time.After(time.Second):
		t.Fatal("expected OnMessageSent callback after a successful message")
	}

	select {
	case <-callbackCalls:
		t.Fatal("expected exactly one OnMessageSent callback")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestMessageService_SendMessageToBotPublishesReplyAndUpdatesConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupServiceTestDB(t)
	hub := ws.NewHub(db, "test-secret")
	go hub.Run()
	svc := NewMessageService(db, hub, ai.NewAIService(&ai.AIConfig{}))

	user := &model.User{Username: "bot-user", PasswordHash: "hash", Nickname: "Bot User"}
	virtualUser := &model.User{Username: "virtual-bot", PasswordHash: "hash", Nickname: "Virtual Bot", Type: "bot"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(virtualUser).Error)

	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: virtualUser.ID}).Error)

	bot := &model.Bot{Name: "Helper", Type: "assistant", IsActive: true, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(bot).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, UserID: user.ID, ConversationID: conv.ID}).Error)

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		ws.ServeWs(hub, c)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	_, err = svc.SendMessage(conv.ID, user.ID, "text", "hello bot", nil)
	require.NoError(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	var incoming ws.WSMessage
	require.NoError(t, conn.ReadJSON(&incoming))
	require.Equal(t, "new_message", incoming.Type)

	dataBytes, err := json.Marshal(incoming.Data)
	require.NoError(t, err)
	var data struct {
		ID             uint       `json:"id"`
		ConversationID uint       `json:"conversation_id"`
		SenderID       uint       `json:"sender_id"`
		Type           string     `json:"type"`
		Content        string     `json:"content"`
		Origin         string     `json:"origin"`
		IsAIMessage    bool       `json:"is_ai_message"`
		Sender         model.User `json:"sender"`
	}
	require.NoError(t, json.Unmarshal(dataBytes, &data))
	require.Equal(t, conv.ID, data.ConversationID)
	require.Equal(t, virtualUser.ID, data.SenderID)
	require.Equal(t, "markdown", data.Type)
	require.Equal(t, "assistant", data.Origin)
	require.True(t, data.IsAIMessage)
	require.Equal(t, virtualUser.ID, data.Sender.ID)
	require.Equal(t, "Virtual Bot", data.Sender.Nickname)
	require.NotEmpty(t, data.Content)

	var updatedConv model.Conversation
	require.NoError(t, db.First(&updatedConv, conv.ID).Error)
	require.NotNil(t, updatedConv.LastMessageID)
	require.Equal(t, data.ID, *updatedConv.LastMessageID)
	require.NotNil(t, updatedConv.LastMessageAt)
}

func TestMessageService_SendMessage_NotMember(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	user3 := &model.User{Username: "user3", PasswordHash: "hash", Nickname: "User 3"}
	db.Create(user1)
	db.Create(user2)
	db.Create(user3)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	_, err := svc.SendMessage(conv.ID, user3.ID, "text", "Hello", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrMessageForbidden, err)
}

func TestMessageService_SendMessage_UsesOnlyStructuredMentionTokens(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	sender := &model.User{Username: "mention-sender", PasswordHash: "hash", Nickname: "Sender"}
	mentionedMember := &model.User{Username: "mention-member", PasswordHash: "hash", Nickname: "Member"}
	nonMember := &model.User{Username: "mention-outsider", PasswordHash: "hash", Nickname: "Outsider"}
	db.Create(sender)
	db.Create(mentionedMember)
	db.Create(nonMember)

	conv, err := NewConversationService(db).CreateSingleConversation(sender.ID, mentionedMember.ID)
	assert.NoError(t, err)

	message, err := svc.SendMessage(conv.ID, sender.ID, "text", "@{mention:"+fmt.Sprint(mentionedMember.ID)+"|Member} 请看", nil)
	assert.NoError(t, err)
	assert.Equal(t, "@{mention:"+fmt.Sprint(mentionedMember.ID)+"|Member} 请看", message.Content)

	_, err = svc.SendMessage(conv.ID, sender.ID, "text", "@Member 请看", nil)
	assert.NoError(t, err, "legacy mention_user_ids must not grant a mention")

	_, err = svc.SendMessage(conv.ID, sender.ID, "text", "@{mention:"+fmt.Sprint(nonMember.ID)+"|Outsider}", nil)
	assert.NoError(t, err)

	_, err = svc.SendMessage(conv.ID, sender.ID, "text", "@{mention:all}", nil)
	assert.ErrorIs(t, err, ErrAtAllForbidden)
}

func TestMessageService_GetMessages(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	for i := 0; i < 5; i++ {
		svc.SendMessage(conv.ID, user1.ID, "text", "Message "+string(rune('A'+i)), nil)
	}

	result, err := svc.GetMessages(MessageQuery{
		ConvID: conv.ID,
		UserID: user1.ID,
		Limit:  10,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(5), result.Total)
	assert.Len(t, result.Messages, 5)
}

func TestMessageService_RecallMessage(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	msg, _ := svc.SendMessage(conv.ID, user1.ID, "text", "Hello", nil)

	recalled, err := svc.RecallMessage(msg.ID, user1.ID)
	assert.NoError(t, err)
	assert.True(t, recalled.IsRecalled)
	assert.Equal(t, "[消息已撤回]", recalled.Content)
}

func TestMessageService_RecallMessage_NotOwner(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	msg, _ := svc.SendMessage(conv.ID, user1.ID, "text", "Hello", nil)

	_, err := svc.RecallMessage(msg.ID, user2.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrMessageForbidden, err)
}

func TestMessageService_DeleteMessage(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	msg, _ := svc.SendMessage(conv.ID, user1.ID, "text", "Hello", nil)

	err := svc.DeleteMessage(msg.ID, user1.ID)
	assert.NoError(t, err)

	_, err = svc.GetMessageByID(msg.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrMessageNotFound, err)
}

func TestMessageService_MarkAsRead(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	svc.SendMessage(conv.ID, user2.ID, "text", "Hello from user2", nil)

	err := svc.MarkAsRead(conv.ID, user1.ID)
	assert.NoError(t, err)

	var member model.ConversationMember
	db.Where("conversation_id = ? AND user_id = ?", conv.ID, user1.ID).First(&member)
	assert.Equal(t, 0, member.UnreadCount)
}

func TestMessageService_GetMessageByID(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewMessageService(db, nil, nil)

	user1 := &model.User{Username: "user1", PasswordHash: "hash", Nickname: "User 1"}
	user2 := &model.User{Username: "user2", PasswordHash: "hash", Nickname: "User 2"}
	db.Create(user1)
	db.Create(user2)

	convSvc := NewConversationService(db)
	conv, _ := convSvc.CreateSingleConversation(user1.ID, user2.ID)

	msg, _ := svc.SendMessage(conv.ID, user1.ID, "text", "Hello", nil)

	found, err := svc.GetMessageByID(msg.ID)
	assert.NoError(t, err)
	assert.Equal(t, msg.ID, found.ID)
	assert.Equal(t, "Hello", found.Content)
}

func TestNotificationService_GetNotifications(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewNotificationService(db)

	user := &model.User{Username: "testuser", PasswordHash: "hash", Nickname: "Test User"}
	db.Create(user)

	notifications, total, err := svc.GetNotifications(user.ID, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, notifications, 0)
}

func TestNotificationService_Create(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewNotificationService(db)

	user := &model.User{Username: "testuser", PasswordHash: "hash", Nickname: "Test User"}
	db.Create(user)

	err := svc.Create(&model.Notification{
		UserID:  user.ID,
		Type:    "system",
		Title:   "Test notification",
		Content: "This is a test",
	})
	assert.NoError(t, err)

	notifications, total, _ := svc.GetNotifications(user.ID, 1, 10)
	assert.Equal(t, int64(1), total)
	assert.Len(t, notifications, 1)
	assert.Equal(t, "Test notification", notifications[0].Title)
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewNotificationService(db)

	user := &model.User{Username: "testuser", PasswordHash: "hash", Nickname: "Test User"}
	db.Create(user)

	svc.Create(&model.Notification{
		UserID:  user.ID,
		Type:    "system",
		Title:   "Test",
		Content: "Content",
	})

	notifications, _, _ := svc.GetNotifications(user.ID, 1, 10)
	notifID := notifications[0].ID

	updated, err := svc.MarkAsRead(user.ID, notifID)
	assert.NoError(t, err)
	assert.True(t, updated.Read)
}

func TestNotificationService_MarkAllAsRead(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewNotificationService(db)

	user := &model.User{Username: "testuser", PasswordHash: "hash", Nickname: "Test User"}
	db.Create(user)

	svc.Create(&model.Notification{
		UserID:  user.ID,
		Type:    "system",
		Title:   "Test 1",
		Content: "Content 1",
	})
	svc.Create(&model.Notification{
		UserID:  user.ID,
		Type:    "system",
		Title:   "Test 2",
		Content: "Content 2",
	})

	err := svc.MarkAllAsRead(user.ID)
	assert.NoError(t, err)

	notifications, _, _ := svc.GetNotifications(user.ID, 1, 10)
	for _, n := range notifications {
		assert.True(t, n.Read)
	}
}
