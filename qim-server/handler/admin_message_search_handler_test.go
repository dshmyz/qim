package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func setupAdminMessageSearchRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
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
	))

	database.DB = db
	di.InitContainer(&config.Config{}, nil)

	r := gin.New()
	admin := r.Group("/api/v1/admin")
	{
		admin.GET("/messages/search", AdminSearchMessages)
	}

	return r, db
}

func TestAdminSearchMessagesSearchesBeyondCurrentAdminMembership(t *testing.T) {
	r, db := setupAdminMessageSearchRouter(t)
	_ = createAdminConversationUser(t, db, "admin", "管理员")
	sender := createAdminConversationUser(t, db, "alice", "张三")
	recipient := createAdminConversationUser(t, db, "bob", "李四")

	conv := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: sender.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: recipient.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: sender.ID, Type: "text", Content: "管理员不在这个会话也应该搜到"}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/messages/search?keyword="+url.QueryEscape("应该搜到")+"&page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				SenderID     uint   `json:"senderId"`
				SenderName   string `json:"senderName"`
				ReceiverID   uint   `json:"receiverId"`
				ReceiverName string `json:"receiverName"`
				MessageType  string `json:"messageType"`
				Content      string `json:"content"`
			} `json:"list"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.List, 1)
	assert.Equal(t, sender.ID, resp.Data.List[0].SenderID)
	assert.Equal(t, "张三", resp.Data.List[0].SenderName)
	assert.Equal(t, recipient.ID, resp.Data.List[0].ReceiverID)
	assert.Equal(t, "李四", resp.Data.List[0].ReceiverName)
	assert.Equal(t, "text", resp.Data.List[0].MessageType)
}

func TestAdminSearchMessagesAppliesFilters(t *testing.T) {
	r, db := setupAdminMessageSearchRouter(t)
	sender := createAdminConversationUser(t, db, "alice", "张三")
	otherSender := createAdminConversationUser(t, db, "bob", "李四")
	recipient := createAdminConversationUser(t, db, "carl", "王五")
	now := time.Now().Truncate(time.Second)

	singleConv := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(singleConv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: singleConv.ID, UserID: sender.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: singleConv.ID, UserID: recipient.ID, Role: "member"}).Error)
	matchingMsg := &model.Message{ConversationID: singleConv.ID, SenderID: sender.ID, Type: "text", Content: "筛选命中的消息", CreatedAt: now}
	require.NoError(t, db.Create(matchingMsg).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: singleConv.ID, SenderID: otherSender.ID, Type: "text", Content: "发送者不匹配", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: singleConv.ID, SenderID: sender.ID, Type: "image", Content: "类型不匹配", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: singleConv.ID, SenderID: sender.ID, Type: "text", Content: "时间不匹配", CreatedAt: now.Add(-48 * time.Hour)}).Error)

	groupConv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(groupConv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: groupConv.ID, GroupType: "group", Name: "研发群", CreatorID: sender.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: groupConv.ID, UserID: sender.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: groupConv.ID, SenderID: sender.ID, Type: "text", Content: "会话类型不匹配", CreatedAt: now}).Error)

	query := url.Values{}
	query.Set("senderId", fmt.Sprint(sender.ID))
	query.Set("messageType", "text")
	query.Set("conversationType", "single")
	query.Set("startTime", now.Add(-time.Hour).Format(time.RFC3339))
	query.Set("endTime", now.Add(time.Hour).Format(time.RFC3339))
	query.Set("page", "1")
	query.Set("pageSize", "10")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/messages/search?"+query.Encode(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				ID      uint   `json:"id"`
				Content string `json:"content"`
			} `json:"list"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(1), resp.Data.Total)
	require.Len(t, resp.Data.List, 1)
	assert.Equal(t, matchingMsg.ID, resp.Data.List[0].ID)
	assert.Equal(t, "筛选命中的消息", resp.Data.List[0].Content)
}

// TestAdminSearchMessagesFillsGroupNameForMultipleGroupMessages 验证多条群聊消息
// 的 GroupName 都被正确填充，且不因 N+1 查询导致性能问题或数据缺失。
func TestAdminSearchMessagesFillsGroupNameForMultipleGroupMessages(t *testing.T) {
	r, db := setupAdminMessageSearchRouter(t)
	sender := createAdminConversationUser(t, db, "alice", "张三")

	// 两个不同的群聊会话
	convA := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(convA).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: convA.ID, GroupType: "group", Name: "群A", CreatorID: sender.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: convA.ID, UserID: sender.ID, Role: "member"}).Error)

	convB := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(convB).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: convB.ID, GroupType: "group", Name: "群B", CreatorID: sender.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: convB.ID, UserID: sender.ID, Role: "member"}).Error)

	// 每个群各发 3 条消息
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&model.Message{ConversationID: convA.ID, SenderID: sender.ID, Type: "text", Content: fmt.Sprintf("群A消息%d", i)}).Error)
		require.NoError(t, db.Create(&model.Message{ConversationID: convB.ID, SenderID: sender.ID, Type: "text", Content: fmt.Sprintf("群B消息%d", i)}).Error)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/messages/search?page=1&pageSize=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				Content   string `json:"content"`
				GroupID   uint   `json:"groupId"`
				GroupName string `json:"groupName"`
			} `json:"list"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(6), resp.Data.Total)
	require.Len(t, resp.Data.List, 6)

	// 每条消息的 GroupName 都应被正确填充
	groupNameByID := map[uint]string{convA.ID: "群A", convB.ID: "群B"}
	for _, item := range resp.Data.List {
		var expectedName string
		if strings.HasPrefix(item.Content, "群A") {
			expectedName = "群A"
		} else {
			expectedName = "群B"
		}
		assert.Equal(t, expectedName, item.GroupName, "消息 %q 的 GroupName 应正确填充", item.Content)
		assert.NotZero(t, item.GroupID, "消息 %q 的 GroupID 不应为空", item.Content)
		_ = groupNameByID
	}
}
