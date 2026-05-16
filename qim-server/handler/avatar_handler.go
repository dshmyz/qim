package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"qim-server/ai"
	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/pkg/validation"
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
	Icon        string
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

		// 记忆管理
		avatar.GET("/memories", h.GetMemories)
		avatar.DELETE("/memory/:id", h.DeleteMemory)
		avatar.POST("/memory/search", h.SearchMemories)

		// 笔记搜索
		avatar.POST("/note-search", h.SearchNotes)

		// 触发检查
		avatar.POST("/trigger-check", h.CheckTrigger)
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

	if err := validation.ValidateAliasName(req.Name); err != nil {
		response.BadRequest(c, err.Error())
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

	if err := validation.ValidateAliasName(req.Name); err != nil {
		response.BadRequest(c, err.Error())
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

	// 创建学习任务
	task := model.AvatarLearnTask{
		UserID: userID,
		Status: "pending",
	}
	if err := h.db.Create(&task).Error; err != nil {
		response.InternalServerError(c, "创建任务失败")
		return
	}

	// 使用异步方式学习
	utils.SafeGoWithLabel("avatar-learn", func() {
		h.avatarService.LearnPersona(userID, task.ID)
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"taskId": task.ID}})
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

	// 查询最新的学习任务
	var task model.AvatarLearnTask
	err := h.db.Where("user_id = ?", userID).Order("created_at DESC").First(&task).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"status":        "idle",
			"progress":      0,
			"messageCount":  0,
			"error":         nil,
			"lastLearnedAt": config.LastLearnedAt,
		}})
		return
	}

	// 映射状态值：后端状态 -> 前端期望状态
	frontendStatus := task.Status
	if task.Status == "pending" || task.Status == "processing" {
		frontendStatus = "learning"
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"status":        frontendStatus,
		"progress":      task.Progress,
		"messageCount":  task.MessageCount,
		"error":         task.Error,
		"lastLearnedAt": config.LastLearnedAt,
	}})
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
		{ID: "chat", Name: "智能对话", Description: "基础对话能力", Icon: "fas fa-comments"},
		{ID: "knowledge_search", Name: "知识检索", Description: "搜索群知识库和文档", Icon: "fas fa-search"},
		{ID: "knowledge_save", Name: "知识存储", Description: "将内容存入知识库", Icon: "fas fa-save"},
		{ID: "memory_search", Name: "记忆检索", Description: "搜索历史记忆", Icon: "fas fa-brain"},
		{ID: "summary", Name: "摘要总结", Description: "生成消息摘要", Icon: "fas fa-list"},
		{ID: "translate", Name: "翻译", Description: "多语言翻译", Icon: "fas fa-language"},
		{ID: "server_monitor", Name: "服务器监控", Description: "查看系统资源状态", Icon: "fas fa-chart-line"},
		{ID: "log_analyzer", Name: "日志分析", Description: "分析系统日志", Icon: "fas fa-file-alt"},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": tools})
}

// GetAvatarTools 获取分身绑定的工具
func (h *AvatarHandler) GetAvatarTools(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "分身配置不存在")
		return
	}

	var bindings []model.AvatarToolBinding
	if err := h.db.Where("avatar_id = ?", config.ID).Order("priority DESC").Find(&bindings).Error; err != nil {
		response.InternalServerError(c, "获取工具失败")
		return
	}

	toolMap := map[string]model.AvatarToolBinding{}
	for _, b := range bindings {
		toolMap[b.ToolID] = b
	}

	type ToolWithInfo struct {
		model.AvatarToolBinding
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}

	allTools := []AvatarTool{
		{ID: "chat", Name: "智能对话", Description: "基础对话能力", Icon: "fas fa-comments"},
		{ID: "knowledge_search", Name: "知识检索", Description: "搜索群知识库和文档", Icon: "fas fa-search"},
		{ID: "knowledge_save", Name: "知识存储", Description: "将内容存入知识库", Icon: "fas fa-save"},
		{ID: "memory_search", Name: "记忆检索", Description: "搜索历史记忆", Icon: "fas fa-brain"},
		{ID: "summary", Name: "摘要总结", Description: "生成消息摘要", Icon: "fas fa-list"},
		{ID: "translate", Name: "翻译", Description: "多语言翻译", Icon: "fas fa-language"},
		{ID: "server_monitor", Name: "服务器监控", Description: "查看系统资源状态", Icon: "fas fa-chart-line"},
		{ID: "log_analyzer", Name: "日志分析", Description: "分析系统日志", Icon: "fas fa-file-alt"},
	}

	result := make([]ToolWithInfo, 0, len(allTools))
	for _, t := range allTools {
		info := ToolWithInfo{
			Name:        t.Name,
			Description: t.Description,
			Icon:        t.Icon,
		}
		if b, ok := toolMap[t.ID]; ok {
			info.AvatarToolBinding = b
		} else {
			info.AvatarToolBinding = model.AvatarToolBinding{
				AvatarID: config.ID,
				ToolID:   t.ID,
				Enabled:  false,
				Priority: 1,
			}
		}
		result = append(result, info)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// BindTool 绑定工具到分身
func (h *AvatarHandler) BindTool(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	toolID := c.Param("toolId")

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "分身配置不存在")
		return
	}

	var binding model.AvatarToolBinding
	if err := h.db.Where("avatar_id = ? AND tool_id = ?", config.ID, toolID).First(&binding).Error; err == nil {
		binding.Enabled = true
		h.db.Save(&binding)
	} else {
		binding = model.AvatarToolBinding{
			AvatarID: config.ID,
			ToolID:   toolID,
			Enabled:  true,
			Priority: 1,
		}
		h.db.Create(&binding)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": binding})
}

// UpdateAvatarTools 批量更新分身工具配置
func (h *AvatarHandler) UpdateAvatarTools(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "分身配置不存在")
		return
	}

	var tools []struct {
		ToolID   string `json:"tool_id"`
		Enabled  bool   `json:"enabled"`
		Priority int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&tools); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	for _, t := range tools {
		var binding model.AvatarToolBinding
		if err := h.db.Where("avatar_id = ? AND tool_id = ?", config.ID, t.ToolID).First(&binding).Error; err == nil {
			binding.Enabled = t.Enabled
			binding.Priority = t.Priority
			h.db.Save(&binding)
		} else if t.Enabled {
			h.db.Create(&model.AvatarToolBinding{
				AvatarID: config.ID,
				ToolID:   t.ToolID,
				Enabled:  true,
				Priority: t.Priority,
			})
		}
	}

	response.SuccessWithMessage(c, "工具配置已更新", nil)
}

// UnbindTool 解绑工具
func (h *AvatarHandler) UnbindTool(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	toolID := c.Param("toolId")

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		response.NotFound(c, "分身配置不存在")
		return
	}

	if err := h.db.Where("avatar_id = ? AND tool_id = ?", config.ID, toolID).Delete(&model.AvatarToolBinding{}).Error; err != nil {
		response.InternalServerError(c, "解绑失败")
		return
	}

	response.SuccessWithMessage(c, "解绑成功", nil)
}

func (h *AvatarHandler) GetMemories(c *gin.Context) {
	userID, _ := c.Get("user_id")
	memorySvc := di.GlobalContainer.AvatarMemoryService
	if memorySvc == nil {
		response.Success(c, []interface{}{})
		return
	}

	memories, err := memorySvc.GetUserMemories(userID.(uint), 50)
	if err != nil {
		response.InternalServerError(c, "获取记忆失败")
		return
	}

	response.Success(c, memories)
}

func (h *AvatarHandler) DeleteMemory(c *gin.Context) {
	memoryID := c.Param("id")
	memorySvc := di.GlobalContainer.AvatarMemoryService
	if memorySvc == nil {
		response.NotFound(c, "记忆不存在")
		return
	}

	if err := memorySvc.DeleteMemory(memoryID); err != nil {
		response.InternalServerError(c, "删除记忆失败")
		return
	}

	response.SuccessWithMessage(c, "记忆已删除", nil)
}

func (h *AvatarHandler) SearchMemories(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	memorySvc := di.GlobalContainer.AvatarMemoryService
	if memorySvc == nil {
		response.Success(c, []interface{}{})
		return
	}

	results, err := memorySvc.Recall(userID.(uint), req.Query, req.TopK)
	if err != nil {
		response.InternalServerError(c, "搜索记忆失败")
		return
	}

	response.Success(c, results)
}

func (h *AvatarHandler) SearchNotes(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	noteSvc := di.GlobalContainer.NoteVectorService
	if noteSvc == nil {
		response.Success(c, []interface{}{})
		return
	}

	results, err := noteSvc.SearchNotes(userID.(uint), req.Query, req.TopK)
	if err != nil {
		response.InternalServerError(c, "搜索笔记失败")
		return
	}

	response.Success(c, results)
}

func (h *AvatarHandler) CheckTrigger(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req struct {
		ConversationID uint   `json:"conversation_id" binding:"required"`
		Message        string `json:"message" binding:"required"`
		SenderName     string `json:"sender_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	triggerSvc := di.GlobalContainer.AvatarTriggerService
	if triggerSvc == nil {
		response.Success(c, gin.H{"should_reply": false, "reason": "触发服务未初始化"})
		return
	}

	shouldReply, reason, err := triggerSvc.ShouldReply(userID.(uint), req.ConversationID, req.Message, req.SenderName)
	if err != nil {
		response.InternalServerError(c, "触发检查失败")
		return
	}

	response.Success(c, gin.H{
		"should_reply": shouldReply,
		"reason":       reason,
	})
}
