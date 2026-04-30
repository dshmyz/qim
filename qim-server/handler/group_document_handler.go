package handler

import (
	"net/http"
	"qim-server/database"
	"qim-server/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetGroupDocuments(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊不存在"})
		return
	}

	var documents []model.GroupDocument
	db.Preload("File").Where("group_id = ?", group.ID).Find(&documents)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": documents,
	})
}

func AddGroupDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	convID, _ := strconv.ParseUint(convIDStr, 10, 32)

	var req struct {
		FileID uint `json:"file_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊不存在"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是成员"})
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以管理知识库"})
		return
	}

	doc := model.GroupDocument{GroupID: group.ID, FileID: req.FileID}
	db.Create(&doc)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "文档绑定成功", "data": doc})
}

func RemoveGroupDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	fileIDStr := c.Param("file_id")
	convID, _ := strconv.ParseUint(convIDStr, 10, 32)
	fileID, _ := strconv.ParseUint(fileIDStr, 10, 32)

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊不存在"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是成员"})
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以管理知识库"})
		return
	}

	db.Where("group_id = ? AND file_id = ?", group.ID, uint(fileID)).Delete(&model.GroupDocument{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "文档解绑成功"})
}
