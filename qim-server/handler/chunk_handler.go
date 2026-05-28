package handler

import (
	"io"
	"strconv"

	"qim-server/di"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// InitUploadRequest 初始化上传请求
type InitUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
	FileSize int64  `json:"file_size" binding:"required"`
	FileHash string `json:"file_hash" binding:"required"`
	FolderID *uint  `json:"folder_id"`
}

// InitUploadResponse 初始化上传响应
type InitUploadResponse struct {
	UploadID        string `json:"upload_id"`
	TotalChunks     int    `json:"total_chunks"`
	UploadedChunks  []int  `json:"uploaded_chunks"`
	IsInstantUpload bool   `json:"is_instant_upload"`
}

// UploadChunkRequest 上传分片请求（multipart form）
type UploadChunkRequest struct {
	UploadID   string `form:"upload_id" binding:"required"`
	ChunkIndex int    `form:"chunk_index" binding:"required"`
	ChunkHash  string `form:"chunk_hash" binding:"required"`
}

// CompleteUploadRequest 完成上传请求
type CompleteUploadRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

// CancelUploadRequest 取消上传请求
type CancelUploadRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

// InitUpload 初始化上传
func InitUpload(c *gin.Context) {
	// 检查文件上传开关
	if cfg, err := di.GlobalContainer.SystemConfigService.GetConfig("enableFileUpload"); err == nil && cfg.Value == "false" {
		response.Forbidden(c, "文件上传功能已关闭")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	chunkService := di.GlobalContainer.ChunkService
	if chunkService == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	task, uploadedIndexes, isInstant, err := chunkService.InitUpload(
		userID.(uint),
		req.Filename,
		req.FileSize,
		req.FileHash,
		req.FolderID,
	)
	if err != nil {
		response.InternalServerError(c, "初始化上传失败: "+err.Error())
		return
	}

	response.Success(c, InitUploadResponse{
		UploadID:        task.UploadID,
		TotalChunks:     task.TotalChunks,
		UploadedChunks:  uploadedIndexes,
		IsInstantUpload: isInstant,
	})
}

// UploadChunk 上传分片
func UploadChunk(c *gin.Context) {
	// 从 multipart form 获取参数
	uploadID := c.PostForm("upload_id")
	if uploadID == "" {
		response.BadRequest(c, "upload_id 参数缺失")
		return
	}

	chunkIndexStr := c.PostForm("chunk_index")
	if chunkIndexStr == "" {
		response.BadRequest(c, "chunk_index 参数缺失")
		return
	}
	chunkIndex := parseInt(chunkIndexStr)

	chunkHash := c.PostForm("chunk_hash")
	if chunkHash == "" {
		response.BadRequest(c, "chunk_hash 参数缺失")
		return
	}

	// 获取分片文件
	file, err := c.FormFile("chunk")
	if err != nil {
		response.BadRequest(c, "分片文件不存在")
		return
	}

	// 打开文件
	fileData, err := file.Open()
	if err != nil {
		response.InternalServerError(c, "打开分片文件失败")
		return
	}
	defer fileData.Close()

	// 读取文件内容
	chunkData, err := io.ReadAll(fileData)
	if err != nil {
		response.InternalServerError(c, "读取分片数据失败")
		return
	}

	chunkService := di.GlobalContainer.ChunkService
	if chunkService == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	err = chunkService.UploadChunk(uploadID, chunkIndex, chunkData, chunkHash)
	if err != nil {
		response.InternalServerError(c, "上传分片失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"chunk_index": chunkIndex,
		"message":     "分片上传成功",
	})
}

// CompleteUpload 完成上传
func CompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	chunkService := di.GlobalContainer.ChunkService
	if chunkService == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	file, err := chunkService.CompleteUpload(req.UploadID)
	if err != nil {
		response.InternalServerError(c, "完成上传失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"id":           file.ID,
		"name":         file.Name,
		"size":         file.Size,
		"mime_type":    file.MimeType,
		"storage_path": file.StoragePath,
		"checksum":     file.Checksum,
		"created_at":   file.CreatedAt,
	})
}

// CancelUpload 取消上传
func CancelUpload(c *gin.Context) {
	var req CancelUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	chunkService := di.GlobalContainer.ChunkService
	if chunkService == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	err := chunkService.CancelUpload(req.UploadID)
	if err != nil {
		response.InternalServerError(c, "取消上传失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "上传已取消",
	})
}

// 辅助函数：字符串转整数
func parseInt(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}
