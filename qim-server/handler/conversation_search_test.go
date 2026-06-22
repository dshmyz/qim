package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupSearchTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	r, db := setupTestRouter(t)

	// 只创建 Group 表（setupTestRouter 已创建其他表，AutoMigrate 会因关联表重复创建报错）
	if !db.Migrator().HasTable(&model.Group{}) {
		if err := db.Migrator().CreateTable(&model.Group{}); err != nil {
			t.Fatalf("failed to create Group table: %v", err)
		}
	}

	authed := r.Group("/api/v1")
	authed.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		c.Next()
	})
	authed.GET("/conversations/search", SearchConversations)

	return r, db
}

func createSearchTestUsers(t *testing.T, db *gorm.DB) {
	db.Create(&model.User{ID: 1, Username: "user1", Nickname: "当前用户"})
	db.Create(&model.User{ID: 2, Username: "user2", Nickname: "张三"})
	db.Create(&model.User{ID: 3, Username: "user3", Nickname: "李四"})
}

// 搜索群聊名称应返回匹配的群聊会话
func TestSearchConversations_ByGroupName(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	createSearchTestUsers(t, db)

	conv1 := model.Conversation{ID: 10, Type: "group"}
	db.Create(&conv1)
	db.Create(&model.Group{ConversationID: 10, Name: "项目组讨论", GroupType: "group", CreatorID: 1})
	db.Create(&model.ConversationMember{ConversationID: 10, UserID: 1, Role: "owner"})

	conv2 := model.Conversation{ID: 11, Type: "group"}
	db.Create(&conv2)
	db.Create(&model.Group{ConversationID: 11, Name: "闲聊群", GroupType: "group", CreatorID: 1})
	db.Create(&model.ConversationMember{ConversationID: 11, UserID: 1, Role: "owner"})

	req := httptest.NewRequest("GET", "/api/v1/conversations/search?query=项目", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int                      `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, float64(10), resp.Data[0]["id"])
}

// 搜索单聊对方昵称应返回匹配的单聊会话
func TestSearchConversations_BySingleChatNickname(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	createSearchTestUsers(t, db)

	conv := model.Conversation{ID: 20, Type: "single"}
	db.Create(&conv)
	db.Create(&model.ConversationMember{ConversationID: 20, UserID: 1, Role: "member"})
	db.Create(&model.ConversationMember{ConversationID: 20, UserID: 2, Role: "member"})

	req := httptest.NewRequest("GET", "/api/v1/conversations/search?query=张", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int                      `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 1)
}

// 空查询应返回空列表
func TestSearchConversations_EmptyQuery(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	createSearchTestUsers(t, db)

	conv := model.Conversation{ID: 30, Type: "group"}
	db.Create(&conv)
	db.Create(&model.Group{ConversationID: 30, Name: "测试群", GroupType: "group", CreatorID: 1})
	db.Create(&model.ConversationMember{ConversationID: 30, UserID: 1, Role: "owner"})

	req := httptest.NewRequest("GET", "/api/v1/conversations/search?query=", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int                      `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 0, resp.Code)
	assert.Empty(t, resp.Data)
}

// 只返回当前用户参与的会话
func TestSearchConversations_OnlyUserConversations(t *testing.T) {
	r, db := setupSearchTestRouter(t)
	createSearchTestUsers(t, db)

	conv1 := model.Conversation{ID: 40, Type: "group"}
	db.Create(&conv1)
	db.Create(&model.Group{ConversationID: 40, Name: "项目组", GroupType: "group", CreatorID: 1})
	db.Create(&model.ConversationMember{ConversationID: 40, UserID: 1, Role: "owner"})

	conv2 := model.Conversation{ID: 41, Type: "group"}
	db.Create(&conv2)
	db.Create(&model.Group{ConversationID: 41, Name: "项目组2", GroupType: "group", CreatorID: 2})
	db.Create(&model.ConversationMember{ConversationID: 41, UserID: 2, Role: "owner"})

	req := httptest.NewRequest("GET", "/api/v1/conversations/search?query=项目", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int                      `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, float64(40), resp.Data[0]["id"])
}
