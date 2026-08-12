package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newAvatarHandlerTestDB 构造仅含 AvatarSession 的测试库，供 ClearSessions 单测使用。
func newAvatarHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AvatarSession{}))
	return db
}

// TestClearSessions_DeletesOnlyOwnSessions 验证清空接口删除该分身用户全部会话行，
// 且不误删其他用户的会话行（按 user_id 隔离）。
func TestClearSessions_DeletesOnlyOwnSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAvatarHandlerTestDB(t)
	h := &AvatarHandler{db: db}

	// 用户 1 有三条会话设置（覆盖不同会话）
	require.NoError(t, db.Create(&model.AvatarSession{UserID: 1, ConversationID: 10, AvatarEnabled: true}).Error)
	require.NoError(t, db.Create(&model.AvatarSession{UserID: 1, ConversationID: 11, AvatarEnabled: false}).Error)
	require.NoError(t, db.Create(&model.AvatarSession{UserID: 1, ConversationID: 12, AvatarEnabled: true}).Error)
	// 用户 2 的一条，不应被删除
	require.NoError(t, db.Create(&model.AvatarSession{UserID: 2, ConversationID: 99, AvatarEnabled: true}).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/avatar/sessions", nil)
	c.Set("user_id", uint(1))

	h.ClearSessions(c)

	// 响应 code=0
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, http.StatusOK, w.Code)

	// 用户 1 的行全部清除
	var count1 int64
	require.NoError(t, db.Model(&model.AvatarSession{}).Where("user_id = ?", 1).Count(&count1).Error)
	assert.Equal(t, int64(0), count1)

	// 用户 2 的行保留
	var count2 int64
	require.NoError(t, db.Model(&model.AvatarSession{}).Where("user_id = ?", 2).Count(&count2).Error)
	assert.Equal(t, int64(1), count2)
}

// TestClearSessions_EmptyIsNoOp 验证无会话行时清空不报错（幂等）。
func TestClearSessions_EmptyIsNoOp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAvatarHandlerTestDB(t)
	h := &AvatarHandler{db: db}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/avatar/sessions", nil)
	c.Set("user_id", uint(42))

	h.ClearSessions(c)

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestApplyOwnMessagePause 覆盖「你发消息后，分身暂停回复」：
// 配置 SelfMessagePause>0 且会话存在分身会话 → 写入 TakeoverUntil（now+pause）；
// 未配置 / 无分身会话 → 静默跳过，不改写 TakeoverUntil。
func TestApplyOwnMessagePause(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AvatarConfig{}, &model.AvatarSession{}))

	// 用户 1：配置 self_message_pause=10，会话 10 有分身会话，会话 20 无分身会话
	require.NoError(t, db.Create(&model.AvatarConfig{UserID: 1, SelfMessagePause: 10}).Error)
	before := time.Now()
	require.NoError(t, db.Create(&model.AvatarSession{UserID: 1, ConversationID: 10, AvatarEnabled: true}).Error)
	// 用户 2：未配置暂停
	require.NoError(t, db.Create(&model.AvatarConfig{UserID: 2, SelfMessagePause: 0}).Error)

	t.Run("配置并存在会话→写入暂停窗口", func(t *testing.T) {
		applyOwnMessagePause(db, 1, 10)
		var s model.AvatarSession
		require.NoError(t, db.Where("user_id = ? AND conversation_id = ?", 1, 10).First(&s).Error)
		require.NotNil(t, s.TakeoverUntil, "应设置暂停窗口")
		// 约为 now + 10 分钟（容差 2s）
		want := before.Add(10 * time.Minute)
		assert.InDelta(t, want.Unix(), s.TakeoverUntil.Unix(), 2, "暂停窗口应为 now+10min")
	})

	t.Run("会话无分身会话→不改写", func(t *testing.T) {
		applyOwnMessagePause(db, 1, 20)
		var count int64
		db.Model(&model.AvatarSession{}).Where("user_id = ? AND conversation_id = ?", 1, 20).Count(&count)
		assert.Equal(t, int64(0), count, "不应创建会话行")
	})

	t.Run("未配置暂停→不改写", func(t *testing.T) {
		applyOwnMessagePause(db, 2, 10)
		var count int64
		db.Model(&model.AvatarSession{}).Where("user_id = ?", 2).Count(&count)
		assert.Equal(t, int64(0), count, "未配置暂停不应写入")
	})

	t.Run("非分身主人→静默", func(t *testing.T) {
		applyOwnMessagePause(db, 999, 10)
		var count int64
		db.Model(&model.AvatarSession{}).Where("user_id = ?", 999).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

// TestTriggerLearnPersona_MissingUserID 验证 user_id 不在 context 时返回 401 而非 panic。
func TestTriggerLearnPersona_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AvatarConfig{}, &model.AvatarLearnTask{}))
	h := &AvatarHandler{db: db}

	// 不设置 user_id → 以前会 panic，现在应返回 401
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/avatar/learn-persona", nil)

	assert.NotPanics(t, func() { h.TriggerLearnPersona(c) })
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestTriggerLearnPersona_BadUserIDType 验证 user_id 类型非 uint 时返回 400 而非 panic。
func TestTriggerLearnPersona_BadUserIDType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AvatarConfig{}, &model.AvatarLearnTask{}))
	h := &AvatarHandler{db: db}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/avatar/learn-persona", nil)
	c.Set("user_id", "not-a-uint") // 字符串而非 uint

	assert.NotPanics(t, func() { h.TriggerLearnPersona(c) })
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
