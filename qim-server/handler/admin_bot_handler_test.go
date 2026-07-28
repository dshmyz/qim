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

func setupAdminBotRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Bot{},
		&model.BotWebhookDelivery{},
	))

	database.DB = db
	di.InitContainer(&config.Config{}, nil)

	r := gin.New()
	admin := r.Group("/api/v1/admin")
	{
		admin.GET("/bots/external", AdminGetExternalBots)
		admin.GET("/webhook-deliveries", AdminGetWebhookDeliveries)
		admin.GET("/webhook-deliveries/:id", AdminGetWebhookDelivery)
		admin.POST("/webhook-deliveries/:id/redeliver", AdminRedeliverWebhook)
	}
	return r, db
}

func externalBotConfig(url string) string {
	return fmt.Sprintf(`{"mode":"external_webhook","webhook_url":%q,"webhook_secret":"shh"}`, url)
}

func TestAdminGetExternalBots_FiltersInternalBotsAndCountsDeliveries(t *testing.T) {
	r, db := setupAdminBotRouter(t)

	// 一个外部 bot + 一个内部 AI bot（应被过滤掉）
	require.NoError(t, db.Create(&model.Bot{
		Name: "agent-bot", Type: "custom", IsActive: true,
		Config: externalBotConfig("http://agent/cb"), CreatorName: "alice",
	}).Error)
	require.NoError(t, db.Create(&model.Bot{
		Name: "internal-ai", Type: "assistant", IsActive: true,
		Config: `{"mode":"internal_ai"}`,
	}).Error)

	// 给外部 bot 制造投递记录：2 pending + 1 dead + 1 done
	bot := model.Bot{}
	db.First(&bot, "name = ?", "agent-bot")
	require.NoError(t, db.Create(&model.BotWebhookDelivery{BotID: bot.ID, Event: "bot.message", Payload: "{}", WebhookURL: "http://agent/cb", Status: "pending"}).Error)
	require.NoError(t, db.Create(&model.BotWebhookDelivery{BotID: bot.ID, Event: "bot.message", Payload: "{}", WebhookURL: "http://agent/cb", Status: "pending"}).Error)
	require.NoError(t, db.Create(&model.BotWebhookDelivery{BotID: bot.ID, Event: "bot.card_action", Payload: "{}", WebhookURL: "http://agent/cb", Status: "dead", LastError: "timeout"}).Error)
	require.NoError(t, db.Create(&model.BotWebhookDelivery{BotID: bot.ID, Event: "bot.message", Payload: "{}", WebhookURL: "http://agent/cb", Status: "done"}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/bots/external?page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			List  []map[string]any `json:"list"`
			Total int64            `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, int64(1), body.Data.Total, "应只列出 1 个外部 bot")
	require.Len(t, body.Data.List, 1)
	row := body.Data.List[0]
	assert.Equal(t, "agent-bot", row["name"])
	assert.Equal(t, "external_webhook", row["mode"])
	assert.Equal(t, "http://agent/cb", row["webhook_url"])
	// 脱敏：不应泄露 secret
	assert.Nil(t, row["webhook_secret"])
	assert.EqualValues(t, 2, row["pending_count"])
	assert.EqualValues(t, 1, row["dead_count"])
}

func TestAdminGetWebhookDeliveries_FiltersAndPagination(t *testing.T) {
	r, db := setupAdminBotRouter(t)
	require.NoError(t, db.Create(&model.Bot{Name: "b1", Type: "custom", Config: externalBotConfig("http://x")}).Error)
	var b1 model.Bot
	db.First(&b1, "name = ?", "b1")

	// 3 dead + 2 pending
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&model.BotWebhookDelivery{BotID: b1.ID, Event: "bot.message", Payload: "{}", Status: "dead"}).Error)
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&model.BotWebhookDelivery{BotID: b1.ID, Event: "bot.card_action", Payload: "{}", Status: "pending"}).Error)
	}

	// status=dead 过滤
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/webhook-deliveries?status=dead&page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var body struct {
		Data struct {
			List  []map[string]any `json:"list"`
			Total int64            `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, int64(3), body.Data.Total)
	assert.Len(t, body.Data.List, 3)
	// bot 名应被 join 进来
	assert.Equal(t, "b1", body.Data.List[0]["bot_name"])
}

func TestAdminGetWebhookDelivery_FullPayload(t *testing.T) {
	r, db := setupAdminBotRouter(t)
	d := &model.BotWebhookDelivery{
		BotID: 1, Event: "bot.message", Payload: `{"content":"full-payload-here","thread_id":42}`,
		WebhookURL: "http://x", Status: "dead", LastError: "conn refused",
	}
	require.NoError(t, db.Create(d).Error)
	require.NoError(t, db.Create(&model.Bot{Name: "agent", Type: "custom", Config: externalBotConfig("http://x")}).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/webhook-deliveries/%d", d.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	// 完整 payload（非截断预览）
	delivery := body.Data["delivery"].(map[string]any)
	assert.Equal(t, `{"content":"full-payload-here","thread_id":42}`, delivery["payload"])
	assert.Equal(t, "agent", body.Data["bot_name"])
}

func TestAdminRedeliverWebhook_DeadBecomesRetried(t *testing.T) {
	r, db := setupAdminBotRouter(t)
	// 用一个 503 的 mock webhook 让重投失败 -> 应回到 pending + attempts=1 + next_retry 设置
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	t.Cleanup(srv.Close)

	d := &model.BotWebhookDelivery{
		BotID: 1, Event: "bot.message", Payload: `{"content":"hi"}`,
		WebhookURL: srv.URL, WebhookSecret: "s", Status: "dead", Attempts: 4,
	}
	require.NoError(t, db.Create(d).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/webhook-deliveries/%d/redeliver", d.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// 库里应已重置并尝试投递：dead -> pending，attempts 归零后 +1
	var after model.BotWebhookDelivery
	require.NoError(t, db.First(&after, d.ID).Error)
	assert.Equal(t, "pending", after.Status)
	assert.Equal(t, 1, after.Attempts)
	assert.NotNil(t, after.NextRetryAt)
}

func TestAdminRedeliverWebhook_DoneRejected(t *testing.T) {
	r, db := setupAdminBotRouter(t)
	d := &model.BotWebhookDelivery{
		BotID: 1, Event: "bot.message", Payload: "{}", Status: "done",
	}
	require.NoError(t, db.Create(d).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/webhook-deliveries/%d/redeliver", d.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code, "已成功的投递不可重投")
}
