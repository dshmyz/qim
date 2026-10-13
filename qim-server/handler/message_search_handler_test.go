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
)

// TestSearchMessages_TruePagination 验证 /messages/search 返回真分页 total：
// 两条命中 + pageSize=1 → total=2，第二页返回剩余 1 条（旧实现 total=len(list) 假分页）。
func TestSearchMessages_TruePagination(t *testing.T) {
	r, db := setupTestRouter(t)
	r.Group("/api/v1").Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	}).GET("/messages/search", SearchMessages)

	user := createTestUser(t, db)
	require.Equal(t, uint(1), user.ID, "authed 中间件固定 user_id=1，须一致")

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID, Role: "member"}).Error)

	require.NoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "排期讨论第一条"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "排期讨论第二条"}).Error)

	// 第一页
	req := httptest.NewRequest("GET", "/api/v1/messages/search?keyword=%E6%8E%92%E6%9C%9F&page=1&pageSize=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
			List  []struct {
				ID uint `json:"id"`
			} `json:"list"`
			Page int `json:"page"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	assert.Equal(t, 2, resp.Data.Total, "total 应为真实命中数而非当前页长度")
	require.Len(t, resp.Data.List, 1)
	assert.Equal(t, 1, resp.Data.Page)

	// 第二页
	req2 := httptest.NewRequest("GET", "/api/v1/messages/search?keyword=%E6%8E%92%E6%9C%9F&page=2&pageSize=1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var resp2 struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
			List  []struct {
				ID uint `json:"id"`
			} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	assert.Equal(t, 2, resp2.Data.Total)
	require.Len(t, resp2.Data.List, 1, "第二页应返回剩余 1 条")
	assert.NotEqual(t, resp.Data.List[0].ID, resp2.Data.List[0].ID, "两页不应返回同一条")

	// 非本人所在会话的消息不可见（成员过滤按"我所在会话"圈定，不看发送人）
	otherConv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(otherConv).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: otherConv.ID, SenderID: user.ID, Type: "text", Content: "排期讨论第三条"}).Error)

	req3 := httptest.NewRequest("GET", "/api/v1/messages/search?keyword=%E6%8E%92%E6%9C%9F&page=1&pageSize=10", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)

	var resp3 struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp3))
	assert.Equal(t, 2, resp3.Data.Total, "非成员会话的消息不应计入搜索结果")
}

// TestSearchMessages_SingleCharKeyword 验证单字关键词在 LIKE 降级下仍可命中。
func TestSearchMessages_SingleCharKeyword(t *testing.T) {
	r, db := setupTestRouter(t)
	r.Group("/api/v1").Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	}).GET("/messages/search", SearchMessages)

	user := createTestUser(t, db)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: user.ID, Type: "text", Content: "项目排期讨论"}).Error)

	req := httptest.NewRequest("GET", "/api/v1/messages/search?keyword=%E6%8E%92&page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	assert.Equal(t, 1, resp.Data.Total, "单字关键词应命中")
}
