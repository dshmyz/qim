package handler

import (
	"encoding/json"
	"net/http"
	"qim-server/model"
	"qim-server/service"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AvatarHandler struct {
	db            *gorm.DB
	avatarService *service.AvatarService
}

func NewAvatarHandler(db *gorm.DB, avatarService *service.AvatarService) *AvatarHandler {
	return &AvatarHandler{
		db:            db,
		avatarService: avatarService,
	}
}

func (h *AvatarHandler) RegisterRoutes(router *gin.RouterGroup) {
	avatar := router.Group("/avatar")
	{
		avatar.GET("/config", h.GetConfig)
		avatar.POST("/config", h.CreateConfig)
		avatar.PUT("/config", h.UpdateConfig)
		avatar.DELETE("/config", h.DeleteConfig)

		avatar.POST("/learn-persona", h.TriggerLearnPersona)
		avatar.GET("/learn-status", h.GetLearnStatus)
		avatar.GET("/learned-persona", h.GetLearnedPersona)

		avatar.GET("/sessions", h.GetSessions)
		avatar.PUT("/sessions/:convId", h.UpdateSession)
		avatar.POST("/sessions/:convId/takeover", h.TakeoverSession)

		avatar.POST("/preview", h.PreviewReply)
	}
}

func (h *AvatarHandler) GetConfig(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var config model.AvatarConfig
	err := h.db.Where("user_id = ?", userID).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}

	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

type AvatarConfigResponse struct {
	ID                 uint                       `json:"id"`
	UserID             uint                       `json:"user_id"`
	Name               string                     `json:"name"`
	Enabled            bool                       `json:"enabled"`
	AutoLearnedPersona string                     `json:"auto_learned_persona"`
	CustomPersonaAddon string                     `json:"custom_persona_addon"`
	PersonaVersion     int                        `json:"persona_version"`
	LastLearnedAt      *time.Time                 `json:"last_learned_at"`
	KnowledgeScope     model.AvatarKnowledgeScope `json:"knowledge_scope"`
	TriggerRules       model.AvatarTriggerRules   `json:"trigger_rules"`
	ReplyStrategy      model.AvatarReplyStrategy  `json:"reply_strategy"`
	ModelConfigID      *uint                      `json:"model_config_id"`
	UseSystemConfig    bool                       `json:"use_system_config"`
	TakeoverCooldown   int                        `json:"takeover_cooldown"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
}

func (h *AvatarHandler) toConfigResponse(config model.AvatarConfig) AvatarConfigResponse {
	var knowledgeScope model.AvatarKnowledgeScope
	var triggerRules model.AvatarTriggerRules
	var replyStrategy model.AvatarReplyStrategy

	if config.KnowledgeScopeJSON != "" {
		json.Unmarshal([]byte(config.KnowledgeScopeJSON), &knowledgeScope)
	}
	if config.TriggerRulesJSON != "" {
		json.Unmarshal([]byte(config.TriggerRulesJSON), &triggerRules)
	}
	if config.ReplyStrategyJSON != "" {
		json.Unmarshal([]byte(config.ReplyStrategyJSON), &replyStrategy)
	}

	return AvatarConfigResponse{
		ID:                 config.ID,
		UserID:             config.UserID,
		Name:               config.Name,
		Enabled:            config.Enabled,
		AutoLearnedPersona: config.AutoLearnedPersona,
		CustomPersonaAddon: config.CustomPersonaAddon,
		PersonaVersion:     config.PersonaVersion,
		LastLearnedAt:      config.LastLearnedAt,
		KnowledgeScope:     knowledgeScope,
		TriggerRules:       triggerRules,
		ReplyStrategy:      replyStrategy,
		ModelConfigID:      config.ModelConfigID,
		UseSystemConfig:    config.UseSystemConfig,
		TakeoverCooldown:   config.TakeoverCooldown,
		CreatedAt:          config.CreatedAt,
		UpdatedAt:          config.UpdatedAt,
	}
}

type CreateAvatarConfigRequest struct {
	Name               string                     `json:"name"`
	UseSystemConfig    bool                       `json:"use_system_config"`
	ModelConfigID      *uint                      `json:"model_config_id"`
	TriggerRules       model.AvatarTriggerRules   `json:"trigger_rules"`
	KnowledgeScope     model.AvatarKnowledgeScope `json:"knowledge_scope"`
	ReplyStrategy      model.AvatarReplyStrategy  `json:"reply_strategy"`
	TakeoverCooldown   int                        `json:"takeover_cooldown"`
	CustomPersonaAddon string                     `json:"custom_persona_addon"`
}

func (h *AvatarHandler) CreateConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var existingConfig model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&existingConfig).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "已存在分身配置"})
		return
	}

	var req CreateAvatarConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	knowledgeScopeJSON, _ := json.Marshal(req.KnowledgeScope)
	triggerRulesJSON, _ := json.Marshal(req.TriggerRules)
	replyStrategyJSON, _ := json.Marshal(req.ReplyStrategy)

	config := model.AvatarConfig{
		UserID:             userID,
		Name:               req.Name,
		Enabled:            false,
		UseSystemConfig:    req.UseSystemConfig,
		ModelConfigID:      req.ModelConfigID,
		KnowledgeScopeJSON: string(knowledgeScopeJSON),
		TriggerRulesJSON:   string(triggerRulesJSON),
		ReplyStrategyJSON:  string(replyStrategyJSON),
		TakeoverCooldown:   req.TakeoverCooldown,
		CustomPersonaAddon: req.CustomPersonaAddon,
	}

	if err := h.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

type UpdateAvatarConfigRequest struct {
	Name               *string                     `json:"name"`
	Enabled            *bool                       `json:"enabled"`
	UseSystemConfig    *bool                       `json:"use_system_config"`
	ModelConfigID      *uint                       `json:"model_config_id"`
	TriggerRules       *model.AvatarTriggerRules   `json:"trigger_rules"`
	KnowledgeScope     *model.AvatarKnowledgeScope `json:"knowledge_scope"`
	ReplyStrategy      *model.AvatarReplyStrategy  `json:"reply_strategy"`
	TakeoverCooldown   *int                        `json:"takeover_cooldown"`
	CustomPersonaAddon *string                     `json:"custom_persona_addon"`
}

func (h *AvatarHandler) UpdateConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	var req UpdateAvatarConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.UseSystemConfig != nil {
		updates["use_system_config"] = *req.UseSystemConfig
	}
	if req.ModelConfigID != nil {
		updates["model_config_id"] = *req.ModelConfigID
	}
	if req.TriggerRules != nil {
		jsonData, _ := json.Marshal(req.TriggerRules)
		updates["trigger_rules_json"] = string(jsonData)
	}
	if req.KnowledgeScope != nil {
		jsonData, _ := json.Marshal(req.KnowledgeScope)
		updates["knowledge_scope_json"] = string(jsonData)
	}
	if req.ReplyStrategy != nil {
		jsonData, _ := json.Marshal(req.ReplyStrategy)
		updates["reply_strategy_json"] = string(jsonData)
	}
	if req.TakeoverCooldown != nil {
		updates["takeover_cooldown"] = *req.TakeoverCooldown
	}
	if req.CustomPersonaAddon != nil {
		updates["custom_persona_addon"] = *req.CustomPersonaAddon
	}

	if len(updates) > 0 {
		if err := h.db.Model(&config).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
			return
		}
	}

	h.db.Where("user_id = ?", userID).First(&config)
	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

func (h *AvatarHandler) DeleteConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	result := h.db.Where("user_id = ?", userID).Delete(&model.AvatarConfig{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	h.db.Where("user_id = ?", userID).Delete(&model.AvatarSession{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *AvatarHandler) GetSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var sessions []model.AvatarSession
	if err := h.db.Where("user_id = ?", userID).Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sessions})
}

type UpdateSessionRequest struct {
	AvatarEnabled *bool `json:"avatar_enabled"`
}

func (h *AvatarHandler) UpdateSession(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	convID := c.Param("convId")

	var req UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var session model.AvatarSession
	err := h.db.Where("user_id = ? AND conversation_id = ?", userID, convID).First(&session).Error

	if err == gorm.ErrRecordNotFound {
		if req.AvatarEnabled != nil && *req.AvatarEnabled {
			session = model.AvatarSession{
				UserID:         userID,
				ConversationID: parseUint(convID),
				AvatarEnabled:  true,
			}
			if err := h.db.Create(&session).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
				return
			}
		} else {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	} else {
		if req.AvatarEnabled != nil {
			h.db.Model(&session).Update("avatar_enabled", *req.AvatarEnabled)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

func (h *AvatarHandler) TakeoverSession(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	convID := c.Param("convId")

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "分身配置不存在"})
		return
	}

	takeoverUntil := time.Now().Add(time.Duration(config.TakeoverCooldown) * time.Minute)

	var session model.AvatarSession
	err := h.db.Where("user_id = ? AND conversation_id = ?", userID, convID).First(&session).Error

	if err == gorm.ErrRecordNotFound {
		session = model.AvatarSession{
			UserID:         userID,
			ConversationID: parseUint(convID),
			AvatarEnabled:  false,
			TakeoverUntil:  &takeoverUntil,
		}
		h.db.Create(&session)
	} else if err == nil {
		h.db.Model(&session).Update("takeover_until", takeoverUntil)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

func parseUint(s string) uint {
	var result uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + uint(c-'0')
		}
	}
	return result
}

func (h *AvatarHandler) TriggerLearnPersona(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "请先创建分身配置"})
		return
	}

	var existingTask model.AvatarLearnTask
	err := h.db.Where("user_id = ? AND status IN ?", userID, []string{"pending", "processing"}).First(&existingTask).Error
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "已有学习任务在进行中"})
		return
	}

	task := model.AvatarLearnTask{
		UserID: userID,
		Status: "pending",
	}
	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建任务失败"})
		return
	}

	go h.avatarService.LearnPersona(userID, task.ID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"task_id": task.ID}})
}

func (h *AvatarHandler) GetLearnStatus(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var task model.AvatarLearnTask
	err := h.db.Where("user_id = ?", userID).Order("created_at DESC").First(&task).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"status":        "idle",
			"progress":      0,
			"message_count": 0,
			"error":         nil,
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"status":        task.Status,
		"progress":      task.Progress,
		"message_count": task.MessageCount,
		"error":         task.Error,
	}})
}

func (h *AvatarHandler) GetLearnedPersona(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config.AutoLearnedPersona})
}

func (h *AvatarHandler) PreviewReply(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	reply, err := h.avatarService.PreviewReply(userID, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"reply": reply}})
}
