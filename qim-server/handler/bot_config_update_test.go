package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupBotConfigDB 建内存库，迁移 Bot / 操作日志（UpdateBotConfig 结尾写审计日志），
// 并注入 GlobalContainer：否则 handler 取 di.GlobalContainer.OperationLogService 时
// GlobalContainer 为 nil，取字段直接 panic（nil 指针解引用）。
func setupBotConfigDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Bot{}, &model.OperationLog{}))
	database.DB = db

	previousContainer := di.GlobalContainer
	di.GlobalContainer = &di.Container{
		DB:                  db,
		OperationLogService: service.NewOperationLogService(db),
	}
	t.Cleanup(func() { di.GlobalContainer = previousContainer })
	return db
}

// newBotConfigRouter 挂 UpdateBotConfig，以 userID 身份鉴权（创建者本人）。
func newBotConfigRouter(t *testing.T, userID uint) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("roles", []string{})
		c.Next()
	})
	h := NewBotAPIHandler(nil)
	r.PUT("/bots/:id/config", h.UpdateBotConfig)
	return r
}

func putBotConfig(t *testing.T, r *gin.Engine, botID uint, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/bots/%d/config", botID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestUpdateBotConfig_MergeSemantics 锚定合并语义（曾因结构体缺字段连续出过三个 bug）：
// 1) use_creator_notes 前端发了必须落库（此前请求结构体没有该字段，开关保存不生效）；
// 2) webhook_url 显式空串=清空（切换 pull 模式），nil=不修改；
// 3) 未提交的字段（system_prompt/模型来源）合并后原样保留。
func TestUpdateBotConfig_MergeSemantics(t *testing.T) {
	db := setupBotConfigDB(t)
	virtualUser := &model.User{Username: "vbotcfg", PasswordHash: "h", Nickname: "V", Type: "bot"}
	require.NoError(t, db.Create(virtualUser).Error)
	bot := &model.Bot{
		Name:         "B",
		Type:         model.BotTypeAssistant,
		IsActive:     true,
		CreatorID:     1,
		VirtualUserID: &virtualUser.ID,
		Config:        `{"mode":"internal_ai","system_prompt":"你是测试助手","use_system_config":true,"use_creator_notes":true,"webhook_url":"https://old.example/hook","webhook_secret":"keep-me"}`,
	}
	require.NoError(t, db.Create(bot).Error)

	r := newBotConfigRouter(t, 1)

	// external 模式保存：不带 use_creator_notes / 模型字段，清空 webhook_url
	w := putBotConfig(t, r, bot.ID, `{"mode":"external_webhook","webhook_url":""}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored model.Bot
	require.NoError(t, db.First(&stored, bot.ID).Error)
	assert.Contains(t, stored.Config, `"system_prompt":"你是测试助手"`, "未提交的 system_prompt 应保留")
	assert.Contains(t, stored.Config, `"use_system_config":true`, "未提交的模型来源应保留")
	assert.Contains(t, stored.Config, `"use_creator_notes":true`, "external 模式未提交的知识库开关应保留")
	assert.Contains(t, stored.Config, `"webhook_secret":"keep-me"`, "空 secret 表示不修改，应保留")
	assert.NotContains(t, stored.Config, "old.example", "显式空串应清空 webhook_url（切换 pull 模式）")

	// 切回 internal 并显式关掉知识库开关：新值必须落库（此前被静默丢弃）
	w = putBotConfig(t, r, bot.ID, `{"mode":"internal_ai","use_creator_notes":false}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, db.First(&stored, bot.ID).Error)
	assert.Contains(t, stored.Config, `"use_creator_notes":false`, "显式提交的 use_creator_notes 必须落库")
	assert.Contains(t, stored.Config, `"system_prompt":"你是测试助手"`, "system_prompt 在多次合并后仍应保留")
}

// TestUpdateBotConfig_NonOwnerForbidden 非创建者/非管理员不可改配置。
func TestUpdateBotConfig_NonOwnerForbidden(t *testing.T) {
	db := setupBotConfigDB(t)
	virtualUser := &model.User{Username: "vbotcfg2", PasswordHash: "h", Nickname: "V", Type: "bot"}
	require.NoError(t, db.Create(virtualUser).Error)
	bot := &model.Bot{Name: "B2", Type: model.BotTypeAssistant, IsActive: true, CreatorID: 1, VirtualUserID: &virtualUser.ID}
	require.NoError(t, db.Create(bot).Error)

	r := newBotConfigRouter(t, 999)
	w := putBotConfig(t, r, bot.ID, `{"mode":"internal_ai"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
