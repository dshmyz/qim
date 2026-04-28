package handler

import (
	"net/http"
	"strconv"
	"qim-server/database"
	"qim-server/model"
	"github.com/gin-gonic/gin"
)

// BotApprovalItem 审批列表项
type BotApprovalItem struct {
	model.Bot
	CreatorAvatar   string `json:"creator_avatar"`
	CreatorBotCount int64  `json:"creator_bot_count"`
}

// GetBotApprovals 获取 Bot 审批列表
func GetBotApprovals(c *gin.Context) {
	db := database.GetDB()

	status := c.DefaultQuery("status", "pending") // pending, approved, rejected, all

	var bots []model.Bot
	query := db.Model(&model.Bot{}).Where("creator_id != 0") // 只看用户创建的

	if status != "all" {
		query = query.Where("approval_status = ?", status)
	}

	query.Order("created_at DESC").Find(&bots)

	// 组装审批列表项
	items := make([]BotApprovalItem, 0, len(bots))
	for _, bot := range bots {
		item := BotApprovalItem{Bot: bot}
		// 查询创建者头像
		var creator model.User
		db.Where("id = ?", bot.CreatorID).First(&creator)
		item.CreatorAvatar = creator.Avatar
		// 查询创建者已创建 Bot 数量
		var count int64
		db.Model(&model.Bot{}).Where("creator_id = ?", bot.CreatorID).Count(&count)
		item.CreatorBotCount = count
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": items,
	})
}

// ApproveBot 通过 Bot 申请
func ApproveBot(c *gin.Context) {
	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 Bot ID"})
		return
	}
	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND approval_status = ?", uint(botID), "pending").First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无需审批"})
		return
	}

	if err := db.Model(&bot).Updates(map[string]interface{}{
		"approval_status": "approved",
		"is_active":       true,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "审批失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批通过", "data": bot})
}

// RejectBotRequest 拒绝 Bot 请求
type RejectBotRequest struct {
	Reason string `json:"reason"`
}

// RejectBot 拒绝 Bot 申请
func RejectBot(c *gin.Context) {
	botID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的 Bot ID"})
		return
	}
	db := database.GetDB()

	var bot model.Bot
	if err := db.Where("id = ? AND approval_status = ?", uint(botID), "pending").First(&bot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Bot 不存在或无需审批"})
		return
	}

	var req RejectBotRequest
	c.ShouldBindJSON(&req)

	if err := db.Model(&bot).Updates(map[string]interface{}{
		"approval_status": "rejected",
		"is_active":       false,
		"reject_reason":   req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "操作失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已拒绝", "data": bot})
}
