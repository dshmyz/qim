package handler

import (
	"io"
	"strconv"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/pkg/upload"

	"github.com/gin-gonic/gin"
)

// InitUploadRequest 初始化上传请求
// 秒传功能已移除：不再接收 file_hash（前端不再算 MD5）
type InitUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
	FileSize int64  `json:"file_size" binding:"required"`
	FolderID *uint  `json:"folder_id"`
}

// InitUploadResponse 初始化上传响应
// 秒传字段（is_quick_upload / file_id）已移除
type InitUploadResponse struct {
	UploadID       string `json:"upload_id"`
	ChunkSize      int64  `json:"chunk_size"`
	TotalChunks    int    `json:"total_chunks"`
	UploadedChunks []int  `json:"uploaded_chunks"`
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
	// 统一上传开关检查
	if !upload.IsUploadEnabled(func() (string, error) {
		cfg, err := di.GlobalContainer.SystemConfigService.GetConfig("enableFileUpload")
		if err != nil {
			return "", err
		}
		return cfg.Value, nil
	}) {
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

	// 统一大小校验：用 DB 配置的 max_size 限制总分片文件大小
	ucfg := getUploadConfig()
	policy := upload.NewPolicy(ucfg.MaxSize, ucfg.AllowedExtensions, ucfg.EnableTypeCheck)
	if err := policy.ValidateSize(req.FileSize); err != nil {
		maxMB := ucfg.MaxSize / (1024 * 1024)
		response.BadRequest(c, "文件过大，最大支持"+strconv.FormatInt(maxMB, 10)+"MB")
		return
	}

	// 统一类型校验（黑名单始终生效，防止 .html/.exe 等危险文件）
	// 分片上传在 init 阶段无法读取实际内容，仅按扩展名校验；MIME 在 complete 时由合并文件检测
	if err := policy.ValidateType(req.Filename, "application/octet-stream"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	chunkService := di.GlobalContainer.ChunkService
	if chunkService == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	task, uploadedIndexes, err := chunkService.InitUpload(
		userID.(uint),
		req.Filename,
		req.FileSize,
		req.FolderID,
	)
	if err != nil {
		response.InternalServerError(c, "初始化上传失败: "+err.Error())
		return
	}

	response.Success(c, InitUploadResponse{
		UploadID:       task.UploadID,
		ChunkSize:      task.ChunkSize,
		TotalChunks:    task.TotalChunks,
		UploadedChunks: uploadedIndexes,
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

	// 单分片大小限制，防止恶意大分片打爆内存
	if file.Size > upload.DefaultChunkMaxSize {
		maxMB := upload.DefaultChunkMaxSize / (1024 * 1024)
		response.BadRequest(c, "分片过大，最大支持"+strconv.FormatInt(int64(maxMB), 10)+"MB")
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
	// 统一上传开关检查：防止关闭上传后已 init 的分片继续传完
	if !upload.IsUploadEnabled(func() (string, error) {
		cfg, err := di.GlobalContainer.SystemConfigService.GetConfig("enableFileUpload")
		if err != nil {
			return "", err
		}
		return cfg.Value, nil
	}) {
		response.Forbidden(c, "文件上传功能已关闭")
		return
	}

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
