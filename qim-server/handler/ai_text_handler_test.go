package handler

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadAsDataURLRejectsOversizedImageWithoutTruncating(t *testing.T) {
	payload := strings.Repeat("a", maxImageTranslateSize+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)

	if got := downloadAsDataURL(srv.URL); got != "" {
		t.Fatalf("oversized image should be rejected, got data URL length %d", len(got))
	}
}

func TestDownloadAsDataURLSuccess(t *testing.T) {
	png := bytes.Repeat([]byte{0x89, 0x50, 0x4E, 0x47}, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	t.Cleanup(srv.Close)

	got := downloadAsDataURL(srv.URL)
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("expected data URL with image/png, got %q", got[:min(len(got), 40)])
	}
}

func TestDownloadAsDataURLHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	if got := downloadAsDataURL(srv.URL); got != "" {
		t.Fatalf("non-200 should return empty, got %q", got)
	}
}

func TestDownloadAsDataURLMissingContentType(t *testing.T) {
	// PNG 魔数开头，http.DetectContentType 应识别为 image/png
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 故意不设置 Content-Type
		_, _ = w.Write(png)
	}))
	t.Cleanup(srv.Close)

	got := downloadAsDataURL(srv.URL)
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("missing Content-Type should fall back to DetectContentType, got %q", got[:min(len(got), 40)])
	}
}

func TestReadImageWithLimitAtBoundary(t *testing.T) {
	// 恰好等于上限，应成功（不 tooLarge）
	data := bytes.Repeat([]byte("x"), maxImageTranslateSize)
	got, tooLarge, err := readImageWithLimit(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tooLarge {
		t.Fatal("data at boundary should not be too large")
	}
	if len(got) != maxImageTranslateSize {
		t.Fatalf("expected %d bytes, got %d", maxImageTranslateSize, len(got))
	}
}

func TestReadImageWithLimitEmpty(t *testing.T) {
	got, tooLarge, err := readImageWithLimit(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tooLarge {
		t.Fatal("empty input should not be too large")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 bytes, got %d", len(got))
	}
}

func TestDataURLTooLargeNoComma(t *testing.T) {
	// 无逗号：直接按总长度判断
	small := "data:text/plainabcdefgh"
	if dataURLTooLarge(small) {
		t.Fatal("small dataURL without comma should not be too large")
	}
	huge := "data:" + strings.Repeat("a", maxImageTranslateSize+1)
	if !dataURLTooLarge(huge) {
		t.Fatal("huge dataURL without comma should be too large")
	}
}

func TestDataURLTooLargeBase64(t *testing.T) {
	// base64 payload 解码后超过上限 → too large
	payload := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("x"), maxImageTranslateSize+1))
	dataURL := "data:image/png;base64," + payload
	if !dataURLTooLarge(dataURL) {
		t.Fatal("base64 dataURL with decoded size over limit should be too large")
	}

	// base64 payload 解码后未超过上限 → not too large
	smallPayload := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("x"), 100))
	smallDataURL := "data:image/png;base64," + smallPayload
	if dataURLTooLarge(smallDataURL) {
		t.Fatal("small base64 dataURL should not be too large")
	}
}

func TestDataURLTooLargePlain(t *testing.T) {
	// 非 base64 的 dataURL：按 payload 原始长度判断
	small := "data:text/plain,hello"
	if dataURLTooLarge(small) {
		t.Fatal("small plain dataURL should not be too large")
	}
	huge := "data:text/plain," + strings.Repeat("a", maxImageTranslateSize+1)
	if !dataURLTooLarge(huge) {
		t.Fatal("huge plain dataURL should be too large")
	}
}

// describeCaptureProvider 测试用视觉 Provider 桩：复用 mockToolProvider 的 no-op 实现，
// Chat 捕获最近一次收到的 ai.Message（断言 ImageURL 透传）并按 reply 返回固定描述。
type describeCaptureProvider struct {
	mockToolProvider
	reply        string
	lastMessages []ai.Message
}

func (p *describeCaptureProvider) Chat(messages []ai.Message) (string, error) {
	p.lastMessages = messages
	return p.reply, nil
}

// WithModel 覆盖 mockToolProvider 的实现：默认返回被嵌入的裸 mockToolProvider
// （Chat 恒返回 "ok" 且不记录消息）。GetCompletion 带 model 路由时走
// provider.WithModel(model).Chat(...)，这里返回自身以命中上面的消息捕获。
func (p *describeCaptureProvider) WithModel(model string) ai.Provider {
	return p
}

// newDescribeImageTestAI 构造带视觉路由（可选）的 AIHandler，注入捕获型视觉 provider。
func newDescribeImageTestAI(routeVision bool, reply string) (*AIHandler, *describeCaptureProvider) {
	routes := map[ai.TaskType]ai.Route{
		ai.TaskTypeChat: {Provider: "mock", Model: "chat"},
	}
	if routeVision {
		routes[ai.TaskTypeVision] = ai.Route{Provider: "mock", Model: "vision"}
	}
	aiSvc := ai.NewAIService(&ai.AIConfig{
		Router: ai.RouterConfig{
			DefaultTask: ai.TaskTypeChat,
			Routes:      routes,
		},
	})
	capProv := &describeCaptureProvider{reply: reply}
	aiSvc.SetProviderForTesting("mock", capProv)
	return NewAIHandler(aiSvc, nil), capProv
}

func describeImageRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/describe-image", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestDescribeImage_Success 验证图片识别正常路径：provider 返回描述 JSON 时，端点返回
// 描述文本，且透传给模型的最后一条 user 消息携带图片 base64 data URL。
func TestDescribeImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, capProv := newDescribeImageTestAI(true, `{"description":"一只猫蹲在窗台上"}`)
	router := gin.New()
	router.POST("/describe-image", h.DescribeImage)

	w := describeImageRequest(router, `{"image_url":"data:image/png;base64,aW1n"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "一只猫蹲在窗台上")

	var lastUser ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			lastUser = m
		}
	}
	assert.Equal(t, "data:image/png;base64,aW1n", lastUser.ImageURL, "DescribeImage 应把图片 data URL 交给视觉模型")
	assert.Contains(t, lastUser.Content, "识别图片内容并详细描述", "未传 instruction 时应用默认识别指令")
}

// TestDescribeImage_CustomInstruction 验证自定义 instruction 透传。
func TestDescribeImage_CustomInstruction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, capProv := newDescribeImageTestAI(true, `{"description":"图中有一份体检报告"}`)
	router := gin.New()
	router.POST("/describe-image", h.DescribeImage)

	w := describeImageRequest(router, `{"image_url":"data:image/png;base64,aW1n","instruction":"请提取体检报告中的关键指标"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var lastUser ai.Message
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			lastUser = m
		}
	}
	assert.Contains(t, lastUser.Content, "请提取体检报告中的关键指标", "自定义 instruction 应透传给模型")
}

// TestDescribeImage_NoVisionRoute 验证未配置「视觉理解」路由时返回 400，
// 而不是把图片 base64 硬塞给不支持视觉的模型。
func TestDescribeImage_NoVisionRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newDescribeImageTestAI(false, `{"description":"x"}`)
	router := gin.New()
	router.POST("/describe-image", h.DescribeImage)

	w := describeImageRequest(router, `{"image_url":"data:image/png;base64,aW1n"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "视觉理解", "应提示配置视觉任务路由")
}

// TestDescribeImage_EmptyDescription 验证模型返回空文本时返回 400 明确文案。
func TestDescribeImage_EmptyDescription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newDescribeImageTestAI(true, "")
	router := gin.New()
	router.POST("/describe-image", h.DescribeImage)

	w := describeImageRequest(router, `{"image_url":"data:image/png;base64,aW1n"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "未能识别图片内容")
}
