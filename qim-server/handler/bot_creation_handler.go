package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateBotRequest 创建 Bot 请求体
type CreateBotRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Description    string                 `json:"description" binding:"required"`
	Type           string                 `json:"type" binding:"required"` // ai, custom
	Provider       string                 `json:"provider"`
	CustomModelURL string                 `json:"custom_model_url"`
	LobsterURL     string                 `json:"lobster_url"`
	Avatar         string                 `json:"avatar"`
	IsTemplate     bool                   `json:"is_template"`
	Config         map[string]interface{} `json:"config"`
}

// GetMyBots 获取我创建的 Bot 列表
func GetMyBots(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var bots []model.Bot
	db.Where("creator_id = ?", userID).Order("created_at DESC").Find(&bots)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

// GetMyBotCount 获取我已创建的 Bot 数量
func GetMyBotCount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var count int64
	db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ('custom', 'ai')", userID).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"count": count},
	})
}

// GetTemplates 获取模板 Bot 列表
func GetTemplates(c *gin.Context) {
	db := database.GetDB()

	var bots []model.Bot
	db.Where("is_template = ? AND is_active = ? AND approval_status = ?", true, true, "approved").Find(&bots)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

// CreateBot 创建 Bot
func CreateBot(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	// 检查用户是否已达到创建上限（模板 Bot 不计入限制）
	if !req.IsTemplate {
		var count int64
		db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ('custom', 'ai')", userID).Count(&count)
		if count >= getMaxBotsPerUser(db) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "BOT_LIMIT_EXCEEDED",
				"message": "已达到创建上限，请联系管理员",
			})
			return
		}
	}

	// 构建 Config JSON
	configJSON, _ := json.Marshal(req.Config)
	if req.Config == nil {
		configJSON = []byte("{}")
	}

	// 判断审批状态
	approvalStatus := "approved" // 系统 Bot 和模板 Bot 直接通过
	creatorID := userID.(uint)
	creatorName := ""

	if !req.IsTemplate && creatorID != 0 {
		approvalStatus = "pending" // 用户自建 Bot 需要审批
	}

	bot := model.Bot{
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Avatar:         req.Avatar,
		Config:         string(configJSON),
		IsActive:       true,
		ApprovalStatus: approvalStatus,
		CreatorID:      creatorID,
		CreatorName:    creatorName,
		IsTemplate:     req.IsTemplate,
	}

	db.Create(&bot)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bot,
	})
}

// UpdateMyBot 更新我的 Bot（仅允许待审批和已拒绝状态）
func UpdateMyBot(c *gin.Context) {
	userID, _ := c.Get("user_id")
	botID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND creator_id = ?", botID, userID).First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无权操作"})
		return
	}

	// 仅允许待审批和已拒绝状态的 Bot 编辑
	if bot.ApprovalStatus != "pending" && bot.ApprovalStatus != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅可编辑待审批或已拒绝的 Bot"})
		return
	}

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	configJSON, _ := json.Marshal(req.Config)
	if req.Config == nil {
		configJSON = []byte("{}")
	}

	db.Model(&bot).Updates(map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"avatar":      req.Avatar,
		"config":      string(configJSON),
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": bot})
}

// DeleteMyBot 删除我的 Bot
func DeleteMyBot(c *gin.Context) {
	userID, _ := c.Get("user_id")
	botID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db := database.GetDB()

	result := db.Where("id = ? AND creator_id = ?", botID, userID).Delete(&model.Bot{})
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// getMaxBotsPerUser 从系统配置获取每个用户的最大 Bot 数量
func getMaxBotsPerUser(db *gorm.DB) int64 {
	// 默认 5，后续从 system_configs 表读取
	return 5
}
