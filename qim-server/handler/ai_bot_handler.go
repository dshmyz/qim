package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIBotHandler struct {
	db *gorm.DB
}

func NewAIBotHandler(db *gorm.DB) *AIBotHandler {
	return &AIBotHandler{db: db}
}

func (h *AIBotHandler) GetAIBots(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var bots []model.Bot
	var total int64

	query := h.db.Model(&model.Bot{}).Where("type = ?", "assistant")

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&bots).Error; err != nil {
		logger.WithModule("aibot").Error("获取 AI 机器人失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 AI 机器人失败",
		})
		return
	}

	// 转换为前端期望的格式
	list := make([]gin.H, 0, len(bots))
	for _, bot := range bots {
		list = append(list, botToFrontend(bot))
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     list,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// botToFrontend 将 Bot 模型转换为前端期望的 JSON 格式
func botToFrontend(bot model.Bot) gin.H {
	var config map[string]interface{}
	if bot.Config != "" {
		json.Unmarshal([]byte(bot.Config), &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	systemPrompt, _ := config["systemPrompt"].(string)
	modelName, _ := config["model"].(string)
	temperature, _ := config["temperature"].(float64)
	maxTokensF, _ := config["maxTokens"].(float64)
	maxTokens := int(maxTokensF)

	status := "inactive"
	if bot.IsActive {
		status = "active"
	}

	return gin.H{
		"id":                bot.ID,
		"name":              bot.Name,
		"avatar":            bot.Avatar,
		"description":       bot.Description,
		"type":              bot.Type,
		"systemPrompt":      systemPrompt,
		"model":             modelName,
		"temperature":       temperature,
		"maxTokens":         maxTokens,
		"status":            status,
		"isActive":          bot.IsActive,
		"conversationCount": 0,
		"creatorId":         bot.CreatorID,
		"creatorName":       bot.CreatorName,
		"createdAt":         bot.CreatedAt.Format("2006-01-02 15:04:05"),
		"isTemplate":        bot.IsTemplate,
	}
}

func (h *AIBotHandler) CreateAIBot(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required"`
		Description  string  `json:"description"`
		Avatar       string  `json:"avatar"`
		SystemPrompt string  `json:"systemPrompt"`
		Model        string  `json:"model"`
		Temperature  float64 `json:"temperature"`
		MaxTokens    int     `json:"maxTokens"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint)

	config := gin.H{
		"systemPrompt": req.SystemPrompt,
		"model":        req.Model,
		"temperature":  req.Temperature,
		"maxTokens":    req.MaxTokens,
	}
	configJSON, _ := json.Marshal(config)

	bot := model.Bot{
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Type:        "assistant",
		Config:      string(configJSON),
		IsActive:    true, // 管理员创建的AI Bot直接启用
		CreatorID:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.Create(&bot).Error; err != nil {
		logger.WithModule("aibot").Error("创建 AI 机器人失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建 AI 机器人失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    botToFrontend(bot),
	})
}

func (h *AIBotHandler) UpdateAIBot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	var req struct {
		Name         *string  `json:"name"`
		Description  *string  `json:"description"`
		Avatar       *string  `json:"avatar"`
		SystemPrompt *string  `json:"systemPrompt"`
		Model        *string  `json:"model"`
		Temperature  *float64 `json:"temperature"`
		MaxTokens    *int     `json:"maxTokens"`
		IsActive     *bool    `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var bot model.Bot
	if err := h.db.Where("id = ? AND type = ?", id, "assistant").First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "AI 机器人不存在",
		})
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if req.SystemPrompt != nil || req.Model != nil || req.Temperature != nil || req.MaxTokens != nil {
		var config map[string]interface{}
		json.Unmarshal([]byte(bot.Config), &config)
		if config == nil {
			config = make(map[string]interface{})
		}

		if req.SystemPrompt != nil {
			config["systemPrompt"] = *req.SystemPrompt
		}
		if req.Model != nil {
			config["model"] = *req.Model
		}
		if req.Temperature != nil {
			config["temperature"] = *req.Temperature
		}
		if req.MaxTokens != nil {
			config["maxTokens"] = *req.MaxTokens
		}

		configJSON, _ := json.Marshal(config)
		updates["config"] = string(configJSON)
	}

	if err := h.db.Model(&bot).Updates(updates).Error; err != nil {
		logger.WithModule("aibot").Error("更新 AI 机器人失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新 AI 机器人失败",
		})
		return
	}

	h.db.First(&bot, id)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    botToFrontend(bot),
	})
}

func (h *AIBotHandler) DeleteAIBot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	var bot model.Bot
	if err := h.db.Where("id = ? AND type = ?", id, "assistant").First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "AI 机器人不存在",
		})
		return
	}

	if err := h.db.Delete(&bot).Error; err != nil {
		logger.WithModule("aibot").Error("删除 AI 机器人失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除 AI 机器人失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

func (h *AIBotHandler) ToggleAIBotStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var bot model.Bot
	if err := h.db.Where("id = ? AND type = ?", id, "assistant").First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "AI 机器人不存在",
		})
		return
	}

	bot.IsActive = req.Status == "active"
	bot.UpdatedAt = time.Now()

	if err := h.db.Save(&bot).Error; err != nil {
		logger.WithModule("aibot").Error("切换 AI 机器人状态失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "切换状态失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "状态已更新",
		"data":    botToFrontend(bot),
	})
}
