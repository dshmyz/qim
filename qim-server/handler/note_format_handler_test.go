package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupFormatNoteRouter 构造 FormatNote 测试环境：内存 sqlite + DI 容器 +
// 可注入的 AI 桩 + 预置 user_id 的中间件。
func setupFormatNoteRouter(t *testing.T, aiSvc *ai.AIService) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Note{}))
	database.DB = db
	di.InitContainer(&config.Config{}, nil)
	di.GlobalContainer.AIService = aiSvc

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	r.POST("/api/v1/notes/:id/format", FormatNote)
	return r, db
}

// formatTestAI 构造带 analysis 路由的捕获型 AI 桩（GetCompletion 走
// WithModel().Chat，describeCaptureProvider 覆写了 WithModel 返回自身）。
func formatTestAI(reply string) (*ai.AIService, *describeCaptureProvider) {
	routes := map[ai.TaskType]ai.Route{
		ai.TaskTypeAnalysis: {Provider: "mock", Model: "analysis"},
	}
	aiSvc := ai.NewAIService(&ai.AIConfig{
		Router: ai.RouterConfig{
			DefaultTask: ai.TaskTypeChat,
			Routes:      routes,
		},
	})
	capProv := &describeCaptureProvider{reply: reply}
	aiSvc.SetProviderForTesting("mock", capProv)
	return aiSvc, capProv
}

func formatNoteRequest(router *gin.Engine, noteID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID+"/format", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// formatResponse 解析接口成功响应体。
type formatResponse struct {
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
}

// TestFormatNote_Success 验证正常路径：AI 桩返回格式化 Markdown 时，接口返回
// content 且 truncated=false；同时校验发给模型的最后一条 user 消息确实是
// 笔记正文（未夹带多余指令）。
func TestFormatNote_Success(t *testing.T) {
	aiSvc, capProv := formatTestAI("## 收稿日期\n\n2012-12-3\n\n## 基金\n\n本研究得到国家林业公益性行业科研专项经费")
	router, db := setupFormatNoteRouter(t, aiSvc)

	note := model.Note{UserID: 1, Title: "导入的论文", Content: "收稿日期:2012-12-3 本研究得到国家林业……"}
	require.NoError(t, db.Create(&note).Error)

	w := formatNoteRequest(router, "1")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data formatResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "## 收稿日期\n\n2012-12-3\n\n## 基金\n\n本研究得到国家林业公益性行业科研专项经费", resp.Data.Content)
	require.False(t, resp.Data.Truncated)

	// 模型收到的 user 消息是笔记正文
	require.Len(t, capProv.lastMessages, 2)
	require.Contains(t, capProv.lastMessages[1].Content, "收稿日期:2012-12-3")
	require.Contains(t, capProv.lastMessages[0].Content, "排版助手")
}

// TestFormatNote_Validations 验证各类拒绝路径。
func TestFormatNote_Validations(t *testing.T) {
	t.Run("invalid note id", func(t *testing.T) {
		aiSvc, _ := formatTestAI("x")
		router, _ := setupFormatNoteRouter(t, aiSvc)
		w := formatNoteRequest(router, "abc")
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	t.Run("note not found or not owned", func(t *testing.T) {
		aiSvc, _ := formatTestAI("x")
		router, db := setupFormatNoteRouter(t, aiSvc)
		// 笔记属于 user 2，请求方是 user 1
		require.NoError(t, db.Create(&model.Note{UserID: 2, Title: "别人的笔记", Content: "内容"}).Error)
		w := formatNoteRequest(router, "1")
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	})

	t.Run("AI service not configured", func(t *testing.T) {
		router, db := setupFormatNoteRouter(t, nil) // AIService 置 nil
		require.NoError(t, db.Create(&model.Note{UserID: 1, Title: "n", Content: "内容"}).Error)
		w := formatNoteRequest(router, "1")
		require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	})

	t.Run("mojibake content rejected", func(t *testing.T) {
		aiSvc, _ := formatTestAI("x")
		router, db := setupFormatNoteRouter(t, aiSvc)
		require.NoError(t, db.Create(&model.Note{UserID: 1, Title: "乱码笔记", Content: strings.Repeat("�", 50) + "内容"}).Error)
		w := formatNoteRequest(router, "1")
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		require.Contains(t, w.Body.String(), "乱码")
	})

	t.Run("overlong content truncated", func(t *testing.T) {
		aiSvc, capProv := formatTestAI("格式化结果")
		router, db := setupFormatNoteRouter(t, aiSvc)
		long := strings.Repeat("长", noteFormatMaxRunes+1000)
		require.NoError(t, db.Create(&model.Note{UserID: 1, Title: "长笔记", Content: long}).Error)
		w := formatNoteRequest(router, "1")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		// 发给模型的正文被截到上限（截断标记返回给前端）
		require.Len(t, []rune(capProv.lastMessages[1].Content), noteFormatMaxRunes)
		var resp struct {
			Data formatResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.True(t, resp.Data.Truncated)
	})

	t.Run("AI mojibake output rejected", func(t *testing.T) {
		aiSvc, _ := formatTestAI(strings.Repeat("�", 100))
		router, db := setupFormatNoteRouter(t, aiSvc)
		require.NoError(t, db.Create(&model.Note{UserID: 1, Title: "n", Content: "正常内容"}).Error)
		w := formatNoteRequest(router, "1")
		require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	})
}
