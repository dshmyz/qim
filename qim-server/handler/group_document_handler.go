package handler

import (
	"errors"
	"net/http"
	"strings"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

	// 安全修复：读接口同样校验群成员身份，防止非成员越权查看任意群文档
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}
	if !isGroupMember(db, group.ConversationID, userID) {
		response.Forbidden(c, "您不是该群成员")
		return
	}

	var documents []model.GroupDocument
	db.Preload("File").Where("group_id = ?", group.ID).Find(&documents)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": documents,
	})
}

// isGroupMember 校验用户是否为指定会话的成员（用于群文档读接口的权限校验）
func isGroupMember(db *gorm.DB, conversationID uint, userID interface{}) bool {
	var member model.ConversationMember
	err := db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&member).Error
	return err == nil
}

// requireGroupAdmin 校验当前用户是群成员且具有管理员权限（群主/管理员）。
// 返回 group 和 member，校验失败时已写入响应，调用方应直接 return。
// 注意：单聊（GroupType != "group"）不要求管理员权限，成员即可。
func requireGroupAdmin(c *gin.Context, db *gorm.DB, convID uint) (*model.Group, *model.ConversationMember, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return nil, nil, false
	}

	var group model.Group
	if err := db.Where("conversation_id = ?", convID).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return nil, nil, false
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "您不是成员")
		return nil, nil, false
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以管理知识库")
		return nil, nil, false
	}

	return &group, &member, true
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
	group, _, ok := requireGroupAdmin(c, db, uint(convID))
	if !ok {
		return
	}

	// 安全修复：校验文件归属，防止管理员将他人私有文件挂到群文档（IDOR）
	// 文件必须属于当前用户（上传者本人），或文件已挂载到当前会话空间（群文件）
	var file model.File
	if err := db.Where("id = ? AND (user_id = ? OR (scope_type = ? AND scope_id = ?))",
		req.FileID, userID, "conversation", group.ConversationID).First(&file).Error; err != nil {
		response.NotFound(c, "文件不存在或无权操作")
		return
	}

	// MIME 匹配前去除 "; charset=..." 后缀，避免带 charset 的 MIME 无法精确匹配白名单
	mimeBase := file.MimeType
	if idx := strings.Index(mimeBase, ";"); idx > 0 {
		mimeBase = strings.TrimSpace(mimeBase[:idx])
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
		"text/csv",
		"text/markdown",
	}

	isAllowed := false
	for _, t := range allowedTypes {
		if mimeBase == t {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		response.BadRequest(c, "只支持添加文档类型的文件（PDF、Word、Excel、PPT、TXT等）")
		return
	}

	// 防重复绑定：同一群同一文件不重复添加
	var existing model.GroupDocument
	if err := db.Where("group_id = ? AND file_id = ?", group.ID, req.FileID).First(&existing).Error; err == nil {
		response.BadRequest(c, "该文件已添加到群文档")
		return
	}

	doc := model.GroupDocument{GroupID: group.ID, FileID: req.FileID}
	db.Create(&doc)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "文档绑定成功", "data": doc})
}

func RemoveGroupDocument(c *gin.Context) {
	convIDStr := c.Param("id")
	fileIDStr := c.Param("file_id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	db := database.GetDB()
	group, _, ok := requireGroupAdmin(c, db, uint(convID))
	if !ok {
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

	// 安全修复：读接口同样校验群成员身份，防止非成员越权查看任意群文档
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}
	if !isGroupMember(db, group.ConversationID, userID) {
		response.Forbidden(c, "您不是该群成员")
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
	convIDStr := c.Param("id")
	fileIDStr := c.Param("file_id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	db := database.GetDB()
	group, _, ok := requireGroupAdmin(c, db, uint(convID))
	if !ok {
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

	utils.SafeGoWithLabel("doc-process", func() {
		if err := docSvc.ProcessDocument(doc.ID); err != nil {
			logger.WithModule("Handler").Error("文档处理失败", "doc_id", doc.ID, "error", err)
		}
	})

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
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	db := database.GetDB()
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	// 安全修复：读接口同样校验群成员身份
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}
	if !isGroupMember(db, group.ConversationID, userID) {
		response.Forbidden(c, "您不是该群成员")
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
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	db := database.GetDB()
	group, _, ok := requireGroupAdmin(c, db, uint(convID))
	if !ok {
		return
	}

	var req struct {
		DocumentIDs []uint `json:"document_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 安全修复：校验所有 document_ids 都属于当前群，防止跨群越权操作
	if err := validateDocumentsBelongToGroup(db, group.ID, req.DocumentIDs); err != nil {
		response.BadRequest(c, err.Error())
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
	convIDStr := c.Param("id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	db := database.GetDB()
	group, _, ok := requireGroupAdmin(c, db, uint(convID))
	if !ok {
		return
	}

	var req struct {
		DocumentIDs []uint `json:"document_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 安全修复：校验所有 document_ids 都属于当前群，防止跨群越权操作
	if err := validateDocumentsBelongToGroup(db, group.ID, req.DocumentIDs); err != nil {
		response.BadRequest(c, err.Error())
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

// validateDocumentsBelongToGroup 校验所有 document_ids 都属于指定群。
// 防止群管理员传入其他群的 document_id 触发跨群越权操作。
func validateDocumentsBelongToGroup(db *gorm.DB, groupID uint, documentIDs []uint) error {
	if len(documentIDs) == 0 {
		return errors.New("document_ids 不能为空")
	}
	var count int64
	db.Model(&model.GroupDocument{}).
		Where("id IN ? AND group_id = ?", documentIDs, groupID).
		Count(&count)
	if int(count) != len(documentIDs) {
		return errors.New("部分文档不属于该群")
	}
	return nil
}
