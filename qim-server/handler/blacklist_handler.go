package handler

import (
	"errors"
	"strconv"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetBlacklist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	blacklistSvc := di.GlobalContainer.BlacklistService
	blacklist, total, err := blacklistSvc.GetBlacklist(page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":     blacklist,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func AddToBlacklist(c *gin.Context) {
	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	operatorName, _ := c.Get("username")
	uname, _ := operatorName.(string)

	entry := model.Blacklist{
		UserID:   req.UserID,
		Reason:   req.Reason,
		Operator: uname,
	}

	blacklistSvc := di.GlobalContainer.BlacklistService
	if err := blacklistSvc.AddToBlacklist(&entry); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response.BadRequest(c, "该用户已在黑名单中")
			return
		}
		response.InternalServerError(c, "添加到黑名单失败")
		return
	}

	response.Success(c, entry)
}

func RemoveBlacklistEntry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	blacklistSvc := di.GlobalContainer.BlacklistService
	if err := blacklistSvc.RemoveFromBlacklist(uint(id)); err != nil {
		response.NotFound(c, "黑名单记录不存在")
		return
	}

	response.Success(c, gin.H{"message": "移除成功"})
}
