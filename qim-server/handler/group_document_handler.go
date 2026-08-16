package handler

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/utils"

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
	isMember, err := isGroupMember(db, group.ConversationID, userID)
	if err != nil {
		response.InternalServerError(c, "校验群成员失败")
		return
	}
	if !isMember {
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

// GetGroupKnowledgeGraph 返回指定群的知识图谱（非管理员接口，群成员即可查看）。
// 群文档向量存于 gracedb 集合 "group_{group.ID}"，这里按集合读回所有向量块，
// 组装成与管理员 GetKnowledgeGraph 一致的 nodes/edges 结构供前端渲染。
func GetGroupKnowledgeGraph(c *gin.Context) {
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

	// 群成员身份校验，防止非成员越权查看任意群知识图谱
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}
	isMember, err := isGroupMember(db, group.ConversationID, userID)
	if err != nil {
		response.InternalServerError(c, "校验群成员失败")
		return
	}
	if !isMember {
		response.Forbidden(c, "您不是该群成员")
		return
	}

	maxNodes := 50
	if maxNodesStr := c.Query("max_nodes"); maxNodesStr != "" {
		fmt.Sscanf(maxNodesStr, "%d", &maxNodes)
	}

	// 从存储图构建真正的拓扑（文档/实体节点 + 关系边 + 实体反查），对齐分身知识图谱。
	// 无图谱数据时返回空结构，不报错。
	docSvc := di.GlobalContainer.GroupDocumentService
	if docSvc == nil {
		response.Success(c, gin.H{
			"nodes":           []interface{}{},
			"edges":           []interface{}{},
			"total_nodes":     0,
			"total_edges":     0,
			"knowledge_count": 0,
		})
		return
	}

	graph, err := docSvc.BuildGroupKnowledgeGraph(group.ID, "", maxNodes)
	if err != nil {
		logger.WithModule("GroupKnowledgeGraph").Error("构建群知识图谱失败", "groupID", group.ID, "error", err)
		response.Success(c, gin.H{
			"nodes":           []interface{}{},
			"edges":           []interface{}{},
			"total_nodes":     0,
			"total_edges":     0,
			"knowledge_count": 0,
		})
		return
	}

	response.Success(c, gin.H{
		"nodes":           graph.Nodes,
		"edges":           graph.Edges,
		"total_nodes":     graph.TotalNodes,
		"total_edges":     graph.TotalEdges,
		"knowledge_count": graph.KnowledgeCount,
	})
}

// isGroupMember 校验用户是否为指定会话的成员（用于群文档读接口的权限校验）
// 返回 (是否成员, 错误)：DB 异常时返回错误，调用方可决定返回 500 而非误判为非成员
func isGroupMember(db *gorm.DB, conversationID uint, userID interface{}) (bool, error) {
	var member model.ConversationMember
	err := db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&member).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	// DB 异常（连接断开、超时等），不应误判为非成员
	return false, err
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

	// 直接上传的图片（截图/照片/扫描件）也允许入知识库，入库时由服务端走视觉 OCR
	// 识别文字后切片向量化；扫描件 PDF 不在此列（维持现有"无文字层"明确报错）。
	if !isAllowed && strings.HasPrefix(mimeBase, "image/") {
		isAllowed = true
	}

	// OOXML 文档（docx/xlsx/pptx）是 ZIP 容器，服务端 DetectMimeType（http.DetectContentType）
	// 一律返回 application/zip，MIME 白名单无法命中。用原始文件扩展名兜底识别，
	// 使这类文档也能绑定进群知识库。
	if !isAllowed {
		ext := strings.ToLower(filepath.Ext(file.OriginalName))
		for _, e := range []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv", ".md", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"} {
			if ext == e {
				isAllowed = true
				break
			}
		}
	}

	if !isAllowed {
		response.BadRequest(c, "只支持添加文档或图片类型的文件（PDF、Word、Excel、PPT、TXT、PNG、JPG等）")
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
	isMember, err := isGroupMember(db, group.ConversationID, userID)
	if err != nil {
		response.InternalServerError(c, "校验群成员失败")
		return
	}
	if !isMember {
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
	isMember, err := isGroupMember(db, group.ConversationID, userID)
	if err != nil {
		response.InternalServerError(c, "校验群成员失败")
		return
	}
	if !isMember {
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
