package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSendMessageVersionGateTest 构造 SendMessage 发送上下文：
// 用户 + 单聊会话 + 成员关系，注册 SendMessage 路由，可注入 SystemConfig 门槛。
func setupSendMessageVersionGateTest(t *testing.T) (*gin.Engine, *gorm.DB, model.User, model.Conversation) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
		&model.Message{},
		&model.SensitiveWord{},
		&model.SystemConfig{},
	))

	database.DB = db
	di.GlobalContainer = &di.Container{
		DB:                  db,
		MessageService:      service.NewMessageService(db, nil, nil),
		SystemConfigService: service.NewSystemConfigService(db),
	}
	invalidateMinSendVersionCache()

	user := model.User{Username: "u1", PasswordHash: "hash", Nickname: "用户1", Type: "user"}
	require.NoError(t, db.Create(&user).Error)
	conv := model.Conversation{Type: "single"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{
		ConversationID: conv.ID, UserID: user.ID, Role: "member",
	}).Error)

	router := gin.New()
	router.POST("/api/v1/conversations/:id/messages", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		SendMessage(c)
	})

	return router, db, user, conv
}

// TestSendMessageVersionGate 端到端验证版本门槛：
// 未配置 → 老版本/无版本均可发；配置后低于门槛或未上报版本 → 403；
// 平台专属门槛覆盖通用、未配置平台回退通用。
func TestSendMessageVersionGate(t *testing.T) {
	router, db, _, conv := setupSendMessageVersionGateTest(t)

	send := func(version, platform string) *httptest.ResponseRecorder {
		body := bytes.NewBufferString(`{"type":"text","content":"hello"}`)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/conversations/%d/messages", conv.ID), body)
		req.Header.Set("Content-Type", "application/json")
		if version != "" {
			req.Header.Set("X-App-Version", version)
		}
		if platform != "" {
			req.Header.Set("X-App-Platform", platform)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// 未配置门槛：无版本、老版本均可发送
	require.Equal(t, http.StatusOK, send("", "").Code)
	require.Equal(t, http.StatusOK, send("2.0.29", "windows").Code)

	// 配置通用门槛 2.0.30
	require.NoError(t, db.Create(&model.SystemConfig{
		ConfigKey: "client:min_send_version", Value: "2.0.30", Type: "string",
	}).Error)
	invalidateMinSendVersionCache()

	require.Equal(t, http.StatusOK, send("2.0.30", "windows").Code)
	require.Equal(t, http.StatusOK, send("2.0.31", "windows").Code)
	require.Equal(t, http.StatusForbidden, send("2.0.29", "windows").Code)
	require.Equal(t, http.StatusForbidden, send("", "windows").Code) // 未上报版本 → 拦截

	// Windows 专属门槛 2.0.35 覆盖通用；macOS 回退通用 2.0.30
	require.NoError(t, db.Create(&model.SystemConfig{
		ConfigKey: "client:min_send_version:windows", Value: "2.0.35", Type: "string",
	}).Error)
	invalidateMinSendVersionCache()

	require.Equal(t, http.StatusForbidden, send("2.0.30", "windows").Code)
	require.Equal(t, http.StatusOK, send("2.0.30", "macos").Code)
	require.Equal(t, http.StatusForbidden, send("2.0.29", "macos").Code)

	// 清空门槛后恢复可发送
	require.NoError(t, db.Where("config_key = ?", "client:min_send_version").Delete(&model.SystemConfig{}).Error)
	require.NoError(t, db.Where("config_key = ?", "client:min_send_version:windows").Delete(&model.SystemConfig{}).Error)
	invalidateMinSendVersionCache()
	require.Equal(t, http.StatusOK, send("2.0.29", "windows").Code)
}

// TestSendMessageVersionGateWSFallback 老客户端不携带 X-App-Version/X-App-Platform 头时，
// 回退到 WS 连接上报的版本/平台：WS 版本达标 → 放行，未达标 → 403。
func TestSendMessageVersionGateWSFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
		&model.Message{},
		&model.SensitiveWord{},
		&model.SystemConfig{},
	))

	database.DB = db
	di.GlobalContainer = &di.Container{
		DB:                  db,
		MessageService:      service.NewMessageService(db, nil, nil),
		SystemConfigService: service.NewSystemConfigService(db),
	}
	invalidateMinSendVersionCache()

	// 通用门槛 2.0.30
	require.NoError(t, db.Create(&model.SystemConfig{
		ConfigKey: "client:min_send_version", Value: "2.0.30", Type: "string",
	}).Error)

	newer := model.User{Username: "newer", PasswordHash: "hash", Nickname: "新版", Type: "user"}
	older := model.User{Username: "older", PasswordHash: "hash", Nickname: "旧版", Type: "user"}
	require.NoError(t, db.Create(&newer).Error)
	require.NoError(t, db.Create(&older).Error)

	hub := ws.NewHub(db, "test-secret", "http")
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = nil })
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		var u model.User
		if err := db.Where("id = ?", c.Query("uid")).First(&u).Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Set("user_id", u.ID)
		c.Set("username", u.Username)
		ws.ServeWs(hub, c)
	})
	router.POST("/api/v1/conversations/:id/messages", func(c *gin.Context) {
		uid, _ := strconv.ParseUint(c.Query("sender_id"), 10, 64)
		c.Set("user_id", uint(uid))
		SendMessage(c)
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// 两用户各建单聊会话 + 成员关系
	mkConv := func(u model.User) model.Conversation {
		conv := model.Conversation{Type: "single"}
		require.NoError(t, db.Create(&conv).Error)
		require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u.ID, Role: "member"}).Error)
		return conv
	}
	newerConv := mkConv(newer)
	olderConv := mkConv(older)

	// 老客户端拨 WS：newer 上报 2.0.35/windows（达标），older 上报 2.0.29/windows（不达标）
	dial := func(u model.User, version, platform string) {
		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") +
			fmt.Sprintf("/ws?uid=%d&version=%s&platform=%s", u.ID, version, platform)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		t.Cleanup(func() { conn.Close() })
	}
	dial(newer, "2.0.35", "windows")
	dial(older, "2.0.29", "windows")

	// 等注册完成
	require.Eventually(t, func() bool {
		return len(hub.GetVersionUsers("2.0.35")) == 1 && len(hub.GetVersionUsers("2.0.29")) == 1
	}, 3*time.Second, 20*time.Millisecond, "新旧客户端应完成 WS 注册")

	// 不带头发送：newer（WS 2.0.35 达标）→ 200；older（WS 2.0.29 不达标）→ 403
	postNoHeader := func(senderID uint, convID uint) int {
		body := bytes.NewBufferString(`{"type":"text","content":"hi"}`)
		req := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/v1/conversations/%d/messages?sender_id=%d", convID, senderID), body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}
	require.Equal(t, http.StatusOK, postNoHeader(newer.ID, newerConv.ID))
	require.Equal(t, http.StatusForbidden, postNoHeader(older.ID, olderConv.ID))
}
