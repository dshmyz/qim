package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupGroupFileHandler(t *testing.T) (*gin.Engine, *gorm.DB, *model.Group, *model.User, *model.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Group{},
		&model.File{},
		&model.Folder{},
	))

	owner := &model.User{Username: "group-file-owner", PasswordHash: "hash"}
	member := &model.User{Username: "group-file-member", PasswordHash: "hash"}
	require.NoError(t, db.Create(owner).Error)
	require.NoError(t, db.Create(member).Error)

	conversation := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conversation).Error)
	group := &model.Group{ConversationID: conversation.ID, GroupType: "group", Name: "文件群", CreatorID: owner.ID}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: owner.ID, Role: "owner"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conversation.ID, UserID: member.ID, Role: "member"}).Error)

	previousContainer := di.GlobalContainer
	t.Cleanup(func() { di.GlobalContainer = previousContainer })
	di.GlobalContainer = &di.Container{
		GroupService:     service.NewGroupService(db),
		FileSpaceService: service.NewFileSpaceService(db),
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		userID, err := strconv.ParseUint(c.GetHeader("X-Test-User-ID"), 10, 32)
		if err == nil {
			c.Set("user_id", uint(userID))
		}
	})
	RegisterGroupFileRoutes(router.Group("/api/v1"))
	return router, db, group, owner, member
}

func requestAsGroupFileUser(t *testing.T, router http.Handler, userID uint, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Test-User-ID", strconv.FormatUint(uint64(userID), 10))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestGroupFileHandlerForbidsMemberFolderCreation(t *testing.T) {
	router, _, group, _, member := setupGroupFileHandler(t)

	response := requestAsGroupFileUser(t, router, member.ID, http.MethodPost, "/api/v1/groups/"+strconv.FormatUint(uint64(group.ConversationID), 10)+"/folders", `{"name":"规范"}`)
	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestGroupFileHandlerListsOnlyRequestedGroupScope(t *testing.T) {
	router, db, group, owner, member := setupGroupFileHandler(t)
	otherConversation := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(otherConversation).Error)
	otherGroup := &model.Group{ConversationID: otherConversation.ID, GroupType: "group", Name: "其他群", CreatorID: owner.ID}
	require.NoError(t, db.Create(otherGroup).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: otherConversation.ID, UserID: owner.ID, Role: "owner"}).Error)
	require.NoError(t, db.Create(&model.File{UserID: owner.ID, ScopeType: "group", ScopeID: group.ID, Name: "群文件.txt", StoragePath: "uploads/group-file", Size: 1}).Error)
	require.NoError(t, db.Create(&model.File{UserID: owner.ID, ScopeType: "group", ScopeID: otherGroup.ID, Name: "其他群文件.txt", StoragePath: "uploads/other-group-file", Size: 1}).Error)

	response := requestAsGroupFileUser(t, router, member.ID, http.MethodGet, "/api/v1/groups/"+strconv.FormatUint(uint64(group.ConversationID), 10)+"/files", "")
	require.Equal(t, http.StatusOK, response.Code)

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Total int64        `json:"total"`
			Files []model.File `json:"files"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	require.Equal(t, int64(1), payload.Data.Total)
	require.Len(t, payload.Data.Files, 1)
	require.Equal(t, "群文件.txt", payload.Data.Files[0].Name)
}
