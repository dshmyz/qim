package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"qim-server/ai"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AvatarTool 工具信息
type AvatarTool struct {
	ID          string
	Name        string
	Description string
}

type AvatarHandler struct {
	db            *gorm.DB
	avatarService *service.AvatarService
	mcpServer     *ai.MCPServer
}

func NewAvatarHandler(db *gorm.DB, avatarService *service.AvatarService, mcpServer *ai.MCPServer) *AvatarHandler {
	return &AvatarHandler{
		db:            db,
		avatarService: avatarService,
		mcpServer:     mcpServer,
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

		// 审批相关
		avatar.POST("/apply", h.ApplyForApproval)
		avatar.POST("/cancel-apply", h.CancelApplication)

		// 工具绑定相关
		avatar.GET("/tools", h.GetAvailableTools)
		avatar.GET("/:id/tools", h.GetAvatarTools)
		avatar.POST("/:id/tools/:toolId", h.BindTool)
		avatar.DELETE("/:id/tools/:toolId", h.UnbindTool)
	}
}

type AvatarConfigResponse struct {
	ID                 uint                       `json:"id"`
	UserID             uint                       `json:"userId"`
	Name               string                     `json:"name"`
	Enabled            bool                       `json:"enabled"`
	AutoLearnedPersona string                     `json:"autoLearnedPersona"`
	CustomPersonaAddon string                     `json:"customPersonaAddon"`
	PersonaVersion     int                        `json:"personaVersion"`
	LastLearnedAt      *time.Time                 `json:"lastLearnedAt"`
	KnowledgeScope     model.AvatarKnowledgeScope `json:"knowledgeScope"`
	TriggerRules       model.AvatarTriggerRules   `json:"triggerRules"`
	ReplyStrategy      model.AvatarReplyStrategy  `json:"replyStrategy"`
	ModelConfigID      *uint                      `json:"modelConfigId"`
	UseSystemConfig    bool                       `json:"useSystemConfig"`
	TakeoverCooldown   int                        `json:"takeoverCooldown"`
	// 审批相关
	ApprovalStatus         string     `json:"approvalStatus"`
	ApprovalRejectedReason string     `json:"approvalRejectedReason"`
	ApprovalAppliedAt      *time.Time `json:"approvalAppliedAt"`
	ApprovalReviewedAt     *time.Time `json:"approvalReviewedAt"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
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
		ID:                     config.ID,
		UserID:                 config.UserID,
		Name:                   config.Name,
		Enabled:                config.Enabled,
		AutoLearnedPersona:     config.AutoLearnedPersona,
		CustomPersonaAddon:     config.CustomPersonaAddon,
		PersonaVersion:         config.PersonaVersion,
		LastLearnedAt:          config.LastLearnedAt,
		KnowledgeScope:         knowledgeScope,
		TriggerRules:           triggerRules,
		ReplyStrategy:          replyStrategy,
		ModelConfigID:          config.ModelConfigID,
		UseSystemConfig:        config.UseSystemConfig,
		TakeoverCooldown:       config.TakeoverCooldown,
		ApprovalStatus:         config.ApprovalStatus,
		ApprovalRejectedReason: config.RejectReason,
		ApprovalAppliedAt:      config.AppliedAt,
		ApprovalReviewedAt:     config.ApprovedAt,
		CreatedAt:              config.CreatedAt,
		UpdatedAt:              config.UpdatedAt,
	}
}

type CreateAvatarConfigRequest struct {
	Name               string                     `json:"name"`
	UseSystemConfig    bool                       `json:"useSystemConfig"`
	ModelConfigID      *uint                      `json:"modelConfigId"`
	TriggerRules       model.AvatarTriggerRules   `json:"triggerRules"`
	KnowledgeScope     model.AvatarKnowledgeScope `json:"knowledgeScope"`
	ReplyStrategy      model.AvatarReplyStrategy  `json:"replyStrategy"`
	TakeoverCooldown   int                        `json:"takeoverCooldown"`
	CustomPersonaAddon string                     `json:"customPersonaAddon"`
}

func (h *AvatarHandler) CreateConfig(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未认证")
		return
	}

	userID, ok := userIDAny.(uint)
	if !ok {
		response.BadRequest(c, "用户ID类型错误")
		return
	}

	var existingConfig model.AvatarConfig
	// 检查是否存在（包括软删除的记录）
	if err := h.db.Unscoped().Where("user_id = ?", userID).First(&existingConfig).Error; err == nil {
		// 如果是软删除记录，恢复它并重置审批状态
		if existingConfig.DeletedAt.Valid {
			if err := h.db.Unscoped().Model(&existingConfig).Updates(map[string]interface{}{
				"deleted_at":      nil,
				"approval_status": model.ApprovalStatusNone,
				"reject_reason":   "",
				"applied_at":      nil,
				"approved_at":     nil,
			}).Error; err != nil {
				log.Printf("恢复软删除记录失败: %v", err)
				response.InternalServerError(c, "恢复分身配置失败")
				return
			}
			response := h.toConfigResponse(existingConfig)
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
			return
		}
		response.BadRequest(c, "已存在分身配置")
		return
	}

	var req CreateAvatarConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Create avatar config bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	knowledgeScopeJSON, err := json.Marshal(req.KnowledgeScope)
	if err != nil {
		log.Printf("Create avatar config marshal knowledgeScope error: %v", err)
		response.BadRequest(c, "知识范围序列化失败")
		return
	}

	triggerRulesJSON, err := json.Marshal(req.TriggerRules)
	if err != nil {
		log.Printf("Create avatar config marshal triggerRules error: %v", err)
		response.BadRequest(c, "触发规则序列化失败")
		return
	}

	replyStrategyJSON, err := json.Marshal(req.ReplyStrategy)
	if err != nil {
		log.Printf("Create avatar config marshal replyStrategy error: %v", err)
		response.BadRequest(c, "回复策略序列化失败")
		return
	}

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
		log.Printf("Create avatar config failed: %v, userID: %d, config: %+v", err, userID, config)
		response.InternalServerError(c, "创建失败")
		return
	}

	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

type UpdateAvatarConfigRequest struct {
	Name               string                     `json:"name"`
	Enabled            bool                       `json:"enabled"`
	UseSystemConfig    bool                       `json:"useSystemConfig"`
	ModelConfigID      *uint                      `json:"modelConfigId"`
	TriggerRules       model.AvatarTriggerRules   `json:"triggerRules"`
	KnowledgeScope     model.AvatarKnowledgeScope `json:"knowledgeScope"`
	ReplyStrategy      model.AvatarReplyStrategy  `json:"replyStrategy"`
	TakeoverCooldown   int                        `json:"takeoverCooldown"`
	CustomPersonaAddon string                     `json:"customPersonaAddon"`
}

func (h *AvatarHandler) UpdateConfig(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未认证")
		return
	}

	userID, ok := userIDAny.(uint)
	if !ok {
		response.BadRequest(c, "用户ID类型错误")
		return
	}

	var config model.AvatarConfig
	if err := h.db.Unscoped().Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}
	// 如果是软删除的记录，恢复它
	if config.DeletedAt.Valid {
		if err := h.db.Unscoped().Model(&config).Update("deleted_at", nil).Error; err != nil {
			response.InternalServerError(c, "恢复配置失败")
			return
		}
	}

	var req UpdateAvatarConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	triggerRulesJSON, _ := json.Marshal(req.TriggerRules)
	knowledgeScopeJSON, _ := json.Marshal(req.KnowledgeScope)
	replyStrategyJSON, _ := json.Marshal(req.ReplyStrategy)

	updates := map[string]interface{}{
		"name":                 req.Name,
		"enabled":              req.Enabled,
		"use_system_config":    req.UseSystemConfig,
		"model_config_id":      req.ModelConfigID,
		"takeover_cooldown":    req.TakeoverCooldown,
		"custom_persona_addon": req.CustomPersonaAddon,
		"trigger_rules_json":   string(triggerRulesJSON),
		"knowledge_scope_json": string(knowledgeScopeJSON),
		"reply_strategy_json":  string(replyStrategyJSON),
	}

	if err := h.db.Model(&config).Updates(updates).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	h.db.First(&config, config.ID)
	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

func (h *AvatarHandler) GetConfig(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未认证")
		return
	}

	userID, ok := userIDAny.(uint)
	if !ok {
		response.BadRequest(c, "用户ID类型错误")
		return
	}

	var config model.AvatarConfig
	if err := h.db.Unscoped().Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	// 如果是软删除的记录，视为不存在
	if config.DeletedAt.Valid {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}

	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

func (h *AvatarHandler) DeleteConfig(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未认证")
		return
	}

	userID, ok := userIDAny.(uint)
	if !ok {
		response.BadRequest(c, "用户ID类型错误")
		return
	}

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	// 软删除
	if err := h.db.Delete(&config).Error; err != nil {
		response.InternalServerError(c, "删除失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// TriggerLearnPersona 触发人设学习
func (h *AvatarHandler) TriggerLearnPersona(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	// 使用异步方式学习
	go func() {
		h.avatarService.LearnFromMultipleSources(userID)
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"taskId": "async_task"}})
}

// GetLearnStatus 获取学习状态
func (h *AvatarHandler) GetLearnStatus(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	status := gin.H{
		"status":        "idle",
		"progress":      0,
		"messageCount":  0,
		"error":         nil,
		"lastLearnedAt": config.LastLearnedAt,
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": status})
}

// GetLearnedPersona 获取学习结果
func (h *AvatarHandler) GetLearnedPersona(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config.AutoLearnedPersona})
}

// GetSessions 获取会话分身状态
func (h *AvatarHandler) GetSessions(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var sessions []model.AvatarSession
	if err := h.db.Where("user_id = ?", userID).Find(&sessions).Error; err != nil {
		response.InternalServerError(c, "获取会话失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sessions})
}

// UpdateSession 更新会话分身状态
func (h *AvatarHandler) UpdateSession(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	convIdStr := c.Param("convId")

	var req struct {
		AvatarEnabled bool `json:"avatarEnabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	convId, err := strconv.ParseUint(convIdStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "会话ID格式错误")
		return
	}

	var session model.AvatarSession
	if err := h.db.Where("user_id = ? AND conversation_id = ?", userID, convId).First(&session).Error; err != nil {
		// 不存在则创建
		session = model.AvatarSession{
			UserID:         userID,
			ConversationID: uint(convId),
			AvatarEnabled:  req.AvatarEnabled,
		}
	} else {
		session.AvatarEnabled = req.AvatarEnabled
	}

	if err := h.db.Save(&session).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

// TakeoverSession 接管分身
func (h *AvatarHandler) TakeoverSession(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	convIdStr := c.Param("convId")

	convId, err := strconv.ParseUint(convIdStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "会话ID格式错误")
		return
	}

	var session model.AvatarSession
	if err := h.db.Where("user_id = ? AND conversation_id = ?", userID, convId).First(&session).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	now := time.Now()
	tenMinutesLater := now.Add(10 * time.Minute)
	session.TakeoverUntil = &tenMinutesLater

	if err := h.db.Save(&session).Error; err != nil {
		response.InternalServerError(c, "接管失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

// PreviewReply 预览回复
func (h *AvatarHandler) PreviewReply(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "消息不能为空")
		return
	}

	reply, err := h.avatarService.PreviewReply(userID, req.Message)
	if err != nil {
		response.InternalServerError(c, "生成预览失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"reply": reply}})
}

// ApplyForApproval 申请审批
func (h *AvatarHandler) ApplyForApproval(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	if !config.CanApply() {
		response.BadRequest(c, "当前状态不允许申请")
		return
	}

	now := time.Now()
	config.ApprovalStatus = model.ApprovalStatusPending
	config.AppliedAt = &now

	if err := h.db.Save(&config).Error; err != nil {
		response.InternalServerError(c, "申请失败")
		return
	}

	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

// CancelApplication 取消申请
func (h *AvatarHandler) CancelApplication(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	if !config.CanCancel() {
		response.BadRequest(c, "当前状态不允许取消")
		return
	}

	config.ApprovalStatus = model.ApprovalStatusNone
	config.AppliedAt = nil

	if err := h.db.Save(&config).Error; err != nil {
		response.InternalServerError(c, "取消失败")
		return
	}

	response := h.toConfigResponse(config)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": response})
}

// GetAvailableTools 获取可用工具列表
func (h *AvatarHandler) GetAvailableTools(c *gin.Context) {
	tools := []AvatarTool{
		{ID: "chat", Name: "智能对话", Description: "基础对话能力"},
		{ID: "search", Name: "知识检索", Description: "搜索历史消息和文档"},
		{ID: "summary", Name: "摘要总结", Description: "生成消息摘要"},
		{ID: "translate", Name: "翻译", Description: "多语言翻译"},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": tools})
}

// GetAvatarTools 获取分身绑定的工具
func (h *AvatarHandler) GetAvatarTools(c *gin.Context) {
	avatarID := c.Param("id")

	var bindings []model.AvatarToolBinding
	if err := h.db.Where("avatar_id = ?", avatarID).Find(&bindings).Error; err != nil {
		response.InternalServerError(c, "获取工具失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": bindings})
}

// BindTool 绑定工具到分身
func (h *AvatarHandler) BindTool(c *gin.Context) {
	toolID := c.Param("toolId")

	binding := model.AvatarToolBinding{
		AvatarID: 0,
		ToolID:   toolID,
		Enabled:  true,
		Priority: 1,
	}

	if err := h.db.Create(&binding).Error; err != nil {
		response.InternalServerError(c, "绑定失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": binding})
}

// UnbindTool 解绑工具
func (h *AvatarHandler) UnbindTool(c *gin.Context) {
	avatarID := c.Param("id")
	toolID := c.Param("toolId")

	if err := h.db.Where("avatar_id = ? AND tool_id = ?", avatarID, toolID).Delete(&model.AvatarToolBinding{}).Error; err != nil {
		response.InternalServerError(c, "解绑失败")
		return
	}

	response.SuccessWithMessage(c, "解绑成功", nil)
}
