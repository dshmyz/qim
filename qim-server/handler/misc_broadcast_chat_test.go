package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupBroadcastChatTest 构造后台服务上下文：迁移表、注入 DI 容器、
// 创建 1 个系统账号(type=system) + 2 个普通用户，并注册 BroadcastChatMessage 及其任务查询路由。
func setupBroadcastChatTest(t *testing.T) (*gin.Engine, *gorm.DB, model.User) {
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
	))

	database.DB = db
	di.GlobalContainer = &di.Container{
		DB:                  db,
		AIService:           ai.NewAIService(&ai.AIConfig{}),
		ConversationService: service.NewConversationService(db),
		MessageService:      service.NewMessageService(db, nil, nil),
	}

	sysUser := model.User{Username: "system", PasswordHash: "hash", Nickname: "系统", Type: "system"}
	user1 := model.User{Username: "u1", PasswordHash: "hash", Nickname: "用户1", Type: "user"}
	user2 := model.User{Username: "u2", PasswordHash: "hash", Nickname: "用户2", Type: "user"}
	require.NoError(t, db.Create(&sysUser).Error)
	require.NoError(t, db.Create(&user1).Error)
	require.NoError(t, db.Create(&user2).Error)

	router := gin.New()
	router.POST("/api/v1/system-messages/broadcast-chat", func(c *gin.Context) {
		c.Set("user_id", sysUser.ID)
		BroadcastChatMessage(c)
	})
	router.GET("/api/v1/system-messages/broadcast-chat/jobs/:id", func(c *gin.Context) {
		c.Set("user_id", sysUser.ID)
		GetBroadcastChatJob(c)
	})

	return router, db, sysUser
}

// postBroadcast 提交群发任务并轮询任务结果直至 done（异步执行），返回任务结果。
func postBroadcast(t *testing.T, router *gin.Engine, body string) (sent, failed, skipped int) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/system-messages/broadcast-chat",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			JobID string `json:"job_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.JobID)

	// 轮询任务结果（worker 池并发执行，SQLite 单连接下通常瞬时完成）
	var job broadcastChatJob
	require.Eventually(t, func() bool {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system-messages/broadcast-chat/jobs/"+resp.Data.JobID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			return false
		}
		var r struct {
			Data broadcastChatJob `json:"data"`
		}
		if json.Unmarshal(w.Body.Bytes(), &r) != nil {
			return false
		}
		job = r.Data
		return job.Status == "done"
	}, 5*time.Second, 20*time.Millisecond, "群发任务应在超时内完成")

	return job.Sent, job.Failed, job.Skipped
}

// TestBroadcastChatMessageToAll 全员私聊：系统账号向所有普通用户单聊发一条 text 消息。
// 断言：每个目标用户都收到、消息发送者为系统账号、单聊会话存在成员双方、对端未读+1。
func TestBroadcastChatMessageToAll(t *testing.T) {
	router, db, _ := setupBroadcastChatTest(t)

	var users []model.User
	db.Where("type != ?", "system").Find(&users)
	require.Len(t, users, 2)

	sent, failed, skipped := postBroadcast(t, router, `{"content":"请升级到最新版本"}`)
	require.Equal(t, 2, sent)
	require.Equal(t, 0, failed)
	require.Equal(t, 0, skipped)

	var messages []model.Message
	db.Find(&messages)
	require.Len(t, messages, 2)

	// 每个普通用户都收到系统账号发的消息
	for _, u := range users {
		var msg model.Message
		err := db.Where("sender_id = (?)", db.Select("id").Where("username = ?", "system").Table("users")).
			Where("conversation_id IN (?)", db.Select("conversation_id").Where("user_id = ?", u.ID).Table("conversation_members")).
			Order("id DESC").First(&msg).Error
		require.NoError(t, err)
		require.Equal(t, "请升级到最新版本", msg.Content)
	}

	// 每个普通用户的单聊会话成员双方已建立，且对端未读=1
	for _, u := range users {
		var convID uint
		require.NoError(t, db.Model(&model.ConversationMember{}).
			Where("user_id = ? AND conversation_id IN (?)",
				u.ID,
				db.Select("conversation_id").Where("user_id = (?)",
					db.Select("id").Where("username = ?", "system").Table("users")).Table("conversation_members")).
			Pluck("conversation_id", &convID).Error)
		require.Greater(t, convID, uint(0))

		var members []model.ConversationMember
		db.Where("conversation_id = ?", convID).Find(&members)
		require.Len(t, members, 2)

		var mine model.ConversationMember
		require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", convID, u.ID).First(&mine).Error)
		require.Equal(t, 1, mine.UnreadCount)
	}
}

// TestBroadcastChatMessageReusesConversation 幂等：再次全员发送应复用既有单聊会话，不重复创建。
func TestBroadcastChatMessageReusesConversation(t *testing.T) {
	router, db, _ := setupBroadcastChatTest(t)

	postBroadcast(t, router, `{"content":"hello"}`)
	var convs1 int64
	db.Model(&model.Conversation{}).Count(&convs1)
	require.Equal(t, int64(2), convs1) // 一个用户一个单聊会话

	postBroadcast(t, router, `{"content":"hello"}`)
	var convs2 int64
	db.Model(&model.Conversation{}).Count(&convs2)
	require.Equal(t, convs1, convs2) // 复用旧会话，不新增

	var total int64
	db.Model(&model.Message{}).Count(&total)
	require.Equal(t, int64(4), total) // 两次发送，每条两人共 4 条
}

// TestBroadcastChatMessageTargeted 定向：仅发给指定 target_user_ids 的用户，其它收不到。
func TestBroadcastChatMessageTargeted(t *testing.T) {
	router, db, _ := setupBroadcastChatTest(t)

	var user2 model.User
	db.Where("username = ?", "u2").First(&user2)

	postBroadcast(t, router, fmt.Sprintf(`{"content":"仅通知你","target_user_ids":[%d]}`, user2.ID))

	var total int64
	db.Model(&model.Message{}).Count(&total)
	require.Equal(t, int64(1), total) // 只有 u2 收到
}
