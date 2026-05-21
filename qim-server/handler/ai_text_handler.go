package handler

import (
	"net/http"
	"qim-server/ai"
	"qim-server/pkg/response"
	"qim-server/service"

	"github.com/gin-gonic/gin"
)

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

	systemPrompt := "你是一个图片翻译助手。请完成以下两步：\n1. 识别图片中的文字内容\n2. 将识别出的内容翻译成" + langName + "\n\n请以以下格式输出：\n【原文】\n识别出的文字\n\n【译文】\n翻译结果"

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "请识别这张图片中的文字并翻译成" + langName, ImageURL: req.ImageURL},
	}

	result, err := h.aiService.GetCompletion(ai.TaskTypeAnalysis, messages_input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片翻译失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"translated_text": result,
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
