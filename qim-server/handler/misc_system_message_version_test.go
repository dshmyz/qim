package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupSystemMessageVersionTest 构造定向版本通知的端到端上下文：
// 内存库（User/SystemMessage/Notification）+ 真实 WS Hub（Run 后台跑）+ httptest 服务器，
// 提供 /ws 升级入口（uid 查询参数直接以已认证身份注册）与 CreateSystemMessage 路由。
func setupSystemMessageVersionTest(t *testing.T) (*gorm.DB, *ws.Hub, *httptest.Server, []model.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.SystemMessage{}, &model.Notification{}))

	database.DB = db
	di.GlobalContainer = &di.Container{DB: db}

	admin := model.User{Username: "admin", PasswordHash: "hash", Nickname: "管理员", Type: "system"}
	alice := model.User{Username: "alice", PasswordHash: "hash", Nickname: "Alice", Type: "user"}
	dave := model.User{Username: "dave", PasswordHash: "hash", Nickname: "Dave", Type: "user"}
	bob := model.User{Username: "bob", PasswordHash: "hash", Nickname: "Bob", Type: "user"}
	old := model.User{Username: "old", PasswordHash: "hash", Nickname: "Old", Type: "user"}
	require.NoError(t, db.Create(&admin).Error)
	require.NoError(t, db.Create(&alice).Error)
	require.NoError(t, db.Create(&dave).Error)
	require.NoError(t, db.Create(&bob).Error)
	require.NoError(t, db.Create(&old).Error)

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
	router.POST("/api/v1/system-messages", func(c *gin.Context) {
		c.Set("user_id", admin.ID)
		CreateSystemMessage(c)
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return db, hub, srv, []model.User{alice, dave, bob, old}
}

// dialVersionClient 以指定版本/平台拨号一条已认证 WS 连接（uid 走 context 免 token 握手）。
func dialVersionClient(t *testing.T, srv *httptest.Server, uid uint, version, platform string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") +
		fmt.Sprintf("/ws?uid=%d&version=%s&platform=%s", uid, version, platform)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestCreateSystemMessageTargetVersion 端到端验证「定向群发通知给某版本的在线用户」：
// 仅 2.1.0 在线的 alice 与 dave 收到各自的通知记录（user_id 与本人一致，可正常已读/删除）；
// 2.2.0 的 bob 与未上报版本（未知版本桶）的 old 既不收 WS 也不落记录。
func TestCreateSystemMessageTargetVersion(t *testing.T) {
	db, hub, srv, users := setupSystemMessageVersionTest(t)
	alice, dave, bob, old := users[0], users[1], users[2], users[3]

	aliceConn := dialVersionClient(t, srv, alice.ID, "2.1.0", "windows")
	daveConn := dialVersionClient(t, srv, dave.ID, "2.1.0", "macos")
	bobConn := dialVersionClient(t, srv, bob.ID, "2.2.0", "windows")
	oldConn := dialVersionClient(t, srv, old.ID, "", "macos")

	// 注册经 Run() 异步消费 register 通道，轮询 GetVersionUsers 确认 2.1.0 用户已上线再发通知
	require.Eventually(t, func() bool {
		matched := 0
		for _, u := range hub.GetVersionUsers("2.1.0") {
			if u.UserID == alice.ID || u.UserID == dave.ID {
				matched++
			}
		}
		return matched == 2
	}, 3*time.Second, 20*time.Millisecond, "alice/dave(2.1.0) 应已完成 WS 注册")

	body := bytes.NewBufferString(`{"title":"升级提醒","content":"请升级到最新版本","target_type":"version","target_version":"2.1.0"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system-messages", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Config.Handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 每个匹配用户应收到 user_id 与本人一致的各自通知记录（修复：不再广播首个用户的记录）
	readNotif := func(conn *websocket.Conn) (title string, userID uint) {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, payload, err := conn.ReadMessage()
		require.NoError(t, err)
		var notif struct {
			Type string `json:"type"`
			Data struct {
				Title  string `json:"title"`
				UserID uint   `json:"user_id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(payload, &notif))
		require.Equal(t, "notification", notif.Type)
		return notif.Data.Title, notif.Data.UserID
	}

	title, uid := readNotif(aliceConn)
	require.Equal(t, "升级提醒", title)
	require.Equal(t, alice.ID, uid, "alice 收到的通知记录 user_id 应为自己")

	title, uid = readNotif(daveConn)
	require.Equal(t, "升级提醒", title)
	require.Equal(t, dave.ID, uid, "dave 收到的通知记录 user_id 应为自己")

	// bob（2.2.0）与 old（未知版本）不应收到
	for _, conn := range []*websocket.Conn{bobConn, oldConn} {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		if _, _, err := conn.ReadMessage(); err == nil {
			t.Fatalf("非目标版本客户端不应收到通知")
		}
	}

	// Notification 记录只写给匹配用户，各一条、user_id 各自正确
	var notifications []model.Notification
	require.NoError(t, db.Find(&notifications).Error)
	require.Len(t, notifications, 2)
	got := map[uint]bool{}
	for _, n := range notifications {
		require.Equal(t, "system_message", n.Type)
		got[n.UserID] = true
	}
	require.True(t, got[alice.ID] && got[dave.ID], "通知记录应分别写给 alice 与 dave")

	// SystemMessage 行记录了定向版本（平台未指定为空）
	var sysMsg model.SystemMessage
	require.NoError(t, db.First(&sysMsg).Error)
	require.Equal(t, "version", sysMsg.TargetType)
	require.Equal(t, "2.1.0", sysMsg.TargetVersion)
	require.Empty(t, sysMsg.TargetPlatform)
}
