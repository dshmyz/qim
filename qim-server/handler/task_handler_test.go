package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTaskListTestRouter 复用 setupTestRouter 的 DI 容器，补建 Task 表并注册 /tasks 路由
func setupTaskListTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	r, db := setupTestRouter(t)

	// setupHandlerTestDB 未建 Task 表，这里补建
	if !db.Migrator().HasTable(&model.Task{}) {
		require.NoError(t, db.Migrator().CreateTable(&model.Task{}))
	}

	authed := r.Group("/api/v1")
	authed.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		c.Next()
	})
	authed.GET("/tasks", GetTasks)

	return r, db
}

// 带 conversation_id 且是会话成员 → 返回该会话的任务列表
func TestGetTasks_ByConversation_MemberCanList(t *testing.T) {
	r, db := setupTaskListTestRouter(t)

	// 当前用户 user_id=1
	conv := model.Conversation{ID: 100, Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: 100, UserID: 1, Role: "member"}).Error)

	// 会话 100 的两条任务
	require.NoError(t, db.Create(&model.Task{Title: "任务一", UserID: 1, ConversationID: 100}).Error)
	require.NoError(t, db.Create(&model.Task{Title: "任务二", UserID: 2, ConversationID: 100}).Error)
	// 会话 200 的任务（不应被查出）
	require.NoError(t, db.Create(&model.Task{Title: "其他会话任务", UserID: 1, ConversationID: 200}).Error)

	req := httptest.NewRequest("GET", "/api/v1/tasks?conversation_id=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 2)
}

// 带 conversation_id 但非会话成员 → 403
func TestGetTasks_ByConversation_NonMemberForbidden(t *testing.T) {
	r, db := setupTaskListTestRouter(t)

	conv := model.Conversation{ID: 100, Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	// 注意：不创建 user_id=1 的成员记录
	require.NoError(t, db.Create(&model.Task{Title: "任务一", UserID: 2, ConversationID: 100}).Error)

	req := httptest.NewRequest("GET", "/api/v1/tasks?conversation_id=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 非成员应被拒绝（403 或错误码）
	assert.NotEqual(t, http.StatusOK, w.Code, "非会话成员不应拿到任务列表")
}

// 带 conversation_id=0 → 拒绝
func TestGetTasks_ByConversation_ZeroIDRejected(t *testing.T) {
	r, db := setupTaskListTestRouter(t)
	// 插入一条 ConversationID=0 的私人任务，确保不会被泄露
	require.NoError(t, db.Create(&model.Task{Title: "私人任务", UserID: 1, ConversationID: 0}).Error)

	req := httptest.NewRequest("GET", "/api/v1/tasks?conversation_id=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "conversation_id=0 应被拒绝")
}

// 不带 conversation_id → 返回当前用户的全部任务（原行为不变）
func TestGetTasks_NoConversation_ReturnsUserTasks(t *testing.T) {
	r, db := setupTaskListTestRouter(t)

	// user_id=1 的任务（创建 + 指派）
	require.NoError(t, db.Create(&model.Task{Title: "我的任务", UserID: 1, ConversationID: 0}).Error)
	require.NoError(t, db.Create(&model.Task{Title: "指派给我", UserID: 2, ConversationID: 0, AssigneeID: "1"}).Error)
	// 他人任务，不应被查出
	require.NoError(t, db.Create(&model.Task{Title: "别人的任务", UserID: 2, ConversationID: 0}).Error)

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	// 应含"我的任务"和"指派给我"两条
	titles := make([]string, 0, len(resp.Data))
	for _, d := range resp.Data {
		titles = append(titles, d.Title)
	}
	assert.Contains(t, titles, "我的任务")
	assert.Contains(t, titles, "指派给我")
	assert.NotContains(t, titles, "别人的任务")
}
