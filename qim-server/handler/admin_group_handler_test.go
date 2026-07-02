package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

func setupAdminGroupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.UserRole{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Group{},
	))

	database.DB = db
	di.InitContainer(&config.Config{}, nil)

	r := gin.New()
	admin := r.Group("/api/v1/admin")
	{
		admin.GET("/groups/:id/members", AdminGetGroupMembers)
		admin.DELETE("/groups/:id", AdminDeleteGroup)
	}

	return r, db
}

func TestAdminGetGroupMembersReturnsMembersByConversationID(t *testing.T) {
	r, db := setupAdminGroupRouter(t)
	owner := createAdminConversationUser(t, db, "owner", "群主")
	member := createAdminConversationUser(t, db, "member", "成员")
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "技术交流群", CreatorID: owner.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "owner"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: member.ID, Role: "member"}).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/groups/%d/members?page=1&pageSize=10", conv.ID), nil)
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
	assert.ElementsMatch(t, []uint{owner.ID, member.ID}, []uint{resp.Data.List[0].UserID, resp.Data.List[1].UserID})
}

// TestAdminDeleteGroupSoftDeletesAndKeepsMembers 验证 AdminDeleteGroup 走软删除路径：
// 会话 IsDeleted=true、群名加 [已解散] 前缀、成员关系保留（不物理删除）。
func TestAdminDeleteGroupSoftDeletesAndKeepsMembers(t *testing.T) {
	r, db := setupAdminGroupRouter(t)
	owner := createAdminConversationUser(t, db, "owner", "群主")
	member := createAdminConversationUser(t, db, "member", "成员")
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "技术交流群", CreatorID: owner.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "owner"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: member.ID, Role: "member"}).Error)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/groups/%d", conv.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var updated model.Conversation
	require.NoError(t, db.First(&updated, conv.ID).Error)
	assert.True(t, updated.IsDeleted, "会话应被软删除")

	var group model.Group
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).First(&group).Error)
	assert.Equal(t, "[已解散] 技术交流群", group.Name, "群名应加 [已解散] 前缀")

	var memberCount int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", conv.ID).Count(&memberCount)
	assert.Equal(t, int64(2), memberCount, "成员关系应保留，不物理删除")
}
