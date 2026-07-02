package handler

import (
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
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAdminConversationRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
		&model.Group{},
		&model.Message{},
		&model.Bot{},
		&model.BotConversation{},
	))

	database.DB = db
	di.InitContainer(&config.Config{}, nil)

	r := gin.New()
	admin := r.Group("/api/v1/admin")
	{
		admin.GET("/conversations", AdminGetConversations)
		admin.GET("/conversations/:id/members", AdminGetConversationMembers)
		admin.DELETE("/conversations/:id", AdminDeleteConversation)
	}

	return r, db
}

func createAdminConversationUser(t *testing.T, db *gorm.DB, username, nickname string) *model.User {
	t.Helper()
	user := &model.User{Username: username, PasswordHash: "hash", Nickname: nickname, Status: "offline"}
	require.NoError(t, db.Create(user).Error)
	return user
}

func TestAdminGetConversationsListsAllConversationsWithFilters(t *testing.T) {
	r, db := setupAdminConversationRouter(t)
	adminUser := createAdminConversationUser(t, db, "admin", "管理员")
	userA := createAdminConversationUser(t, db, "alice", "张三")
	userB := createAdminConversationUser(t, db, "bob", "李四")

	singleConv := &model.Conversation{Type: "single", CreatedAt: time.Now().Add(-time.Hour)}
	require.NoError(t, db.Create(singleConv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: singleConv.ID, UserID: userA.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: singleConv.ID, UserID: userB.ID, Role: "member"}).Error)

	groupConv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(groupConv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: groupConv.ID, GroupType: "group", Name: "研发群", CreatorID: userA.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: groupConv.ID, UserID: userA.ID, Role: "owner"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: groupConv.ID, UserID: adminUser.ID, Role: "member"}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/conversations?page=1&pageSize=10&type=group&keyword=研发", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID          uint   `json:"id"`
				Type        string `json:"type"`
				Name        string `json:"name"`
				CreatorName string `json:"creatorName"`
				MemberCount int64  `json:"memberCount"`
			} `json:"list"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.List, 1)
	assert.Equal(t, groupConv.ID, resp.Data.List[0].ID)
	assert.Equal(t, "group", resp.Data.List[0].Type)
	assert.Equal(t, "研发群", resp.Data.List[0].Name)
	assert.Equal(t, userA.Nickname, resp.Data.List[0].CreatorName)
	assert.Equal(t, int64(2), resp.Data.List[0].MemberCount)
}

func TestAdminGetConversationMembersSupportsSingleConversation(t *testing.T) {
	r, db := setupAdminConversationRouter(t)
	userA := createAdminConversationUser(t, db, "alice", "张三")
	userB := createAdminConversationUser(t, db, "bob", "李四")

	conv := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userA.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userB.ID, Role: "member"}).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/conversations/%d/members?page=1&pageSize=10", conv.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				UserID   uint   `json:"userId"`
				Username string `json:"username"`
				Nickname string `json:"nickname"`
				Role     string `json:"role"`
			} `json:"list"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(2), resp.Data.Total)
	require.Len(t, resp.Data.List, 2)
	assert.ElementsMatch(t, []uint{userA.ID, userB.ID}, []uint{resp.Data.List[0].UserID, resp.Data.List[1].UserID})
}

func TestAdminDeleteConversationMarksConversationDeleted(t *testing.T) {
	r, db := setupAdminConversationRouter(t)
	owner := createAdminConversationUser(t, db, "owner", "群主")

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "项目群", CreatorID: owner.ID}).Error)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/conversations/%d", conv.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var updated model.Conversation
	require.NoError(t, db.First(&updated, conv.ID).Error)
	assert.True(t, updated.IsDeleted)

	var group model.Group
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).First(&group).Error)
	assert.Equal(t, "[已解散] 项目群", group.Name)
}
