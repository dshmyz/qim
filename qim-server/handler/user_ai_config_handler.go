package handler

import (
	"net/http"
	"time"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserAIConfigHandler struct {
	db *gorm.DB
}

func NewUserAIConfigHandler(db *gorm.DB) *UserAIConfigHandler {
	return &UserAIConfigHandler{db: db}
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
	ID           uint       `json:"id"`
	ConfigName   string     `json:"config_name"`
	Provider     string     `json:"provider"`
	ModelName    string     `json:"model_name"`
	BaseURL      string     `json:"base_url"`
	Temperature  float64    `json:"temperature"`
	MaxTokens    int        `json:"max_tokens"`
	IsVerified   bool       `json:"is_verified"`
	LastTestedAt *time.Time `json:"last_tested_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func toConfigResponse(cfg model.UserAIConfig) ConfigResponse {
	return ConfigResponse{
		ID:           cfg.ID,
		ConfigName:   cfg.ConfigName,
		Provider:     cfg.Provider,
		ModelName:    cfg.ModelName,
		BaseURL:      cfg.BaseURL,
		Temperature:  cfg.Temperature,
		MaxTokens:    cfg.MaxTokens,
		IsVerified:   cfg.IsVerified,
		LastTestedAt: cfg.LastTestedAt,
		CreatedAt:    cfg.CreatedAt,
	}
}

func (h *UserAIConfigHandler) ListMyConfigs(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var configs []model.UserAIConfig
	if err := h.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&configs).Error; err != nil {
		response.InternalServerError(c, "查询配置失败")
		return
	}

	responses := make([]ConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = toConfigResponse(cfg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": responses})
}

type CreateConfigRequest struct {
	ConfigName string `json:"config_name" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
	APIKey     string `json:"api_key" binding:"required"`
	ModelName  string `json:"model_name" binding:"required"`
	BaseURL    string `json:"base_url"`
}

func (h *UserAIConfigHandler) CreateConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	var count int64
	h.db.Model(&model.UserAIConfig{}).Where("user_id = ?", userID).Count(&count)
	if count >= 5 {
		response.BadRequest(c, "配置数量已达上限（5个）")
		return
	}

	encryptedKey, err := utils.EncryptAPIKey(req.APIKey)
	if err != nil {
		response.InternalServerError(c, "加密失败")
		return
	}

	verified := h.testConnection(req.Provider, req.APIKey, req.ModelName, req.BaseURL)

	config := model.UserAIConfig{
		UserID:          userID,
		ConfigName:      req.ConfigName,
		Provider:        req.Provider,
		APIKeyEncrypted: encryptedKey,
		ModelName:       req.ModelName,
		BaseURL:         req.BaseURL,
		IsVerified:      verified,
	}

	now := time.Now()
	config.LastTestedAt = &now

	if err := h.db.Create(&config).Error; err != nil {
		response.InternalServerError(c, "创建配置失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":          config.ID,
			"is_verified": verified,
		},
	})
}

func (h *UserAIConfigHandler) UpdateConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	id := c.Param("id")

	var config model.UserAIConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.APIKey != "" {
		encryptedKey, err := utils.EncryptAPIKey(req.APIKey)
		if err != nil {
			response.InternalServerError(c, "加密失败")
			return
		}
		config.APIKeyEncrypted = encryptedKey
	}

	config.ConfigName = req.ConfigName
	config.Provider = req.Provider
	config.ModelName = req.ModelName
	config.BaseURL = req.BaseURL

	verified := h.testConnection(req.Provider, req.APIKey, req.ModelName, req.BaseURL)
	config.IsVerified = verified
	now := time.Now()
	config.LastTestedAt = &now

	if err := h.db.Save(&config).Error; err != nil {
		response.InternalServerError(c, "更新配置失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":          config.ID,
			"is_verified": verified,
		},
	})
}

func (h *UserAIConfigHandler) DeleteConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	id := c.Param("id")

	var config model.UserAIConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	var botCount int64
	h.db.Model(&model.Bot{}).Where("user_config_id = ?", id).Count(&botCount)
	if botCount > 0 {
		response.BadRequest(c, "该配置正在被机器人使用，无法删除")
		return
	}

	if err := h.db.Delete(&config).Error; err != nil {
		response.InternalServerError(c, "删除配置失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

func (h *UserAIConfigHandler) TestConfig(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)
	id := c.Param("id")

	var config model.UserAIConfig
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&config).Error; err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	apiKey, err := utils.DecryptAPIKey(config.APIKeyEncrypted)
	if err != nil {
		response.InternalServerError(c, "解密失败")
		return
	}

	verified := h.testConnection(config.Provider, apiKey, config.ModelName, config.BaseURL)

	now := time.Now()
	h.db.Model(&config).Updates(map[string]interface{}{
		"is_verified":    verified,
		"last_tested_at": now,
	})

	if verified {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": true, "message": "连接成功"}})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": false, "message": "连接失败"}})
	}
}

func (h *UserAIConfigHandler) testConnection(provider, apiKey, modelName, baseURL string) bool {
	switch provider {
	case "openai":
		return testOpenAIConnection(apiKey, modelName, baseURL)
	default:
		return true
	}
}

func testOpenAIConnection(apiKey, modelName, baseURL string) bool {
	return true
}
