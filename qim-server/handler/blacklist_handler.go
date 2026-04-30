package handler

import (
	"strconv"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetBlacklist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	db := database.GetDB()

	query := db.Model(&model.Blacklist{}).Preload("User")
	if keyword != "" {
		query = query.Joins("JOIN users ON users.id = blacklist.user_id").
			Where("users.username LIKE ? OR users.nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var blacklist []model.Blacklist
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&blacklist)

	response.Success(c, gin.H{
		"list":     blacklist,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func AddToBlacklist(c *gin.Context) {
	var req struct {
		UserID   uint   `json:"user_id" binding:"required"`
		Reason   string `json:"reason"`
		Operator string `json:"operator"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var existing model.Blacklist
	if err := db.Where("user_id = ?", req.UserID).First(&existing).Error; err == nil {
		response.BadRequest(c, "该用户已在黑名单中")
		return
	}

	entry := model.Blacklist{
		UserID:   req.UserID,
		Reason:   req.Reason,
		Operator: req.Operator,
	}

	if err := db.Create(&entry).Error; err != nil {
		response.InternalServerError(c, "添加到黑名单失败")
		return
	}

	response.Success(c, entry)
}

func RemoveBlacklistEntry(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	db := database.GetDB()
	var entry model.Blacklist
	if err := db.First(&entry, uint(id)).Error; err != nil {
		response.NotFound(c, "黑名单记录不存在")
		return
	}

	if err := db.Delete(&entry).Error; err != nil {
		response.InternalServerError(c, "移除失败")
		return
	}

	response.Success(c, gin.H{"message": "移除成功"})
}
