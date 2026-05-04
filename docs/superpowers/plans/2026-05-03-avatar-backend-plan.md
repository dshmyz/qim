# 用户分身（Avatar）后端实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现分身功能的后端部分，包括数据模型、API Handler、人设学习服务、分身回复服务（含 Worker Pool 和限流）、消息处理集成。

**架构：** 独立分身系统，分身在私聊中直接回复，在群聊中触发后以私聊方式回复触发者。后端使用 Worker Pool 控制并发 LLM 调用，全局速率限制防止 API 过载。

**技术栈：** Go + Gin + GORM + AI Service（复用现有）

---

## 文件结构

### 新建文件

| 文件 | 职责 |
|------|------|
| `model/avatar.go` | 分身数据模型（AvatarConfig、AvatarSession） |
| `handler/avatar_handler.go` | 分身 API Handler（CRUD + 会话控制） |
| `service/avatar_service.go` | 分身核心服务（回复生成、人设学习） |
| `service/avatar_worker_pool.go` | Worker Pool 和限流器 |

### 修改文件

| 文件 | 变更 |
|------|------|
| `app/routes.go` | 注册分身路由 |
| `handler/smart_reply_handler.go` | 集成分身触发逻辑 |

---

## 任务 1：数据模型

**文件：**
- 创建：`model/avatar.go`

- [ ] **步骤 1：创建分身数据模型**

```go
package model

import (
	"time"
	"gorm.io/gorm"
)

type AvatarConfig struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"uniqueIndex;not null"`
	Name      string         `json:"name" gorm:"size:100;default:'我的分身'"`
	Enabled   bool           `json:"enabled" gorm:"default:false"`

	AutoLearnedPersona string `json:"auto_learned_persona" gorm:"type:text"`
	CustomPersonaAddon string `json:"custom_persona_addon" gorm:"type:text"`
	PersonaVersion     int    `json:"persona_version" gorm:"default:0"`
	LastLearnedAt      *time.Time `json:"last_learned_at"`

	KnowledgeScopeJSON string `json:"-" gorm:"type:text"`
	TriggerRulesJSON   string `json:"-" gorm:"type:text"`
	ReplyStrategyJSON  string `json:"-" gorm:"type:text"`

	ModelConfigID   *uint `json:"model_config_id"`
	UseSystemConfig bool  `json:"use_system_config" gorm:"default:true"`

	TakeoverCooldown int `json:"takeover_cooldown" gorm:"default:10"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	User       User          `json:"user,omitempty" gorm:"foreignkey:UserID"`
	ModelConfig *UserAIConfig `json:"model_config,omitempty" gorm:"foreignkey:ModelConfigID"`
}

type AvatarSession struct {
	ID             uint       `json:"id" gorm:"primarykey"`
	ConversationID uint       `json:"conversation_id" gorm:"not null;uniqueIndex:idx_user_conv"`
	UserID         uint       `json:"user_id" gorm:"not null;uniqueIndex:idx_user_conv"`
	AvatarEnabled  bool       `json:"avatar_enabled" gorm:"default:false"`
	TakeoverUntil  *time.Time `json:"takeover_until"`
	LastReplyAt    *time.Time `json:"last_reply_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Conversation Conversation `json:"conversation,omitempty" gorm:"foreignkey:ConversationID"`
	User         User         `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

type AvatarLearnTask struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	UserID       uint       `json:"user_id" gorm:"not null;index"`
	Status       string     `json:"status" gorm:"size:20;default:'pending'"`
	Progress     int        `json:"progress" gorm:"default:0"`
	MessageCount int        `json:"message_count"`
	Error        string     `json:"error" gorm:"type:text"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	User User `json:"user,omitempty" gorm:"foreignkey:UserID"`
}

type AvatarKnowledgeScope struct {
	ConversationHistory bool `json:"conversation_history"`
	KnowledgeDocs       bool `json:"knowledge_docs"`
	Notes               bool `json:"notes"`
	Tasks               bool `json:"tasks"`
}

type AvatarTriggerRules struct {
	Mode                string   `json:"mode"`
	Keywords            []string `json:"keywords"`
	TimeRanges          []AvatarTimeRange `json:"time_ranges"`
	ExcludedConversations []uint `json:"excluded_conversations"`
}

type AvatarTimeRange struct {
	DayOfWeek  []int `json:"day_of_week"`
	StartHour  int   `json:"start_hour"`
	EndHour    int   `json:"end_hour"`
}

type AvatarReplyStrategy struct {
	MaxReplyLength       string  `json:"max_reply_length"`
	ReplyDelay           int     `json:"reply_delay"`
	ConfidenceThreshold  float64 `json:"confidence_threshold"`
	DisclaimerStyle      string  `json:"disclaimer_style"`
}
```

- [ ] **步骤 2：添加数据库迁移**

在 `database/database.go` 的 AutoMigrate 中添加新模型：

```go
db.AutoMigrate(
	// ... 现有模型 ...
	&model.AvatarConfig{},
	&model.AvatarSession{},
	&model.AvatarLearnTask{},
)
```

- [ ] **步骤 3：Commit**

```bash
git add model/avatar.go database/database.go
git commit -m "feat(avatar): add avatar data models"
```

---

## 任务 2：Avatar Handler

**文件：**
- 创建：`handler/avatar_handler.go`

- [ ] **步骤 1：创建 Handler 结构和基础方法**

```go
package handler

import (
	"encoding/json"
	"net/http"
	"time"
	"qim-server/model"
	"qim-server/service"

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
```

- [ ] **步骤 2：添加 CreateConfig 和 UpdateConfig**

```go
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
```

- [ ] **步骤 3：添加会话管理方法**

```go
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
				UserID:        userID,
				ConversationID: parseUint(convID),
				AvatarEnabled: true,
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
			UserID:        userID,
			ConversationID: parseUint(convID),
			AvatarEnabled: false,
			TakeoverUntil: &takeoverUntil,
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
```

- [ ] **步骤 4：添加学习相关方法**

```go
func (h *AvatarHandler) TriggerLearnPersona(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var config model.AvatarConfig
	if err := h.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "请先创建分身配置"})
		return
	}

	var existingTask model.AvatarLearnTask
	err := h.db.Where("user_id = ? AND status IN ?", userID, []string{"pending", "learning"}).First(&existingTask).Error
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
			"status":       "idle",
			"progress":     0,
			"message_count": 0,
			"error":        nil,
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
```

- [ ] **步骤 5：Commit**

```bash
git add handler/avatar_handler.go
git commit -m "feat(avatar): add avatar API handler"
```

---

## 任务 3：Avatar Service

**文件：**
- 创建：`service/avatar_service.go`

- [ ] **步骤 1：创建服务结构**

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"qim-server/ai"
	"qim-server/model"
	"qim-server/utils"

	"gorm.io/gorm"
)

type AvatarService struct {
	db        *gorm.DB
	aiService *ai.AIService
	workerPool *AvatarWorkerPool
}

func NewAvatarService(db *gorm.DB, aiService *ai.AIService) *AvatarService {
	service := &AvatarService{
		db:        db,
		aiService: aiService,
	}
	service.workerPool = NewAvatarWorkerPool(5, 30, service)
	return service
}

func (s *AvatarService) GetWorkerPool() *AvatarWorkerPool {
	return s.workerPool
}
```

- [ ] **步骤 2：实现人设学习**

```go
func (s *AvatarService) LearnPersona(userID uint, taskID uint) {
	var task model.AvatarLearnTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return
	}

	s.db.Model(&task).Updates(map[string]interface{}{
		"status":     "learning",
		"started_at": time.Now(),
	})

	var messages []model.Message
	s.db.Table("messages m").
		Joins("JOIN conversation_members cm ON m.conversation_id = cm.conversation_id").
		Where("cm.user_id = ? AND m.sender_id = ?", userID, userID).
		Where("m.type = ?", "text").
		Where("m.created_at > ?", time.Now().AddDate(0, -3, 0)).
		Order("m.created_at DESC").
		Limit(500).
		Select("m.content").
		Find(&messages)

	s.db.Model(&task).Update("message_count", len(messages))

	if len(messages) < 10 {
		s.db.Model(&task).Updates(map[string]interface{}{
			"status":       "failed",
			"error":        "历史消息不足，无法学习风格",
			"completed_at": time.Now(),
		})
		return
	}

	var contents []string
	for _, msg := range messages {
		if len(msg.Content) > 20 && len(msg.Content) < 500 {
			contents = append(contents, msg.Content)
		}
	}

	sampleText := strings.Join(contents[:min(50, len(contents))], "\n\n")

	prompt := fmt.Sprintf(`分析以下用户发送的消息样本，总结这个用户的说话风格特点。

消息样本：
%s

请从以下维度分析：
1. 语气特点（正式/随意/幽默/严肃等）
2. 常用表达方式和口头禅
3. 回复长度偏好（简短/详细）
4. 表情符号使用习惯
5. 专业领域或兴趣话题
6. 其他显著的说话风格特征

请用简洁的中文描述，不超过200字。`, sampleText)

	persona, err := s.aiService.GetCompletion(context.Background(), prompt, "", nil)
	if err != nil {
		s.db.Model(&task).Updates(map[string]interface{}{
			"status":       "failed",
			"error":        err.Error(),
			"completed_at": time.Now(),
		})
		return
	}

	now := time.Now()
	s.db.Model(&model.AvatarConfig{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"auto_learned_persona": persona,
		"persona_version":      gorm.Expr("persona_version + 1"),
		"last_learned_at":      now,
	})

	s.db.Model(&task).Updates(map[string]interface{}{
		"status":       "completed",
		"progress":     100,
		"completed_at": now,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **步骤 3：实现回复生成**

```go
func (s *AvatarService) GenerateReply(userID uint, conversationID uint, triggerMessage string) (string, error) {
	var config model.AvatarConfig
	if err := s.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		return "", fmt.Errorf("分身配置不存在")
	}

	var knowledgeScope model.AvatarKnowledgeScope
	if config.KnowledgeScopeJSON != "" {
		json.Unmarshal([]byte(config.KnowledgeScopeJSON), &knowledgeScope)
	}

	var replyStrategy model.AvatarReplyStrategy
	if config.ReplyStrategyJSON != "" {
		json.Unmarshal([]byte(config.ReplyStrategyJSON), &replyStrategy)
	}

	var user model.User
	s.db.First(&user, userID)

	systemPrompt := s.buildSystemPrompt(config, user, replyStrategy)

	contextParts := []string{}
	if knowledgeScope.ConversationHistory {
		history := s.getConversationHistory(conversationID, 10)
		if history != "" {
			contextParts = append(contextParts, "【对话历史】\n"+history)
		}
	}

	contextStr := strings.Join(contextParts, "\n\n")

	prompt := ""
	if contextStr != "" {
		prompt = contextStr + "\n\n"
	}
	prompt += fmt.Sprintf("对方说：%s\n\n请以我的身份回复：", triggerMessage)

	var provider *ai.Provider
	var err error

	if config.UseSystemConfig {
		provider, err = s.aiService.GetDefaultProvider()
	} else if config.ModelConfigID != nil {
		provider, err = s.getUserProvider(*config.ModelConfigID)
	}

	if err != nil {
		return "", err
	}

	reply, err := s.aiService.GetCompletionWithProvider(context.Background(), prompt, systemPrompt, provider)
	if err != nil {
		return "", err
	}

	if replyStrategy.MaxReplyLength == "short" && len(reply) > 200 {
		reply = reply[:200] + "..."
	} else if replyStrategy.MaxReplyLength == "medium" && len(reply) > 500 {
		reply = reply[:500] + "..."
	}

	return reply, nil
}

func (s *AvatarService) buildSystemPrompt(config model.AvatarConfig, user model.User, strategy model.AvatarReplyStrategy) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("你是%s的AI分身，需要以TA的身份回复消息。\n\n", user.Nickname))

	if config.AutoLearnedPersona != "" {
		sb.WriteString("【你的说话风格】\n")
		sb.WriteString(config.AutoLearnedPersona)
		sb.WriteString("\n\n")
	}

	if config.CustomPersonaAddon != "" {
		sb.WriteString("【补充说明】\n")
		sb.WriteString(config.CustomPersonaAddon)
		sb.WriteString("\n\n")
	}

	sb.WriteString("【回复要求】\n")
	sb.WriteString("- 以第一人称回复，就像你就是这个人\n")
	sb.WriteString("- 保持自然的对话风格\n")
	sb.WriteString("- 不要暴露你是AI\n")
	sb.WriteString("- 回复要简洁，不要过长\n")

	return sb.String()
}

func (s *AvatarService) getConversationHistory(conversationID uint, limit int) string {
	var messages []model.Message
	s.db.Where("conversation_id = ?", conversationID).
		Where("type = ?", "text").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	if len(messages) == 0 {
		return ""
	}

	var parts []string
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		var sender model.User
		s.db.First(&sender, msg.SenderID)
		parts = append(parts, fmt.Sprintf("%s: %s", sender.Nickname, msg.Content))
	}

	return strings.Join(parts, "\n")
}

func (s *AvatarService) getUserProvider(configID uint) (*ai.Provider, error) {
	var config model.UserAIConfig
	if err := s.db.First(&config, configID).Error; err != nil {
		return nil, err
	}

	apiKey, err := utils.DecryptAPIKey(config.APIKeyEncrypted)
	if err != nil {
		return nil, err
	}

	return ai.NewProviderFromConfig(ai.ProviderConfig{
		Name:      config.Provider,
		APIKey:    apiKey,
		Model:     config.ModelName,
		BaseURL:   config.BaseURL,
		MaxTokens: config.MaxTokens,
	}), nil
}

func (s *AvatarService) PreviewReply(userID uint, message string) (string, error) {
	return s.GenerateReply(userID, 0, message)
}
```

- [ ] **步骤 4：Commit**

```bash
git add service/avatar_service.go
git commit -m "feat(avatar): add avatar service with persona learning and reply generation"
```

---

## 任务 4：Worker Pool 和限流

**文件：**
- 创建：`service/avatar_worker_pool.go`

- [ ] **步骤 1：创建 Worker Pool**

```go
package service

import (
	"context"
	"fmt"
	"sync"
	"time"
	"qim-server/model"

	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type AvatarTask struct {
	UserID         uint
	ConversationID uint
	TriggerMessage string
	TriggerUserID  uint
	IsGroupChat    bool
	GroupName      string
	TriggerName    string
}

type AvatarWorkerPool struct {
	queue       chan AvatarTask
	workers     int
	limiter     *rate.Limiter
	userLimiters sync.Map
	service     *AvatarService
	db          *gorm.DB
}

func NewAvatarWorkerPool(workers int, globalRPM int, service *AvatarService) *AvatarWorkerPool {
	pool := &AvatarWorkerPool{
		queue:   make(chan AvatarTask, 100),
		workers: workers,
		limiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(globalRPM)), globalRPM),
		service: service,
		db:      service.db,
	}

	for i := 0; i < workers; i++ {
		go pool.run()
	}

	return pool
}

func (p *AvatarWorkerPool) Submit(task AvatarTask) error {
	select {
	case p.queue <- task:
		return nil
	default:
		return fmt.Errorf("队列已满，请稍后重试")
	}
}

func (p *AvatarWorkerPool) run() {
	for task := range p.queue {
		p.process(task)
	}
}

func (p *AvatarWorkerPool) process(task AvatarTask) {
	ctx := context.Background()

	if err := p.limiter.Wait(ctx); err != nil {
		return
	}

	userLimiter := p.getUserLimiter(task.UserID)
	if err := userLimiter.Wait(ctx); err != nil {
		return
	}

	var session model.AvatarSession
	err := p.db.Where("user_id = ? AND conversation_id = ?", task.UserID, task.ConversationID).First(&session).Error
	if err == nil && session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		return
	}

	reply, err := p.service.GenerateReply(task.UserID, task.ConversationID, task.TriggerMessage)
	if err != nil {
		return
	}

	if task.IsGroupChat {
		p.sendPrivateReply(task, reply)
	} else {
		p.sendDirectReply(task, reply)
	}

	now := time.Now()
	p.db.Model(&session).Update("last_reply_at", now)
}

func (p *AvatarWorkerPool) getUserLimiter(userID uint) *rate.Limiter {
	limiterAny, _ := p.userLimiters.LoadOrStore(userID, rate.NewLimiter(rate.Every(time.Minute/10), 10))
	return limiterAny.(*rate.Limiter)
}

func (p *AvatarWorkerPool) sendPrivateReply(task AvatarTask, reply string) {
	// TODO: 实现私聊回复逻辑
	// 1. 找到或创建分身用户与触发者的私聊会话
	// 2. 发送消息，标记 is_avatar_reply: true
	// 3. 通知分身用户
}

func (p *AvatarWorkerPool) sendDirectReply(task AvatarTask, reply string) {
	// TODO: 实现直接回复逻辑
	// 1. 在当前会话中发送消息
	// 2. 标记 is_avatar_reply: true
}
```

- [ ] **步骤 2：Commit**

```bash
git add service/avatar_worker_pool.go
git commit -m "feat(avatar): add avatar worker pool with rate limiting"
```

---

## 任务 5：路由注册和消息处理集成

**文件：**
- 修改：`app/routes.go`
- 修改：`handler/smart_reply_handler.go`

- [ ] **步骤 1：在 routes.go 中注册分身路由**

在 `SetupRoutes` 函数中，在 AI 相关路由部分添加：

```go
// 分身服务
avatarService := service.NewAvatarService(db, globalAIService)
avatarHandler := handler.NewAvatarHandler(db, avatarService)
avatarHandler.RegisterRoutes(authed)
```

需要添加 import：
```go
import "qim-server/service"
```

- [ ] **步骤 2：在 smart_reply_handler.go 中集成分身触发**

在 `HandleMessage` 函数中，添加分身触发检测逻辑：

```go
// 检查是否有用户的分身需要触发
func (e *SmartReplyEngine) checkAvatarTriggers(msg *model.Message, conversation *model.Conversation) {
	var sessions []model.AvatarSession
	e.db.Where("conversation_id = ? AND avatar_enabled = ?", conversation.ID, true).Find(&sessions)

	for _, session := range sessions {
		if e.shouldTriggerAvatar(&session, msg) {
			task := AvatarTask{
				UserID:         session.UserID,
				ConversationID: conversation.ID,
				TriggerMessage: msg.Content,
				TriggerUserID:  msg.SenderID,
				IsGroupChat:    conversation.Type == "group",
			}
			e.avatarWorkerPool.Submit(task)
		}
	}
}

func (e *SmartReplyEngine) shouldTriggerAvatar(session *model.AvatarSession, msg *model.Message) bool {
	if session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		return false
	}

	var config model.AvatarConfig
	if err := e.db.Where("user_id = ?", session.UserID).First(&config).Error; err != nil {
		return false
	}

	if !config.Enabled {
		return false
	}

	var triggerRules model.AvatarTriggerRules
	if config.TriggerRulesJSON != "" {
		json.Unmarshal([]byte(config.TriggerRulesJSON), &triggerRules)
	}

	switch triggerRules.Mode {
	case "mention":
		return strings.Contains(msg.Content, fmt.Sprintf("@%d", session.UserID))
	case "offline":
		var user model.User
		e.db.First(&user, session.UserID)
		return user.Status == "offline"
	case "keyword":
		for _, kw := range triggerRules.Keywords {
			if strings.Contains(msg.Content, kw) {
				return true
			}
		}
		return false
	case "all":
		return true
	default:
		return false
	}
}
```

- [ ] **步骤 3：Commit**

```bash
git add app/routes.go handler/smart_reply_handler.go
git commit -m "feat(avatar): integrate avatar routes and message trigger"
```

---

## 任务 6：验证和测试

- [ ] **步骤 1：编译检查**

```bash
cd /Users/gracegaoya/work/project/qim/qim-server && go build ./...
```

- [ ] **步骤 2：运行服务**

```bash
cd /Users/gracegaoya/work/project/qim/qim-server && go run main.go
```

- [ ] **步骤 3：API 测试**

测试以下 API：
- POST /api/v1/avatar/config - 创建分身配置
- GET /api/v1/avatar/config - 获取分身配置
- PUT /api/v1/avatar/config - 更新分身配置
- POST /api/v1/avatar/learn-persona - 触发学习
- GET /api/v1/avatar/learn-status - 查询学习状态
- PUT /api/v1/avatar/sessions/:convId - 开启会话分身

- [ ] **步骤 4：最终 Commit**

```bash
git add -A
git commit -m "feat(avatar): complete avatar backend implementation"
```
