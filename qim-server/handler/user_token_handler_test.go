package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/middleware"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupUserTokenTest 构造 IssueUserToken 的最小上下文：内存库 + 注入 user_id 的路由。
func setupUserTokenTest(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserToken{}))

	database.DB = db
	di.GlobalContainer = &di.Container{DB: db}

	r := gin.New()
	r.POST("/api/v1/user-tokens", func(c *gin.Context) {
		c.Set("user_id", uint(2))
		IssueUserToken(c)
	})
	return r, db
}

func postIssueToken(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user-tokens", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestIssueUserToken_RejectsMalformedExpiresDays expires_days 传错类型时
// 绝不能静默降级为"永久"——必须 400 且不签发任何令牌。
func TestIssueUserToken_RejectsMalformedExpiresDays(t *testing.T) {
	r, db := setupUserTokenTest(t)

	w := postIssueToken(r, `{"name":"x","expires_days":"30"}`)

	require.Equal(t, http.StatusBadRequest, w.Code,
		"字符串类型的 expires_days 应被 400 拒绝，而不是静默签发永久令牌")
	var count int64
	require.NoError(t, db.Model(&model.UserToken{}).Count(&count).Error)
	require.EqualValues(t, 0, count, "被拒请求不得落库任何令牌")
}

// TestIssueUserToken_ExpiringToken 有效期签发的完整行为：
// 30 天令牌 → 200、qusr_ 前缀明文、落库 ExpiresAt≈now+30d、hash 与明文对应。
func TestIssueUserToken_ExpiringToken(t *testing.T) {
	r, db := setupUserTokenTest(t)

	w := postIssueToken(r, `{"name":"ci","expires_days":30}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Token     string     `json:"token"`
			ExpiresAt *time.Time `json:"expires_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, strings.HasPrefix(resp.Data.Token, "qusr_"), "明文令牌应为 qusr_ 前缀")
	require.NotNil(t, resp.Data.ExpiresAt, "30 天令牌的 expires_at 不应为空")

	var row model.UserToken
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, middleware.HashBotToken(resp.Data.Token), row.TokenHash, "落库 hash 应与明文对应")
	require.NotNil(t, row.ExpiresAt)
	require.WithinDuration(t, row.ExpiresAt.AddDate(0, 0, -30), time.Now(), time.Minute,
		"有效期应为签发时刻 +30 天")
}

// TestIssueUserToken_PermanentWhenOmitted 缺省签发 = 永久令牌（ExpiresAt 为 NULL）。
func TestIssueUserToken_PermanentWhenOmitted(t *testing.T) {
	r, db := setupUserTokenTest(t)

	w := postIssueToken(r, `{"name":"forever"}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			ExpiresAt *time.Time `json:"expires_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Nil(t, resp.Data.ExpiresAt, "缺省 expires_days 应为永久（expires_at=null）")

	var row model.UserToken
	require.NoError(t, db.First(&row).Error)
	require.Nil(t, row.ExpiresAt)
}

// TestIssueUserToken_RangeGuard 有效天数越界与坏 JSON 均拒绝且不落库。
func TestIssueUserToken_RangeGuard(t *testing.T) {
	r, db := setupUserTokenTest(t)

	require.Equal(t, http.StatusBadRequest, postIssueToken(r, `{"expires_days":3651}`).Code)
	require.Equal(t, http.StatusBadRequest, postIssueToken(r, `{"expires_days":-1}`).Code)
	require.Equal(t, http.StatusBadRequest, postIssueToken(r, `{bad json`).Code)

	var count int64
	require.NoError(t, db.Model(&model.UserToken{}).Count(&count).Error)
	require.EqualValues(t, 0, count, "被拒请求不得落库任何令牌")
}
