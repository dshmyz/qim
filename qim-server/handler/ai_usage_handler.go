package handler

import (
	"net/http"
	"strconv"
	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

// LogAIUsage 记录 AI 使用日志（内部调用）
func LogAIUsage(userID uint, botID uint, messagePreview string, callType string) {
	db := database.GetDB()
	log := model.AIUsageLog{
		UserID:         userID,
		BotID:          botID,
		MessagePreview: messagePreview,
		CallType:       callType,
	}
	db.Create(&log)
}

// GetAIUsageLogs 获取 AI 使用审计日志（管理员）
func GetAIUsageLogs(c *gin.Context) {
	db := database.GetDB()

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 筛选
	userID := c.Query("user_id")
	botID := c.Query("bot_id")
	callType := c.Query("call_type")

	query := db.Model(&model.AIUsageLog{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if botID != "" {
		query = query.Where("bot_id = ?", botID)
	}
	if callType != "" {
		query = query.Where("call_type = ?", callType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取审计日志失败"})
		return
	}

	var logs []model.AIUsageLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取审计日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     logs,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
