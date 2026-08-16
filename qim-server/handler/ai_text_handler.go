package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/service/storage"

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

// DescribeImageRequest 图片识别/描述请求
type DescribeImageRequest struct {
	ImageURL    string `json:"image_url" binding:"required"`
	Instruction string `json:"instruction"` // 可选：自定义识别指令，默认"识别图片内容并详细描述"
}

// TextProcessRequest 统一文本处理请求
type TextProcessRequest struct {
	Action     string `json:"action" binding:"required"` // translate/rewrite/polish
	Text       string `json:"text" binding:"required"`
	TargetLang string `json:"target_lang"`
	SourceLang string `json:"source_lang"`
	Style      string `json:"style"`
	Tone       string `json:"tone"`
	Language   string `json:"language"`
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

	// 提取 StoragePath（/uploads/ 或 /s3/ 前缀，含 http URL 内嵌），走存储抽象读取
	if sp := extractStoragePath(imageURL); sp != "" {
		if dataURL := storageReadAsDataURL(sp); dataURL != "" {
			return dataURL
		}
		// 存储读取失败，http URL 尝试 HTTP 下载降级
		if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
			if dataURL := downloadAsDataURL(imageURL); dataURL != "" {
				return dataURL
			}
		}
	}

	// 外部 http URL（非本站存储），直接返回让 AI 访问
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL
	}
	return ""
}

// extractStoragePath 从 imageURL 中提取 StoragePath：
// "/static/xxx" / "/uploads/xxx" / "/s3/xxx" → 原样；"http://host/static/xxx" → "/static/xxx"
func extractStoragePath(imageURL string) string {
	for _, prefix := range []string{storage.StaticPrefix, "/s3/", "/uploads/"} {
		if idx := strings.Index(imageURL, prefix); idx != -1 {
			return imageURL[idx:]
		}
	}
	return ""
}

// storageReadAsDataURL 通过存储抽象读取文件，转为 data URL（受 maxImageTranslateSize 限制）
func storageReadAsDataURL(storagePath string) string {
	mgr := di.GlobalContainer.StorageManager
	if mgr == nil {
		return ""
	}
	st, key, ok := mgr.ByPath(storagePath)
	if !ok || st == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	reader, err := st.Get(ctx, key)
	if err != nil {
		return ""
	}
	defer reader.Close()

	data, tooLarge, err := readImageWithLimit(reader)
	if err != nil || tooLarge {
		return ""
	}
	contentType := mime.TypeByExtension(filepath.Ext(storagePath))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// downloadAsDataURL 从 HTTP URL 下载图片转为 data URL（最大 5MB）
func downloadAsDataURL(url string) string {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	data, tooLarge, err := readImageWithLimit(resp.Body)
	if err != nil || tooLarge {
		return ""
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func readImageWithLimit(reader io.Reader) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(reader, int64(maxImageTranslateSize)+1))
	if err != nil {
		return nil, false, err
	}
	if len(data) > maxImageTranslateSize {
		return nil, true, nil
	}
	return data, false, nil
}

func dataURLTooLarge(dataURL string) bool {
	comma := strings.Index(dataURL, ",")
	if comma == -1 {
		return len(dataURL) > maxImageTranslateSize
	}
	header := dataURL[:comma]
	payload := strings.TrimSpace(dataURL[comma+1:])
	if strings.Contains(strings.ToLower(header), ";base64") {
		decodedLen := base64.StdEncoding.DecodedLen(len(payload))
		if strings.HasSuffix(payload, "==") {
			decodedLen -= 2
		} else if strings.HasSuffix(payload, "=") {
			decodedLen--
		}
		return decodedLen > maxImageTranslateSize
	}
	return len(payload) > maxImageTranslateSize
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
	if strings.HasPrefix(dataURL, "data:") && dataURLTooLarge(dataURL) {
		response.BadRequest(c, fmt.Sprintf("图片过大，最大支持%dMB", maxImageTranslateSize/(1024*1024)))
		return
	}

	// 视觉任务走「视觉理解」路由（TaskTypeVision）。未显式配置独立视觉路由时
	// 会回退到 defaultTask（通常是纯文本 chat 模型），把图片 base64 发给它必然 400，
	// 因此这里要求必须显式配置视觉路由，否则诚实提示而不是硬塞给不支持的模型。
	if !h.aiService.HasVisionRoute() {
		response.BadRequest(c, "图片翻译需要配置「视觉理解」任务路由（管理后台 → AI 模型配置 → 模型路由），当前未配置可用的视觉模型")
		return
	}

	systemPrompt := fmt.Sprintf(`%s

你是一个图片翻译助手。请完成以下步骤：

1. 仔细识别图片中的所有文字内容
2. 如果图片中没有可识别的文字，直接返回 {"original_text": "未检测到文字", "translated_text": ""}
3. 如果有文字，将识别到的内容翻译成%s

请严格按以下 JSON 格式输出，不要包含任何其他内容：
{"original_text": "识别的文字", "translated_text": "翻译结果"}

注意：如果图片中确实没有文字，translated_text 必须为空字符串。不要编造文字。`, aiprompt.CurrentTimeLine(), langName)

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "请识别这张图片中的文字并翻译成" + langName, ImageURL: dataURL},
	}

	// 通过路由选择视觉 Provider / 模型（与群聊引用图片等视觉路径一致），
	// 不传 Override，由 ModelRouter 按「视觉理解」路由解析即可。
	result, err := h.aiService.GetCompletion(ai.TaskTypeVision, messages_input)
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

// DescribeImage 图片识别/描述（AI 视觉理解）。
// 复用图片翻译同一套视觉链路：extractImageURL（支持消息 content JSON）+ imageToDataURL
// （存储→base64 data URL，5MB 护栏）+「视觉理解」路由（TaskTypeVision）门控。
func (h *AIHandler) DescribeImage(c *gin.Context) {
	var req DescribeImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	// 解析 image_url：可能是 JSON 格式（如 {"url": "/uploads/...", ...}）
	imageURL := extractImageURL(req.ImageURL)

	dataURL := imageToDataURL(imageURL)
	if dataURL == "" {
		response.BadRequest(c, fmt.Sprintf("不支持的图片地址格式: %s", imageURL))
		return
	}
	if strings.HasPrefix(dataURL, "data:") && dataURLTooLarge(dataURL) {
		response.BadRequest(c, fmt.Sprintf("图片过大，最大支持%dMB", maxImageTranslateSize/(1024*1024)))
		return
	}

	// 与图片翻译一致：要求显式配置「视觉理解」路由，避免把 base64 硬塞给不支持视觉的模型
	if !h.aiService.HasVisionRoute() {
		response.BadRequest(c, "图片识别需要配置「视觉理解」任务路由（管理后台 → AI 模型配置 → 模型路由），当前未配置可用的视觉模型")
		return
	}

	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		instruction = "识别图片内容并详细描述"
	}

	systemPrompt := fmt.Sprintf(`%s

你是一个图片识别助手。请基于图片内容完成用户指定的任务（识别/描述/提取信息等）。
请严格按以下 JSON 格式输出，不要包含任何其他内容：
{"description": "对图片的识别/描述结果"}

注意：只输出图片中实际存在的信息，不要编造。`, aiprompt.CurrentTimeLine())

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: instruction, ImageURL: dataURL},
	}

	// 通过路由选择视觉 Provider / 模型，与图片翻译一致不传 Override
	result, err := h.aiService.GetCompletion(ai.TaskTypeVision, messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片识别失败: " + err.Error()})
		return
	}

	// 解析 JSON 格式响应，提取描述；未按 JSON 返回时直接取全文（识别/描述可容忍自由文本）
	var parsed struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result)), &parsed); err != nil || parsed.Description == "" {
		parsed.Description = strings.TrimSpace(result)
	}

	if parsed.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未能识别图片内容"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"description": parsed.Description,
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

	h.runTextProcess(c, translateParams(req.Text, req.TargetLang, req.SourceLang), service.TextProcessTranslate)
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

	h.runTextProcess(c, rewriteParams(req.Text, req.Style, req.Tone), service.TextProcessRewrite)
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

	h.runTextProcess(c, polishParams(req.Text, req.Language), service.TextProcessPolish)
}

// TextProcess 统一文本处理入口（翻译/改写/润色）
func (h *AIHandler) TextProcess(c *gin.Context) {
	var req TextProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.aiService.IsConfigured() {
		response.InternalServerError(c, "AI服务未配置")
		return
	}

	switch req.Action {
	case "translate":
		h.runTextProcess(c, translateParams(req.Text, req.TargetLang, req.SourceLang), service.TextProcessTranslate)
	case "rewrite":
		h.runTextProcess(c, rewriteParams(req.Text, req.Style, req.Tone), service.TextProcessRewrite)
	case "polish":
		h.runTextProcess(c, polishParams(req.Text, req.Language), service.TextProcessPolish)
	default:
		response.BadRequest(c, "不支持的 action")
	}
}

// textProcessParams 已规范化的文本处理参数（翻译/改写/润色三种意图共用）。
type textProcessParams struct {
	text       string
	targetLang string
	sourceLang string
	style      string
	tone       string
	language   string
}

// promptContext 兜底直调模式用的 PromptContext。
func (p *textProcessParams) promptContext() *service.PromptContext {
	return &service.PromptContext{
		SourceLang: p.sourceLang,
		TargetLang: p.targetLang,
		Style:      p.style,
		Tone:       p.tone,
		Language:   p.language,
	}
}

// graphInput textProcessGraph 模式的输入。
func (p *textProcessParams) graphInput(intent service.TextProcessIntent) *service.TextProcessInput {
	return &service.TextProcessInput{
		Intent:     intent,
		Text:       p.text,
		TargetLang: p.targetLang,
		SourceLang: p.sourceLang,
		Style:      p.style,
		Tone:       p.tone,
		Language:   p.language,
	}
}

func translateParams(text, targetLang, sourceLang string) *textProcessParams {
	if sourceLang == "" || sourceLang == "auto" {
		sourceLang = "自动检测"
	}
	return &textProcessParams{text: text, targetLang: targetLang, sourceLang: sourceLang}
}

func rewriteParams(text, style, tone string) *textProcessParams {
	if style == "" {
		style = "简洁"
	}
	if tone == "" {
		tone = "专业"
	}
	return &textProcessParams{text: text, style: style, tone: tone}
}

func polishParams(text, language string) *textProcessParams {
	if language == "" {
		language = "中文"
	}
	return &textProcessParams{text: text, language: language}
}

// textProcessMeta 一种文本处理意图的元数据（scene / 兜底过滤标签 / 错误前缀 / 响应 data 形态）。
type textProcessMeta struct {
	scene     service.PromptScene
	filterKey string
	errLabel  string
	buildData func(p *textProcessParams, result string) gin.H
}

var textProcessMetas = map[service.TextProcessIntent]textProcessMeta{
	service.TextProcessTranslate: {
		scene:     service.SceneTranslate,
		filterKey: "ai_translate",
		errLabel:  "翻译",
		buildData: func(p *textProcessParams, result string) gin.H {
			return gin.H{"translated_text": result, "source_lang": p.sourceLang, "target_lang": p.targetLang}
		},
	},
	service.TextProcessRewrite: {
		scene:     service.SceneRewrite,
		filterKey: "ai_rewrite",
		errLabel:  "改写",
		buildData: func(p *textProcessParams, result string) gin.H {
			return gin.H{"rewritten_text": result}
		},
	},
	service.TextProcessPolish: {
		scene:     service.ScenePolish,
		filterKey: "ai_polish",
		errLabel:  "润色",
		buildData: func(p *textProcessParams, result string) gin.H {
			return gin.H{"polished_text": result}
		},
	},
}

// runTextProcess 执行一次文本处理（翻译/改写/润色）：textProcessGraph 优先、直调兜底。
// 三种意图共用同一骨架（原 6 处重复实现收敛于此），差异由 textProcessMetas 元数据表决定。
func (h *AIHandler) runTextProcess(c *gin.Context, p *textProcessParams, intent service.TextProcessIntent) {
	meta, ok := textProcessMetas[intent]
	if !ok {
		response.InternalServerError(c, "不支持的文本处理意图")
		return
	}

	// 翻译必须有目标语言
	if intent == service.TextProcessTranslate && p.targetLang == "" {
		response.BadRequest(c, "翻译需要 target_lang")
		return
	}

	if h.textProcessGraph != nil {
		result, err := h.textProcessGraph.Execute(c.Request.Context(), p.graphInput(intent))
		if err != nil {
			respondTextProcessError(c, meta.errLabel, err)
			return
		}
		respondTextProcessOK(c, meta.buildData(p, result.Result))
		return
	}

	systemPrompt := service.NewPromptManager().BuildSystemPrompt(meta.scene, p.promptContext())
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: p.text},
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages)
	if err != nil {
		respondTextProcessError(c, meta.errLabel, err)
		return
	}
	result = h.aiService.FilterOutput(result, meta.filterKey)

	respondTextProcessOK(c, meta.buildData(p, result))
}

func respondTextProcessError(c *gin.Context, label string, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": label + "失败: " + err.Error()})
}

func respondTextProcessOK(c *gin.Context, data gin.H) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}
