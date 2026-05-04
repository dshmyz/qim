package handler

import (
	"encoding/json"
	"fmt"
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
	if err := db.Where("creator_id = ?", userID).Order("created_at DESC").Find(&bots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 Bot 列表失败"})
		return
	}

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
	if err := db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ('custom', 'ai')", userID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取数量失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"count": count},
	})
}

// GetTemplates 获取模板 Bot 列表
func GetTemplates(c *gin.Context) {
	db := database.GetDB()

	var bots []model.Bot
	if err := db.Where("is_template = ? AND is_active = ? AND approval_status = ?", true, true, "approved").Find(&bots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取模板列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": bots,
	})
}

// CreateBot 创建 Bot
func CreateBot(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户信息错误"})
		return
	}
	db := database.GetDB()

	var req CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	// 校验 Type 字段
	if req.Type != "ai" && req.Type != "custom" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Bot 类型无效"})
		return
	}

	// 普通用户不能创建模板 Bot，强制设置为 false
	req.IsTemplate = false

	// 检查用户是否已达到创建上限
	var count int64
	db.Model(&model.Bot{}).Where("creator_id = ? AND type IN ('custom', 'ai')", userID).Count(&count)
	if count >= getMaxBotsPerUser(db) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "已达到创建上限，请联系管理员",
		})
		return
	}

	// 构建 Config JSON
	configJSON := []byte("{}")
	if req.Config != nil {
		configJSON, _ = json.Marshal(req.Config)
	}

	// 查找创建者用户名
	var creator model.User
	db.Select("nickname").First(&creator, "id = ?", userID)
	creatorName := creator.Nickname
	if creatorName == "" {
		creatorName = creator.Username
	}

	bot := model.Bot{
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Avatar:         req.Avatar,
		Config:         string(configJSON),
		IsActive:       true,
		ApprovalStatus: "pending", // 用户自建 Bot 需要审批
		CreatorID:      userID,
		CreatorName:    creatorName,
		IsTemplate:     false,
	}

	// 开启事务
	tx := db.Begin()

	// 创建 Bot
	if err := tx.Create(&bot).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建 Bot 失败"})
		return
	}

	// 创建虚拟用户
	virtualUser := model.User{
		Username: fmt.Sprintf("bot_%d", bot.ID),
		Nickname: bot.Name,
		Avatar:   bot.Avatar,
		Type:     "bot",
	}
	if err := tx.Create(&virtualUser).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建虚拟用户失败"})
		return
	}

	// 更新 Bot 的 VirtualUserID
	bot.VirtualUserID = &virtualUser.ID
	if err := tx.Save(&bot).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新 Bot 失败"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":              bot.ID,
			"name":            bot.Name,
			"virtual_user_id": virtualUser.ID,
			"approval_status": bot.ApprovalStatus,
		},
	})
}

// UpdateMyBot 更新我的 Bot（仅允许待审批和已拒绝状态）
func UpdateMyBot(c *gin.Context) {
	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(uint)

	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 Bot ID"})
		return
	}

	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND creator_id = ?", uint(botID), userID).First(&bot).Error; err != nil {
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

	// 校验 Type 字段
	if req.Type != "ai" && req.Type != "custom" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Bot 类型无效"})
		return
	}

	configJSON := []byte("{}")
	if req.Config != nil {
		configJSON, _ = json.Marshal(req.Config)
	}

	if err := db.Model(&bot).Updates(map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"type":        req.Type,
		"avatar":      req.Avatar,
		"config":      string(configJSON),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新 Bot 失败"})
		return
	}

	// 重新加载获取最新数据
	db.First(&bot, bot.ID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": bot})
}

// DeleteMyBot 删除我的 Bot
func DeleteMyBot(c *gin.Context) {
	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(uint)

	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 Bot ID"})
		return
	}

	db := database.GetDB()

	result := db.Where("id = ? AND creator_id = ?", uint(botID), userID).Delete(&model.Bot{})
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
