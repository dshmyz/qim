package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

const maxImageTranslateSize = 5 * 1024 * 1024 // 5MB

// TranslateTextRequest 翻译请求
type TranslateTextRequest struct {
	Text       string `json:"text" binding:"required"`
	TargetLang string `json:"target_lang" binding:"required"` // zh/en/ja/ko/fr/de
	SourceLang string `json:"source_lang"`                    // auto/zh/en/...
}

// RewriteTextRequest 改写请求
type RewriteTextRequest struct {
	Text  string `json:"text" binding:"required"`
	Style string `json:"style"` // formal/casual/concise/detailed
	Tone  string `json:"tone"`  // professional/friendly/neutral
}

// PolishTextRequest 润色请求
type PolishTextRequest struct {
	Text     string `json:"text" binding:"required"`
	Language string `json:"language"` // zh/en
}

// TranslateImageRequest 图片翻译请求
type TranslateImageRequest struct {
	ImageURL   string `json:"image_url" binding:"required"`
	TargetLang string `json:"target_lang"` // 默认 zh
}

// extractImageURL 从消息内容中提取图片 URL
// 支持：纯 URL、JSON 格式 {"url": "...", ...}
func extractImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// 尝试解析为 JSON
	if strings.HasPrefix(raw, "{") {
		var imgData map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &imgData); err == nil {
			if url, ok := imgData["url"].(string); ok {
				return url
			}
		}
	}
	return raw
}

// imageToDataURL 将图片 URL/路径转为 base64 data URL 或可用 URL
func imageToDataURL(imageURL string) string {
	// 已经是 data URL，直接使用
	if strings.HasPrefix(imageURL, "data:") {
		return imageURL
	}

	// 处理 http(s) URL
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		// 尝试从本地 uploads 目录读取
		if idx := strings.Index(imageURL, "/uploads/"); idx != -1 {
			localPath := imageURL[idx+1:] // "uploads/xxx"
			if dataURL := readFileAsDataURL(localPath); dataURL != "" {
				return dataURL
			}
			// 本地文件不存在，尝试从 HTTP 下载
			if dataURL := downloadAsDataURL(imageURL); dataURL != "" {
				return dataURL
			}
		}
		// 外部 URL，直接返回让 AI 访问
		return imageURL
	}

	// 以 /uploads/ 开头，从本地读取
	if strings.HasPrefix(imageURL, "/uploads/") {
		if dataURL := readFileAsDataURL(imageURL[1:]); dataURL != "" {
			return dataURL
		}
	}

	// 尝试直接作为本地路径读取（兼容相对路径）
	if dataURL := readFileAsDataURL(imageURL); dataURL != "" {
		return dataURL
	}

	// 非本地 URL 原样返回
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL
	}
	return ""
}

// readFileAsDataURL 从本地文件系统读取图片转为 data URL
func readFileAsDataURL(relPath string) string {
	// 清理路径
	relPath = strings.TrimPrefix(relPath, "./")
	fullPath := filepath.Clean(filepath.Join(".", relPath))

	// 安全检查：确保路径在 uploads 目录内
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return ""
	}
	uploadsDir, _ := filepath.Abs("./uploads")
	if !strings.HasPrefix(absPath, uploadsDir) {
		return ""
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}

	ext := filepath.Ext(absPath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// downloadAsDataURL 从 HTTP URL 下载图片转为 data URL（最大 5MB）
func downloadAsDataURL(url string) string {
	client := &http.Client{Timeout: http.DefaultClient.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageTranslateSize))
	if err != nil {
		return ""
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// TranslateImage 图片翻译（AI 视觉识别 + 翻译）
func (h *AIHandler) TranslateImage(c *gin.Context) {
	var req TranslateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	targetLang := req.TargetLang
	if targetLang == "" {
		targetLang = "zh"
	}
	langName := map[string]string{
		"zh": "中文", "en": "英文", "ja": "日文", "ko": "韩文",
		"fr": "法文", "de": "德文",
	}[targetLang]
	if langName == "" {
		langName = "中文"
	}

	// 解析 image_url：可能是 JSON 格式（如 {"url": "/uploads/...", ...}）
	imageURL := extractImageURL(req.ImageURL)

	dataURL := imageToDataURL(imageURL)
	if dataURL == "" {
		response.BadRequest(c, fmt.Sprintf("不支持的图片地址格式: %s", imageURL))
		return
	}
	if len(dataURL) > maxImageTranslateSize {
		response.BadRequest(c, fmt.Sprintf("图片过大，最大支持%dMB", maxImageTranslateSize/(1024*1024)))
		return
	}

	// 检查是否有可用的视觉模型
	visionProviders := []string{"openai", "anthropic"}
	config := h.aiService.GetConfig()
	var visionModelName string
	for _, name := range visionProviders {
		providerCfg, ok := config.AllProviders()[name]
		if ok && providerCfg.Model != "" {
			visionModelName = fmt.Sprintf("%s (%s)", providerCfg.Model, name)
			break
		}
	}
	if visionModelName == "" {
		response.BadRequest(c, "图片翻译需要配置支持视觉的 AI 模型（如 OpenAI GPT-4、Anthropic Claude），当前未配置可用模型")
		return
	}

	systemPrompt := fmt.Sprintf(`你是一个图片翻译助手。请完成以下步骤：

1. 仔细识别图片中的所有文字内容
2. 如果图片中没有可识别的文字，直接返回 {"original_text": "未检测到文字", "translated_text": ""}
3. 如果有文字，将识别到的内容翻译成%s

请严格按以下 JSON 格式输出，不要包含任何其他内容：
{"original_text": "识别的文字", "translated_text": "翻译结果"}

注意：如果图片中确实没有文字，translated_text 必须为空字符串。不要编造文字。`, langName)

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "请识别这张图片中的文字并翻译成" + langName, ImageURL: dataURL},
	}

	// 使用已找到的视觉模型
	var override *ai.Override
	for _, name := range visionProviders {
		providerCfg, ok := config.AllProviders()[name]
		if ok && providerCfg.Model != "" {
			override = &ai.Override{
				TaskType: ai.TaskTypeVision,
				Provider: name,
				Model:    providerCfg.Model,
			}
			break
		}
	}
	if override == nil {
		response.InternalServerError(c, "图片翻译失败：无可用视觉模型")
		return
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeVision, messages_input, *override)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片翻译失败: " + err.Error()})
		return
	}

	// 解析 JSON 格式响应，提取译文
	var parsed struct {
		OriginalText   string `json:"original_text"`
		TranslatedText string `json:"translated_text"`
	}
	jsonParsed := false
	jsonResult := map[string]interface{}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result)), &jsonResult); err == nil {
		if t, ok := jsonResult["translated_text"].(string); ok {
			parsed.TranslatedText = t
			jsonParsed = true
		}
		if o, ok := jsonResult["original_text"].(string); ok {
			parsed.OriginalText = o
		}
	}

	if !jsonParsed {
		// AI 未按 JSON 格式返回（可能是在闲聊），尝试提取
		// 回退：尝试从 【译文】 标签提取
		if idx := strings.Index(result, "【译文】"); idx != -1 {
			parsed.TranslatedText = strings.TrimSpace(result[idx+len("【译文】"):])
		} else if idx := strings.Index(result, "翻译"); idx != -1 {
			// 尝试从含"翻译"关键词的行提取
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				if strings.Contains(line, "翻译") || strings.Contains(line, "译文") {
					parsed.TranslatedText = strings.TrimSpace(line)
					break
				}
			}
			if parsed.TranslatedText == "" {
				parsed.TranslatedText = result
			}
		} else {
			// 纯闲聊，说明图片中可能没有可识别文字
			parsed.OriginalText = "未检测到可翻译文字"
			parsed.TranslatedText = ""
		}
	}

	if parsed.TranslatedText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "图片中未检测到可翻译的文字内容"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"translated_text": parsed.TranslatedText,
			"original_text":   parsed.OriginalText,
			"target_lang":     targetLang,
		},
	})
}

// TranslateText 翻译文本
func (h *AIHandler) TranslateText(c *gin.Context) {
	var req TranslateTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	if h.textProcessGraph != nil {
		sourceLang := req.SourceLang
		if sourceLang == "" || sourceLang == "auto" {
			sourceLang = "自动检测"
		}

		input := &service.TextProcessInput{
			Intent:     service.TextProcessTranslate,
			Text:       req.Text,
			TargetLang: req.TargetLang,
			SourceLang: sourceLang,
		}

		ctx := c.Request.Context()
		result, err := h.textProcessGraph.Execute(ctx, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "翻译失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"translated_text": result.Result,
				"source_lang":     sourceLang,
				"target_lang":     req.TargetLang,
			},
		})
		return
	}

	sourceLang := req.SourceLang
	if sourceLang == "" || sourceLang == "auto" {
		sourceLang = "自动检测"
	}

	promptCtx := &service.PromptContext{
		SourceLang:   sourceLang,
		TargetLang:   req.TargetLang,
	}
	systemPrompt := service.NewPromptManager().BuildSystemPrompt(service.SceneTranslate, promptCtx)

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "翻译失败: " + err.Error()})
		return
	}

	result = h.aiService.FilterOutput(result, "ai_translate")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"translated_text": result,
			"source_lang":     sourceLang,
			"target_lang":     req.TargetLang,
		},
	})
}

// RewriteText 改写文本
func (h *AIHandler) RewriteText(c *gin.Context) {
	var req RewriteTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	if h.textProcessGraph != nil {
		style := req.Style
		if style == "" {
			style = "简洁"
		}
		tone := req.Tone
		if tone == "" {
			tone = "专业"
		}

		input := &service.TextProcessInput{
			Intent: service.TextProcessRewrite,
			Text:   req.Text,
			Style:  style,
			Tone:   tone,
		}

		ctx := c.Request.Context()
		result, err := h.textProcessGraph.Execute(ctx, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "改写失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"rewritten_text": result.Result,
			},
		})
		return
	}

	style := req.Style
	if style == "" {
		style = "简洁"
	}
	tone := req.Tone
	if tone == "" {
		tone = "专业"
	}

	promptCtx := &service.PromptContext{
		Style: style,
		Tone:  tone,
	}
	systemPrompt := service.NewPromptManager().BuildSystemPrompt(service.SceneRewrite, promptCtx)

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "改写失败: " + err.Error()})
		return
	}

	result = h.aiService.FilterOutput(result, "ai_rewrite")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"rewritten_text": result,
		},
	})
}

// PolishText 润色文本
func (h *AIHandler) PolishText(c *gin.Context) {
	var req PolishTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	if h.textProcessGraph != nil {
		lang := req.Language
		if lang == "" {
			lang = "中文"
		}

		input := &service.TextProcessInput{
			Intent:   service.TextProcessPolish,
			Text:     req.Text,
			Language: lang,
		}

		ctx := c.Request.Context()
		result, err := h.textProcessGraph.Execute(ctx, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "润色失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
			"data": gin.H{
				"polished_text": result.Result,
			},
		})
		return
	}

	lang := req.Language
	if lang == "" {
		lang = "中文"
	}

	promptCtx := &service.PromptContext{
		Language: lang,
	}
	systemPrompt := service.NewPromptManager().BuildSystemPrompt(service.ScenePolish, promptCtx)

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: req.Text},
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "润色失败: " + err.Error()})
		return
	}

	result = h.aiService.FilterOutput(result, "ai_polish")

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"polished_text": result,
		},
	})
}
