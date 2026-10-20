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

// TestBotWhoami_ReturnsVirtualUserID GET /bot/me 的存在意义：pull 模式 agent 要判断群里
// 一条消息的 @{mention:<virtual_user_id>} 是否指向自己，虚拟用户 id 只有 bot 自己该拿到。
// 没有这个端点就只能让用户手抄 /bots/my 的值，换环境/重建 bot 后必然漂移。
func TestBotWhoami_ReturnsVirtualUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	virtualUserID := uint(77)
	bot := &model.Bot{
		ID: 3, Name: "小助手", Type: model.BotTypeCustom,
		IsActive: true, VirtualUserID: &virtualUserID,
	}

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("bot", bot); c.Next() })
	h := NewBotAPIHandler(nil) // Whoami 不依赖 botMessaging
	r.GET("/bot/me", h.Whoami)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/bot/me", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			ID            uint   `json:"id"`
			Name          string `json:"name"`
			VirtualUserID *uint  `json:"virtual_user_id"`
			IsActive      bool   `json:"is_active"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, uint(3), resp.Data.ID)
	assert.Equal(t, "小助手", resp.Data.Name)
	require.NotNil(t, resp.Data.VirtualUserID, "virtual_user_id 必须透出")
	assert.Equal(t, uint(77), *resp.Data.VirtualUserID)
	assert.True(t, resp.Data.IsActive)
}

// TestBotWhoami_Unauthorized 未挂 BotAuthMiddleware（无 bot 身份）时应 401，不得泄漏任何身份。
func TestBotWhoami_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewBotAPIHandler(nil)
	r.GET("/bot/me", h.Whoami)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/bot/me", nil))
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "virtual_user_id")
}
