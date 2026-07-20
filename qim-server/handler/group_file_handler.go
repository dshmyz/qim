package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

// RegisterGroupFileRoutes mounts the group file-space API beneath an authenticated route group.
func RegisterGroupFileRoutes(routes *gin.RouterGroup) {
	routes.GET("/groups/:id/files", GetGroupFiles)
	routes.POST("/groups/:id/files", AttachGroupUpload)
	routes.GET("/groups/:id/files/:file_id/download", DownloadGroupFile)
	routes.POST("/groups/:id/folders", CreateGroupFolder)
	routes.PATCH("/groups/:id/files/:file_id", MoveGroupFile)
	routes.DELETE("/groups/:id/files/:file_id", DeleteGroupFile)
	routes.POST("/groups/:id/files/references", ShareGroupFileReference)
}

func AttachGroupUpload(c *gin.Context) {
	actorID, space, ok := groupFileRequest(c)
	if !ok {
		return
	}
	var request struct {
		FileID   uint  `json:"file_id"`
		FolderID *uint `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.FileID == 0 {
		response.BadRequest(c, "参数错误")
		return
	}

	file, err := di.GlobalContainer.FileSpaceService.AttachUpload(c.Request.Context(), actorID, space, request.FileID, request.FolderID)
	if !respondGroupFileError(c, err) {
		return
	}
	response.Success(c, file)
}

func DownloadGroupFile(c *gin.Context) {
	actorID, space, ok := groupFileRequest(c)
	if !ok {
		return
	}
	fileID, ok := requiredGroupFileID(c, "file_id")
	if !ok {
		return
	}

	file, err := di.GlobalContainer.FileSpaceService.OpenDownload(c.Request.Context(), actorID, space, fileID)
	if !respondGroupFileError(c, err) {
		return
	}
	if di.GlobalContainer.StorageManager == nil {
		response.InternalServerError(c, "存储服务未初始化")
		return
	}
	st, key, supported := di.GlobalContainer.StorageManager.ByPath(file.StoragePath)
	if !supported || st == nil {
		response.InternalServerError(c, "存储类型不支持")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	reader, err := st.Get(ctx, key)
	if err != nil {
		response.InternalServerError(c, "读取文件失败")
		return
	}
	defer reader.Close()

	name := file.OriginalName
	if name == "" {
		name = file.Name
	}
	c.Header("Content-Disposition", sanitizeFilename(name))
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", fmt.Sprintf("%d", file.Size))
	_, _ = io.Copy(c.Writer, reader)
}

func GetGroupFiles(c *gin.Context) {
	actorID, space, ok := groupFileRequest(c)
	if !ok {
		return
	}

	folderID, ok := optionalGroupFileID(c, "folder_id")
	if !ok {
		return
	}
	page := groupFilePage(c.Query("page"), 1)
	pageSize := groupFilePage(c.Query("page_size"), 20)

	items, err := di.GlobalContainer.FileSpaceService.List(c.Request.Context(), actorID, space, service.FileSpaceQuery{
		FolderID: folderID,
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	})
	if !respondGroupFileError(c, err) {
		return
	}

	response.Success(c, gin.H{
		"files":     items.Files,
		"folders":   items.Folders,
		"total":     items.Total,
		"page":      page,
		"page_size": pageSize,
	})
}

func CreateGroupFolder(c *gin.Context) {
	actorID, space, ok := groupFileRequest(c)
	if !ok {
		return
	}
	var request struct {
		Name     string `json:"name"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	folder, err := di.GlobalContainer.FileSpaceService.CreateFolder(c.Request.Context(), actorID, space, request.Name, request.ParentID)
	if !respondGroupFileError(c, err) {
		return
	}
	response.Success(c, folder)
}

func MoveGroupFile(c *gin.Context) {
	actorID, space, ok := groupFileRequest(c)
	if !ok {
		return
	}
	fileID, ok := requiredGroupFileID(c, "file_id")
	if !ok {
		return
	}
	var request struct {
		FolderID *uint `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	err := di.GlobalContainer.FileSpaceService.Move(c.Request.Context(), actorID, space, []uint{fileID}, request.FolderID)
	if !respondGroupFileError(c, err) {
		return
	}
	response.Success(c, nil)
}

func DeleteGroupFile(c *gin.Context) {
	actorID, space, ok := groupFileRequest(c)
	if !ok {
		return
	}
	fileID, ok := requiredGroupFileID(c, "file_id")
	if !ok {
		return
	}

	err := di.GlobalContainer.FileSpaceService.Delete(c.Request.Context(), actorID, space, fileID)
	if !respondGroupFileError(c, err) {
		return
	}
	response.Success(c, nil)
}

func ShareGroupFileReference(c *gin.Context) {
	actorID, space, ok := groupFileRequest(c)
	if !ok {
		return
	}
	var request struct {
		MessageID uint  `json:"message_id"`
		FileID    uint  `json:"file_id"`
		FolderID  *uint `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.MessageID == 0 || request.FileID == 0 {
		response.BadRequest(c, "参数错误")
		return
	}

	file, err := di.GlobalContainer.FileSpaceService.ShareMessageAttachment(c.Request.Context(), actorID, space.ID, request.MessageID, request.FileID, request.FolderID)
	if !respondGroupFileError(c, err) {
		return
	}
	response.Success(c, file)
}

func groupFileRequest(c *gin.Context) (uint, service.FileSpace, bool) {
	actorID, ok := c.Get("user_id")
	userID, isUint := actorID.(uint)
	if !ok || !isUint || userID == 0 {
		response.Unauthorized(c, "未登录")
		return 0, service.FileSpace{}, false
	}
	if di.GlobalContainer == nil || di.GlobalContainer.GroupService == nil || di.GlobalContainer.FileSpaceService == nil {
		response.InternalServerError(c, "群文件服务未初始化")
		return 0, service.FileSpace{}, false
	}

	conversationID, ok := requiredGroupFileID(c, "id")
	if !ok {
		return 0, service.FileSpace{}, false
	}
	group, err := di.GlobalContainer.GroupService.GetGroupByConversationID(conversationID)
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return 0, service.FileSpace{}, false
	}
	return userID, service.FileSpace{Type: "group", ID: group.ID}, true
}

func optionalGroupFileID(c *gin.Context, name string) (*uint, bool) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return nil, true
	}
	id, err := strconv.ParseUint(value, 10, 32)
	if err != nil || id == 0 {
		response.BadRequest(c, "参数错误")
		return nil, false
	}
	result := uint(id)
	return &result, true
}

func requiredGroupFileID(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || id == 0 {
		response.BadRequest(c, "参数错误")
		return 0, false
	}
	return uint(id), true
}

func groupFilePage(value string, fallback int) int {
	page, err := strconv.Atoi(value)
	if err != nil || page < 1 {
		return fallback
	}
	return page
}

func respondGroupFileError(c *gin.Context, err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, service.ErrFileSpaceForbidden) {
		response.Forbidden(c, "无权访问群文件")
		return false
	}
	if errors.Is(err, service.ErrFileSpaceInvalid) {
		response.BadRequest(c, "参数错误")
		return false
	}
	response.InternalServerError(c, "群文件操作失败")
	return false
}
