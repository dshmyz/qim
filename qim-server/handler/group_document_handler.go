package handler

import (
	"log"
	"net/http"
	"qim-server/database"
	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetGroupDocuments(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
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
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	var req struct {
		FileID uint `json:"file_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以管理知识库")
		return
	}

	var file model.File
	if err := db.First(&file, req.FileID).Error; err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	allowedTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"text/html",
		"text/csv",
		"text/markdown",
	}

	isAllowed := false
	for _, t := range allowedTypes {
		if file.MimeType == t {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		response.BadRequest(c, "只支持添加文档类型的文件（PDF、Word、Excel、PPT、TXT等）")
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
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}
	fileID, _ := strconv.ParseUint(fileIDStr, 10, 32)

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以管理知识库")
		return
	}

	db.Where("group_id = ? AND file_id = ?", group.ID, uint(fileID)).Delete(&model.GroupDocument{})

	response.SuccessWithMessage(c, "文档解绑成功", nil)
}

func GetGroupDocumentsWithStatus(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	docSvc := di.GlobalContainer.GroupDocumentService
	if docSvc == nil {
		response.InternalServerError(c, "文档服务未初始化")
		return
	}

	results, err := docSvc.GetDocumentsWithStatus(group.ID)
	if err != nil {
		response.InternalServerError(c, "获取文档列表失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": results,
	})
}

func ProcessGroupDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	fileIDStr := c.Param("file_id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}
	fileID, _ := strconv.ParseUint(fileIDStr, 10, 32)

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以管理知识库")
		return
	}

	var doc model.GroupDocument
	if err := db.Where("group_id = ? AND file_id = ?", group.ID, uint(fileID)).First(&doc).Error; err != nil {
		response.NotFound(c, "文档未绑定到该群")
		return
	}

	docSvc := di.GlobalContainer.GroupDocumentService
	if docSvc == nil {
		response.InternalServerError(c, "文档服务未初始化")
		return
	}

	go func() {
		if err := docSvc.ProcessDocument(doc.ID); err != nil {
			log.Printf("[Handler] 文档处理失败 doc_id=%d: %v", doc.ID, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "文档处理任务已提交",
	})
}

func GetDocumentProcessStatus(c *gin.Context) {
	convIDStr := c.Param("id")
	fileIDStr := c.Param("file_id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}
	fileID, _ := strconv.ParseUint(fileIDStr, 10, 32)

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	var doc model.GroupDocument
	if err := db.Where("group_id = ? AND file_id = ?", group.ID, uint(fileID)).First(&doc).Error; err != nil {
		response.NotFound(c, "文档未绑定到该群")
		return
	}

	var status model.DocumentProcessStatus
	db.Where("group_doc_id = ?", doc.ID).Order("created_at DESC").First(&status)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": status,
	})
}

// BatchProcessDocuments 批量处理文档
func BatchProcessDocuments(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以管理知识库")
		return
	}

	var req struct {
		DocumentIDs []uint `json:"document_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	docSvc := di.GlobalContainer.GroupDocumentService
	if docSvc == nil {
		response.InternalServerError(c, "文档服务未初始化")
		return
	}

	results, err := docSvc.BatchProcessDocuments(req.DocumentIDs)
	if err != nil {
		response.InternalServerError(c, "批量处理失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": results,
	})
}

// BatchRetryDocuments 批量重试失败文档
func BatchRetryDocuments(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以管理知识库")
		return
	}

	var req struct {
		DocumentIDs []uint `json:"document_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	docSvc := di.GlobalContainer.GroupDocumentService
	if docSvc == nil {
		response.InternalServerError(c, "文档服务未初始化")
		return
	}

	results, err := docSvc.BatchRetryDocuments(req.DocumentIDs)
	if err != nil {
		response.InternalServerError(c, "批量重试失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": results,
	})
}
