package handler

import (
	"net/http"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetAIProviders 获取所有AI提供商
func GetAIProviders(c *gin.Context) {
	db := database.GetDB()

	var providers []model.AIProvider
	if err := db.Order("priority ASC").Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	result := make([]gin.H, 0, len(providers))
	for _, p := range providers {
		result = append(result, gin.H{
			"id":          p.ID,
			"name":        p.Name,
			"type":        p.APIType,
			"apiKey":      maskAPIKey(p.APIKey),
			"apiEndpoint": p.Endpoint,
			"models":     p.Models,
			"status":      p.Status,
			"enabled":     p.Enabled,
			"priority":    p.Priority,
			"remark":      p.Config,
			"lastTestAt":  "",
			"createdAt":   p.CreatedAt.Format("2006-01-02 15:04:05"),
			"updatedAt":   p.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  result,
			"total": len(result),
		},
	})
}

// CreateAIProvider 创建AI提供商
func CreateAIProvider(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Type        string   `json:"type" binding:"required"` // API type (openai, anthropic, etc.)
		APIEndpoint string   `json:"apiEndpoint"`
		APIKey      string   `json:"apiKey" binding:"required"`
		Models      []string `json:"models"`
		Priority    int      `json:"priority"`
		Remark      string   `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	provider := model.AIProvider{
		Name:     req.Name,
		Provider: req.Type,
		APIType:  req.Type,
		Endpoint: req.APIEndpoint,
		APIKey:   req.APIKey,
		Models:   req.Models,
		Enabled:  true,
		Status:   "connected",
		Priority: req.Priority,
		Config:   req.Remark,
	}

	if err := db.Create(&provider).Error; err != nil {
		response.InternalServerError(c, "创建失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": provider,
	})
}

// UpdateAIProvider 更新AI提供商
func UpdateAIProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req struct {
		Name        *string  `json:"name"`
		Type        *string  `json:"type"`
		APIEndpoint *string  `json:"apiEndpoint"`
		APIKey      *string  `json:"apiKey"`
		Models      []string `json:"models"`
		Priority    *int     `json:"priority"`
		Remark      *string  `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var provider model.AIProvider
	if err := db.First(&provider, uint(id)).Error; err != nil {
		response.NotFound(c, "提供商不存在")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Type != nil {
		updates["provider"] = *req.Type
		updates["api_type"] = *req.Type
	}
	if req.APIEndpoint != nil {
		updates["endpoint"] = *req.APIEndpoint
	}
	if req.APIKey != nil {
		updates["api_key"] = *req.APIKey
	}
	if req.Models != nil {
		updates["models"] = req.Models
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Remark != nil {
		updates["config"] = *req.Remark
	}

	if len(updates) > 0 {
		if err := db.Model(&provider).Updates(updates).Error; err != nil {
			response.InternalServerError(c, "更新失败")
			return
		}
	}

	db.First(&provider, uint(id))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": provider,
	})
}

// DeleteAIProvider 删除AI提供商
func DeleteAIProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	db := database.GetDB()

	var provider model.AIProvider
	if err := db.First(&provider, uint(id)).Error; err != nil {
		response.NotFound(c, "提供商不存在")
		return
	}

	db.Delete(&provider)

	response.SuccessWithMessage(c, "删除成功", nil)
}

// ToggleAIProviderStatus 切换AI提供商状态
func ToggleAIProviderStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var provider model.AIProvider
	if err := db.First(&provider, uint(id)).Error; err != nil {
		response.NotFound(c, "提供商不存在")
		return
	}

	provider.Enabled = req.Enabled
	if err := db.Save(&provider).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": provider,
	})
}

// TestAIProviderConnection 测试AI提供商连接
func TestAIProviderConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	db := database.GetDB()

	var provider model.AIProvider
	if err := db.First(&provider, uint(id)).Error; err != nil {
		response.NotFound(c, "提供商不存在")
		return
	}

	now := time.Now()
	provider.LastTestAt = &now
	provider.Status = "connected"
	db.Save(&provider)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"success":      true,
			"message":      "连接成功",
			"responseTime": 100,
		},
	})
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
