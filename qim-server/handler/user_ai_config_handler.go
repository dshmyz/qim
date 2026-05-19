package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"qim-server/ai"
	"qim-server/model"
	"qim-server/di"
	"qim-server/pkg/response"
	"qim-server/service"

	"github.com/gin-gonic/gin"
)

type UserAIConfigHandler struct{}

func NewUserAIConfigHandler() *UserAIConfigHandler {
	return &UserAIConfigHandler{}
}

func (h *UserAIConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	configGroup := router.Group("/ai/configs")
	{
		configGroup.GET("/my", h.ListMyConfigs)
		configGroup.POST("/my", h.CreateConfig)
		configGroup.PUT("/my/:id", h.UpdateConfig)
		configGroup.DELETE("/my/:id", h.DeleteConfig)
		configGroup.POST("/my/:id/test", h.TestConfig)
	}
}

type ConfigResponse struct {
	ID           uint          `json:"id"`
	ConfigName   string        `json:"config_name"`
	Provider     string        `json:"provider"`
	ModelName    string        `json:"model_name"`
	BaseURL      string        `json:"base_url"`
	Temperature  float64       `json:"temperature"`
	MaxTokens    int           `json:"max_tokens"`
	IsVerified   bool          `json:"is_verified"`
	Overrides    []ai.Override `json:"overrides,omitempty"`
	LastTestedAt *time.Time    `json:"last_tested_at"`
	CreatedAt    time.Time     `json:"created_at"`
}

func toConfigResponse(cfg model.AIConfig) ConfigResponse {
	var overrides []ai.Override
	if cfg.Overrides != "" {
		json.Unmarshal([]byte(cfg.Overrides), &overrides)
	}
	return ConfigResponse{
		ID:           cfg.ID,
		ConfigName:   cfg.ConfigName,
		Provider:     cfg.Provider,
		ModelName:    cfg.ModelName,
		BaseURL:      cfg.BaseURL,
		Temperature:  cfg.Temperature,
		MaxTokens:    cfg.MaxTokens,
		IsVerified:   cfg.IsVerified,
		Overrides:    overrides,
		LastTestedAt: cfg.LastTestedAt,
		CreatedAt:    cfg.CreatedAt,
	}
}

func (h *UserAIConfigHandler) ListMyConfigs(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	svc := di.GlobalContainer.AIConfigService
	configs, total, err := svc.ListUserConfigs(userID, page, pageSize)
	if err != nil {
		response.InternalServerError(c, "查询配置失败")
		return
	}

	responses := make([]ConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = toConfigResponse(cfg)
	}

	response.Success(c, gin.H{
		"list":     responses,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

type CreateConfigRequest struct {
	ConfigName string        `json:"config_name" binding:"required"`
	Provider   string        `json:"provider" binding:"required"`
	APIKey     string        `json:"api_key" binding:"required"`
	ModelName  string        `json:"model_name" binding:"required"`
	BaseURL    string        `json:"base_url"`
	Overrides  []ai.Override `json:"overrides,omitempty"`
}

func (h *UserAIConfigHandler) CreateConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.AIConfigService
	overridesJSON, _ := json.Marshal(req.Overrides)
	config, err := svc.CreateConfig(userID, req.ConfigName, req.Provider, req.APIKey, req.ModelName, req.BaseURL, string(overridesJSON))
	if err != nil {
		if errors.Is(err, service.ErrConfigLimitExceeded) {
			response.BadRequest(c, "配置数量已达上限（5个）")
			return
		}
		response.InternalServerError(c, "创建配置失败")
		return
	}

	response.Success(c, gin.H{
		"id":          config.ID,
		"is_verified": config.IsVerified,
	})
}

func (h *UserAIConfigHandler) UpdateConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.AIConfigService
	config, err := svc.UpdateConfig(userID, uint(id), req.ConfigName, req.Provider, req.APIKey, req.ModelName, req.BaseURL)
	if err != nil {
		if errors.Is(err, service.ErrConfigNotFound) {
			response.NotFound(c, "配置不存在")
			return
		}
		response.InternalServerError(c, "更新配置失败")
		return
	}

	response.Success(c, gin.H{
		"id":          config.ID,
		"is_verified": config.IsVerified,
	})
}

func (h *UserAIConfigHandler) DeleteConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.AIConfigService
	err = svc.DeleteConfig(userID, uint(id))
	if err != nil {
		if errors.Is(err, service.ErrConfigNotFound) {
			response.NotFound(c, "配置不存在")
			return
		}
		if errors.Is(err, service.ErrConfigInUse) {
			response.BadRequest(c, "该配置正在被机器人使用，无法删除")
			return
		}
		response.InternalServerError(c, "删除配置失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

func (h *UserAIConfigHandler) TestConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.AIConfigService
	verified, err := svc.TestConfig(userID, uint(id))
	if err != nil {
		if errors.Is(err, service.ErrConfigNotFound) {
			response.NotFound(c, "配置不存在")
			return
		}
		response.InternalServerError(c, "测试失败")
		return
	}

	if verified {
		response.Success(c, gin.H{"success": true, "message": "连接成功"})
	} else {
		response.Success(c, gin.H{"success": false, "message": "连接失败"})
	}
}
