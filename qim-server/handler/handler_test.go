package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/dshmyz/qim/qim-server/auth"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
		&model.Message{},
		&model.Notification{},
		&model.Bot{},
		&model.BotConversation{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerTestDB(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
			Expire: 3600,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
	}

	database.DB = db
	di.InitContainer(cfg, nil)
	auth.InitAuthChain()
	SetConfig(cfg)

	r := gin.New()
	r.POST("/api/v1/auth/login", Login)
	r.POST("/api/v1/auth/register", Register)

	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		c.Set("roles", []string{})
		c.Next()
	}

	authed := r.Group("/api/v1")
	authed.Use(authMiddleware)
	{
		authed.GET("/conversations", GetConversations)
		authed.GET("/conversations/:id", GetConversation)
		authed.POST("/conversations", CreateConversation)
		authed.POST("/groups/:id/exit", ExitGroup)
		authed.GET("/users/me", GetCurrentUser)
		authed.PUT("/users/me", UpdateUser)
	}

	return r, db
}

func createTestUser(t *testing.T, db *gorm.DB) *model.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &model.User{
		Username:     "testuser",
		PasswordHash: string(hash),
		Nickname:     "Test User",
		Status:       "offline",
	}
	db.Create(user)
	return user
}

func TestConversationMember_CreateSetsJoinedAt(t *testing.T) {
	db := setupHandlerTestDB(t)
	user := createTestUser(t, db)
	conv := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(conv).Error)

	member := &model.ConversationMember{ConversationID: conv.ID, UserID: user.ID, Role: "member"}
	require.NoError(t, db.Create(member).Error)
	assert.False(t, member.JoinedAt.IsZero())
}

func TestGetConversations_OmitsSoftDeletedGroupMembers(t *testing.T) {
	r, db := setupTestRouter(t)
	if !db.Migrator().HasTable(&model.Group{}) {
		require.NoError(t, db.Migrator().CreateTable(&model.Group{}))
	}
	currentUser := createTestUser(t, db)
	activeUser := &model.User{Username: "active", PasswordHash: "hash", Nickname: "有效成员"}
	require.NoError(t, db.Create(activeUser).Error)
	deletedUser := &model.User{Username: "deleted", PasswordHash: "hash", Nickname: "已删成员"}
	require.NoError(t, db.Create(deletedUser).Error)
	require.NoError(t, db.Delete(deletedUser).Error)

	conversation := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conversation).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conversation.ID, Name: "测试群", GroupType: "group", CreatorID: currentUser.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: currentUser.ID, Role: "owner"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: activeUser.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: deletedUser.ID, Role: "member"}).Error)

	req := httptest.NewRequest("GET", "/api/v1/conversations?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID      uint `json:"id"`
				Members []struct {
					UserID uint `json:"user_id"`
					User   struct {
						Nickname string `json:"nickname"`
					} `json:"user"`
				} `json:"members"`
			} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Len(t, response.Data.List, 1)

	memberIDs := make([]uint, 0, len(response.Data.List[0].Members))
	memberNames := make([]string, 0, len(response.Data.List[0].Members))
	for _, member := range response.Data.List[0].Members {
		memberIDs = append(memberIDs, member.UserID)
		memberNames = append(memberNames, member.User.Nickname)
	}
	assert.ElementsMatch(t, []uint{currentUser.ID, activeUser.ID}, memberIDs)
	assert.NotContains(t, memberNames, deletedUser.Nickname)
}

func TestGetConversation_OmitsSoftDeletedGroupMembers(t *testing.T) {
	r, db := setupTestRouter(t)
	if !db.Migrator().HasTable(&model.Group{}) {
		require.NoError(t, db.Migrator().CreateTable(&model.Group{}))
	}
	currentUser := createTestUser(t, db)
	activeUser := &model.User{Username: "active", PasswordHash: "hash", Nickname: "有效成员"}
	require.NoError(t, db.Create(activeUser).Error)
	deletedUser := &model.User{Username: "deleted", PasswordHash: "hash", Nickname: "已删成员"}
	require.NoError(t, db.Create(deletedUser).Error)
	require.NoError(t, db.Delete(deletedUser).Error)

	conversation := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conversation).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conversation.ID, Name: "测试群", GroupType: "group", CreatorID: currentUser.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: currentUser.ID, Role: "owner"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: activeUser.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: deletedUser.ID, Role: "member"}).Error)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/conversations/%d", conversation.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Members []struct {
				UserID uint `json:"user_id"`
				User   struct {
					Nickname string `json:"nickname"`
				} `json:"user"`
			} `json:"members"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)

	memberIDs := make([]uint, 0, len(response.Data.Members))
	memberNames := make([]string, 0, len(response.Data.Members))
	for _, member := range response.Data.Members {
		memberIDs = append(memberIDs, member.UserID)
		memberNames = append(memberNames, member.User.Nickname)
	}
	assert.ElementsMatch(t, []uint{currentUser.ID, activeUser.ID}, memberIDs)
	assert.NotContains(t, memberNames, deletedUser.Nickname)
}

func TestExitGroup_RejectsOwnerWithoutRemovingMembership(t *testing.T) {
	r, db := setupTestRouter(t)
	if !db.Migrator().HasTable(&model.Group{}) {
		require.NoError(t, db.Migrator().CreateTable(&model.Group{}))
	}
	currentUser := createTestUser(t, db)
	conversation := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conversation).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conversation.ID, Name: "测试群", GroupType: "group", CreatorID: currentUser.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: currentUser.ID, Role: "owner"}).Error)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/exit", conversation.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "群主退出前请先转让群主或解散群聊", response.Message)

	var member model.ConversationMember
	err := db.Where("conversation_id = ? AND user_id = ?", conversation.ID, currentUser.ID).First(&member).Error
	require.NoError(t, err)
	assert.Equal(t, "owner", member.Role)
}

func TestGetConversations_OrdersEmptyConversationByCreatedAt(t *testing.T) {
	r, db := setupTestRouter(t)
	user := createTestUser(t, db)

	olderActivity := time.Now().Add(-time.Hour)
	olderConversation := &model.Conversation{
		Type:          "single",
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		LastMessageAt: &olderActivity,
	}
	require.NoError(t, db.Create(olderConversation).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: olderConversation.ID, UserID: user.ID, Role: "member"}).Error)

	newConversation := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(newConversation).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: newConversation.ID, UserID: user.ID, Role: "member"}).Error)

	req := httptest.NewRequest("GET", "/api/v1/conversations?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID uint `json:"id"`
			} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.NotEmpty(t, response.Data.List)
	assert.Equal(t, newConversation.ID, response.Data.List[0].ID)
}

func TestGetConversations_SelfChatUsesCurrentUserAsDisplayMember(t *testing.T) {
	r, db := setupTestRouter(t)
	user := createTestUser(t, db)
	user.Avatar = "/avatars/self.png"
	require.NoError(t, db.Save(user).Error)

	conversation := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(conversation).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: user.ID, Role: "member"}).Error)

	req := httptest.NewRequest("GET", "/api/v1/conversations?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID              uint   `json:"id"`
				Name            string `json:"name"`
				Avatar          string `json:"avatar"`
				OtherMemberID   uint   `json:"other_member_id"`
				OtherMemberName string `json:"other_member_name"`
			} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Len(t, response.Data.List, 1)
	assert.Equal(t, conversation.ID, response.Data.List[0].ID)
	assert.Equal(t, user.Nickname, response.Data.List[0].Name)
	assert.Equal(t, user.Avatar, response.Data.List[0].Avatar)
	assert.Equal(t, user.ID, response.Data.List[0].OtherMemberID)
	assert.Equal(t, user.Nickname, response.Data.List[0].OtherMemberName)
}

func TestCreateSingleConversation_RestoresHiddenConversationWithMembers(t *testing.T) {
	r, db := setupTestRouter(t)
	currentUser := createTestUser(t, db)
	recipient := &model.User{Username: "recipient", PasswordHash: "hash", Nickname: "对方用户"}
	require.NoError(t, db.Create(recipient).Error)

	conversation := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(conversation).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: currentUser.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: recipient.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationSession{UserID: currentUser.ID, ConversationID: conversation.ID, IsHidden: true}).Error)

	body, err := json.Marshal(map[string]any{"type": "single", "user_id": recipient.ID})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			ID      uint `json:"id"`
			Members []struct {
				User struct {
					Nickname string `json:"nickname"`
				} `json:"user"`
			} `json:"members"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	assert.Equal(t, conversation.ID, response.Data.ID)
	require.Len(t, response.Data.Members, 2)
	assert.Contains(t, []string{response.Data.Members[0].User.Nickname, response.Data.Members[1].User.Nickname}, recipient.Nickname)

	var session model.ConversationSession
	require.NoError(t, db.Where("user_id = ? AND conversation_id = ?", currentUser.ID, conversation.ID).First(&session).Error)
	assert.False(t, session.IsHidden)
}

func TestCreateSingleConversation_SelfChatDoesNotReuseConversationWithAnotherUser(t *testing.T) {
	r, db := setupTestRouter(t)
	currentUser := createTestUser(t, db)
	recipient := &model.User{Username: "recipient", PasswordHash: "hash", Nickname: "对方用户"}
	require.NoError(t, db.Create(recipient).Error)

	existingConversation := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(existingConversation).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: existingConversation.ID, UserID: currentUser.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: existingConversation.ID, UserID: recipient.ID, Role: "member"}).Error)

	body, err := json.Marshal(map[string]any{"type": "single", "user_id": currentUser.ID})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			ID      uint `json:"id"`
			Members []struct {
				UserID uint `json:"user_id"`
			} `json:"members"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	assert.NotEqual(t, existingConversation.ID, response.Data.ID)
	require.Len(t, response.Data.Members, 1)
	assert.Equal(t, currentUser.ID, response.Data.Members[0].UserID)
}

func TestCreateBotConversation_IncludesBotVirtualUserMember(t *testing.T) {
	r, db := setupTestRouter(t)
	currentUser := createTestUser(t, db)
	botUser := &model.User{Username: "bot_1", PasswordHash: "hash", Nickname: "青雀一号", Type: "bot"}
	require.NoError(t, db.Create(botUser).Error)
	bot := &model.Bot{Name: "青雀一号", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &botUser.ID}
	require.NoError(t, db.Create(bot).Error)

	body, err := json.Marshal(map[string]any{"type": "single", "user_id": botUser.ID})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			ID      uint   `json:"id"`
			Type    string `json:"type"`
			Members []struct {
				UserID uint `json:"user_id"`
				User   struct {
					Nickname string `json:"nickname"`
					Type     string `json:"type"`
				} `json:"user"`
			} `json:"members"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Equal(t, "bot", response.Data.Type)
	require.Len(t, response.Data.Members, 2)

	memberIDs := []uint{response.Data.Members[0].UserID, response.Data.Members[1].UserID}
	assert.Contains(t, memberIDs, currentUser.ID)
	assert.Contains(t, memberIDs, botUser.ID)
	assert.Contains(t, []string{response.Data.Members[0].User.Nickname, response.Data.Members[1].User.Nickname}, "青雀一号")
}

// TestCreateSingleConversation_HidesConversationForRecipient 验证创建私聊会话时，
// 接收方的会话默认被隐藏，发起方不受影响。
func TestCreateSingleConversation_HidesConversationForRecipient(t *testing.T) {
	r, db := setupTestRouter(t)
	sender := createTestUser(t, db)
	recipient := &model.User{Username: "recipient", PasswordHash: "hash", Nickname: "对方用户"}
	require.NoError(t, db.Create(recipient).Error)

	body, err := json.Marshal(map[string]any{"type": "single", "user_id": recipient.ID})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var response struct {
		Code int `json:"code"`
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	assert.Greater(t, response.Data.ID, uint(0))

	// 发起方不应有隐藏 session（或 session 不存在，COALESCE 默认 false）
	var senderSession model.ConversationSession
	err = db.Where("user_id = ? AND conversation_id = ?", sender.ID, response.Data.ID).First(&senderSession).Error
	if err == nil {
		assert.False(t, senderSession.IsHidden, "发起方的会话不应被隐藏")
	}

	// 接收方应有隐藏 session
	var recipientSession model.ConversationSession
	require.NoError(t, db.Where("user_id = ? AND conversation_id = ?", recipient.ID, response.Data.ID).First(&recipientSession).Error)
	assert.True(t, recipientSession.IsHidden, "接收方的会话应被隐藏")
}

// TestGetConversations_HiddenConversationNotShownForRecipient 验证接收方查询会话列表时，
// 被隐藏的空会话不会出现。
func TestGetConversations_HiddenConversationNotShownForRecipient(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db) // 发起方（user_id=1）
	recipient := &model.User{Username: "recipient", PasswordHash: "hash", Nickname: "对方用户"}
	require.NoError(t, db.Create(recipient).Error)

	// 创建私聊会话
	body, _ := json.Marshal(map[string]any{"type": "single", "user_id": recipient.ID})
	req := httptest.NewRequest("POST", "/api/v1/conversations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	require.Equal(t, 0, createResp.Code)

	// 临时切换 user_id 为接收方来查询会话列表
	r2 := gin.New()
	authMiddleware2 := func(c *gin.Context) {
		c.Set("user_id", recipient.ID)
		c.Set("username", "recipient")
		c.Set("roles", []string{})
		c.Next()
	}
	authed2 := r2.Group("/api/v1")
	authed2.Use(authMiddleware2)
	authed2.GET("/conversations", GetConversations)

	req2 := httptest.NewRequest("GET", "/api/v1/conversations?page=1&page_size=20", nil)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID uint `json:"id"`
			} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &listResp))
	require.Equal(t, 0, listResp.Code)

	// 接收方不应看到被隐藏的会话
	for _, conv := range listResp.Data.List {
		assert.NotEqual(t, createResp.Data.ID, conv.ID, "接收方不应在列表中看到被隐藏的会话")
	}
}

func TestLogin_Success(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

func TestLogin_InvalidPassword(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"username": "testuser",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]string{
		"username": "nonexistent",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_MissingFields(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]string{
		"username": "testuser",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_Success(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]string{
		"username": "newuser",
		"password": "password123",
		"nickname": "New User",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegister_DuplicateUsername(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCurrentUser_Success(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

func TestUpdateUser_Success(t *testing.T) {
	r, db := setupTestRouter(t)
	createTestUser(t, db)

	body := map[string]string{
		"nickname": "Updated Name",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
