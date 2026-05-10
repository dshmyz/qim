# 文件管理增强功能实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现大文件分片上传、上传进度显示和文件预览增强功能

**架构：** 前端使用 Pinia 管理上传状态，Web Worker 计算 MD5；后端提供分片上传、合并、秒传接口；支持断点续传和并发上传

**技术栈：** Vue 3 + TypeScript + Pinia, Go + Gin + GORM, pdfjs-dist

---

## 阶段 1：数据库和后端基础（1-2天）

### 任务 1：创建数据库模型

**文件：**
- 创建：`qim-server/model/chunk.go`
- 修改：`qim-server/model/model.go`

- [ ] **步骤 1：创建分片模型文件**

创建文件 `qim-server/model/chunk.go`：

```go
package model

import "time"

// FileChunk 文件分片记录
type FileChunk struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UploadID    string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_upload_chunk" json:"upload_id"`
	FileHash    string    `gorm:"type:varchar(64);not null;index" json:"file_hash"`
	ChunkIndex  int       `gorm:"not null;uniqueIndex:idx_upload_chunk" json:"chunk_index"`
	ChunkHash   string    `gorm:"type:varchar(64);not null" json:"chunk_hash"`
	ChunkSize   int64     `gorm:"not null" json:"chunk_size"`
	StoragePath string    `gorm:"type:varchar(512)" json:"storage_path"`
	Status      string    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending/uploaded/merged
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UploadTask 上传任务记录
type UploadTask struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UploadID         string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"upload_id"`
	UserID           uint      `gorm:"not null;index" json:"user_id"`
	Filename         string    `gorm:"type:varchar(255);not null" json:"filename"`
	FileSize         int64     `gorm:"not null" json:"file_size"`
	FileHash         string    `gorm:"type:varchar(64)" json:"file_hash"`
	TotalChunks      int       `gorm:"not null" json:"total_chunks"`
	UploadedChunks   int       `gorm:"default:0" json:"uploaded_chunks"`
	FolderID         *uint     `json:"folder_id"`
	Status           string    `gorm:"type:varchar(20);default:'pending';index" json:"status"` // pending/uploading/completed/failed/cancelled
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName 指定表名
func (FileChunk) TableName() string {
	return "file_chunks"
}

func (UploadTask) TableName() string {
	return "upload_tasks"
}
```

- [ ] **步骤 2：注册模型到数据库**

修改文件 `qim-server/model/model.go`，在 `AutoMigrate` 调用中添加新模型：

```go
// 在现有的 AutoMigrate 调用中添加
db.AutoMigrate(
	// ... 现有模型 ...
	&FileChunk{},
	&UploadTask{},
)
```

- [ ] **步骤 3：运行数据库迁移**

运行：`cd qim-server && go run main.go`
预期：服务器启动成功，数据库表创建成功

- [ ] **步骤 4：验证表结构**

运行：`sqlite3 qim-server/qim.db ".schema file_chunks"`
预期：显示 file_chunks 表结构

运行：`sqlite3 qim-server/qim.db ".schema upload_tasks"`
预期：显示 upload_tasks 表结构

- [ ] **步骤 5：Commit**

```bash
git add qim-server/model/chunk.go qim-server/model/model.go
git commit -m "feat: 添加文件分片和上传任务数据模型"
```

---

### 任务 2：创建分片仓库层

**文件：**
- 创建：`qim-server/repository/chunk_repository.go`

- [ ] **步骤 1：创建分片仓库**

创建文件 `qim-server/repository/chunk_repository.go`：

```go
package repository

import (
	"qim-server/model"

	"gorm.io/gorm"
)

type ChunkRepository struct {
	db *gorm.DB
}

func NewChunkRepository(db *gorm.DB) *ChunkRepository {
	return &ChunkRepository{db: db}
}

// CreateUploadTask 创建上传任务
func (r *ChunkRepository) CreateUploadTask(task *model.UploadTask) error {
	return r.db.Create(task).Error
}

// GetUploadTask 获取上传任务
func (r *ChunkRepository) GetUploadTask(uploadID string) (*model.UploadTask, error) {
	var task model.UploadTask
	err := r.db.Where("upload_id = ?", uploadID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateUploadTask 更新上传任务
func (r *ChunkRepository) UpdateUploadTask(task *model.UploadTask) error {
	return r.db.Save(task).Error
}

// DeleteUploadTask 删除上传任务
func (r *ChunkRepository) DeleteUploadTask(uploadID string) error {
	return r.db.Where("upload_id = ?", uploadID).Delete(&model.UploadTask{}).Error
}

// CreateChunk 创建分片记录
func (r *ChunkRepository) CreateChunk(chunk *model.FileChunk) error {
	return r.db.Create(chunk).Error
}

// GetChunk 获取分片
func (r *ChunkRepository) GetChunk(uploadID string, chunkIndex int) (*model.FileChunk, error) {
	var chunk model.FileChunk
	err := r.db.Where("upload_id = ? AND chunk_index = ?", uploadID, chunkIndex).First(&chunk).Error
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

// GetChunksByUploadID 获取上传任务的所有分片
func (r *ChunkRepository) GetChunksByUploadID(uploadID string) ([]model.FileChunk, error) {
	var chunks []model.FileChunk
	err := r.db.Where("upload_id = ?", uploadID).Order("chunk_index").Find(&chunks).Error
	return chunks, err
}

// GetUploadedChunkIndexes 获取已上传的分片索引
func (r *ChunkRepository) GetUploadedChunkIndexes(uploadID string) ([]int, error) {
	var indexes []int
	err := r.db.Model(&model.FileChunk{}).
		Where("upload_id = ? AND status = ?", uploadID, "uploaded").
		Pluck("chunk_index", &indexes).Error
	return indexes, err
}

// UpdateChunkStatus 更新分片状态
func (r *ChunkRepository) UpdateChunkStatus(uploadID string, chunkIndex int, status string) error {
	return r.db.Model(&model.FileChunk{}).
		Where("upload_id = ? AND chunk_index = ?", uploadID, chunkIndex).
		Update("status", status).Error
}

// DeleteChunksByUploadID 删除上传任务的所有分片
func (r *ChunkRepository) DeleteChunksByUploadID(uploadID string) error {
	return r.db.Where("upload_id = ?", uploadID).Delete(&model.FileChunk{}).Error
}

// GetFileByHash 通过文件哈希获取文件
func (r *ChunkRepository) GetFileByHash(fileHash string) (*model.File, error) {
	var file model.File
	err := r.db.Where("file_hash = ?", fileHash).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}
```

- [ ] **步骤 2：运行测试验证编译**

运行：`cd qim-server && go build`
预期：编译成功，无错误

- [ ] **步骤 3：Commit**

```bash
git add qim-server/repository/chunk_repository.go
git commit -m "feat: 添加分片仓库层"
```

---

### 任务 3：创建分片服务层

**文件：**
- 创建：`qim-server/service/chunk_service.go`

- [ ] **步骤 1：创建分片服务**

创建文件 `qim-server/service/chunk_service.go`：

```go
package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"qim-server/model"
	"qim-server/repository"
	"time"

	"gorm.io/gorm"
)

type ChunkService struct {
	db           *gorm.DB
	chunkRepo    *repository.ChunkRepository
	fileRepo     *repository.FileRepository
	storagePath  string
}

func NewChunkService(db *gorm.DB, storagePath string) *ChunkService {
	return &ChunkService{
		db:          db,
		chunkRepo:   repository.NewChunkRepository(db),
		fileRepo:    repository.NewFileRepository(db),
		storagePath: storagePath,
	}
}

// InitUpload 初始化上传
func (s *ChunkService) InitUpload(userID uint, filename string, fileSize int64, fileHash string, folderID *uint) (*model.UploadTask, []int, bool, error) {
	// 检查秒传
	if fileHash != "" {
		existingFile, err := s.chunkRepo.GetFileByHash(fileHash)
		if err == nil && existingFile != nil {
			// 秒传成功
			return nil, nil, true, nil
		}
	}

	// 检查是否有未完成的上传任务
	var existingTask *model.UploadTask
	if fileHash != "" {
		tasks, err := s.chunkRepo.db.Where("user_id = ? AND file_hash = ? AND status IN ?", 
			userID, fileHash, []string{"pending", "uploading"}).Find(&existingTask).Error
		if err == nil && existingTask != nil {
			// 断点续传
			uploadedIndexes, _ := s.chunkRepo.GetUploadedChunkIndexes(existingTask.UploadID)
			return existingTask, uploadedIndexes, false, nil
		}
	}

	// 创建新的上传任务
	uploadID := generateUploadID()
	chunkSize := s.calculateChunkSize(fileSize)
	totalChunks := int((fileSize + chunkSize - 1) / chunkSize)

	task := &model.UploadTask{
		UploadID:       uploadID,
		UserID:         userID,
		Filename:       filename,
		FileSize:       fileSize,
		FileHash:       fileHash,
		TotalChunks:    totalChunks,
		UploadedChunks: 0,
		FolderID:       folderID,
		Status:         "pending",
	}

	if err := s.chunkRepo.CreateUploadTask(task); err != nil {
		return nil, nil, false, err
	}

	return task, []int{}, false, nil
}

// UploadChunk 上传分片
func (s *ChunkService) UploadChunk(uploadID string, chunkIndex int, chunkData []byte, chunkHash string) error {
	// 验证分片哈希
	hash := md5.Sum(chunkData)
	actualHash := hex.EncodeToString(hash[:])
	if actualHash != chunkHash {
		return fmt.Errorf("chunk hash mismatch: expected %s, got %s", chunkHash, actualHash)
	}

	// 保存分片文件
	chunkFilename := fmt.Sprintf("%s_%d.chunk", uploadID, chunkIndex)
	chunkPath := filepath.Join(s.storagePath, "chunks", chunkFilename)
	
	if err := os.MkdirAll(filepath.Dir(chunkPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
		return err
	}

	// 创建分片记录
	chunk := &model.FileChunk{
		UploadID:    uploadID,
		ChunkIndex:  chunkIndex,
		ChunkHash:   chunkHash,
		ChunkSize:   int64(len(chunkData)),
		StoragePath: chunkPath,
		Status:      "uploaded",
	}

	if err := s.chunkRepo.CreateChunk(chunk); err != nil {
		// 清理已保存的文件
		os.Remove(chunkPath)
		return err
	}

	// 更新上传任务进度
	task, err := s.chunkRepo.GetUploadTask(uploadID)
	if err != nil {
		return err
	}

	task.UploadedChunks++
	task.Status = "uploading"
	if err := s.chunkRepo.UpdateUploadTask(task); err != nil {
		return err
	}

	return nil
}

// CompleteUpload 完成上传
func (s *ChunkService) CompleteUpload(uploadID string) (*model.File, error) {
	task, err := s.chunkRepo.GetUploadTask(uploadID)
	if err != nil {
		return nil, err
	}

	// 检查所有分片是否已上传
	if task.UploadedChunks != task.TotalChunks {
		return nil, fmt.Errorf("not all chunks uploaded: %d/%d", task.UploadedChunks, task.TotalChunks)
	}

	// 获取所有分片
	chunks, err := s.chunkRepo.GetChunksByUploadID(uploadID)
	if err != nil {
		return nil, err
	}

	// 合并分片
	finalFilename := fmt.Sprintf("%d_%s", time.Now().Unix(), task.Filename)
	finalPath := filepath.Join(s.storagePath, "uploads", finalFilename)
	
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return nil, err
	}

	finalFile, err := os.Create(finalPath)
	if err != nil {
		return nil, err
	}
	defer finalFile.Close()

	for _, chunk := range chunks {
		chunkFile, err := os.Open(chunk.StoragePath)
		if err != nil {
			return nil, err
		}
		io.Copy(finalFile, chunkFile)
		chunkFile.Close()
	}

	// 计算最终文件哈希
	finalFile.Seek(0, 0)
	hash := md5.New()
	io.Copy(hash, finalFile)
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// 创建文件记录
	file := &model.File{
		Name:         task.Filename,
		OriginalName: task.Filename,
		StoragePath:  "/uploads/" + finalFilename,
		Size:         task.FileSize,
		MimeType:     getMimeType(task.Filename),
		UserID:       task.UserID,
		FolderID:     task.FolderID,
		FileHash:     fileHash,
		UploadID:     uploadID,
	}

	if err := s.fileRepo.Create(file); err != nil {
		return nil, err
	}

	// 更新上传任务状态
	task.Status = "completed"
	s.chunkRepo.UpdateUploadTask(task)

	// 清理临时分片
	for _, chunk := range chunks {
		os.Remove(chunk.StoragePath)
		chunk.Status = "merged"
		s.chunkRepo.UpdateChunkStatus(uploadID, chunk.ChunkIndex, "merged")
	}

	return file, nil
}

// CancelUpload 取消上传
func (s *ChunkService) CancelUpload(uploadID string) error {
	// 获取所有分片
	chunks, err := s.chunkRepo.GetChunksByUploadID(uploadID)
	if err != nil {
		return err
	}

	// 删除分片文件
	for _, chunk := range chunks {
		os.Remove(chunk.StoragePath)
	}

	// 删除分片记录
	if err := s.chunkRepo.DeleteChunksByUploadID(uploadID); err != nil {
		return err
	}

	// 删除上传任务
	return s.chunkRepo.DeleteUploadTask(uploadID)
}

// calculateChunkSize 计算分片大小
func (s *ChunkService) calculateChunkSize(fileSize int64) int64 {
	const (
		MB = 1024 * 1024
	)
	
	if fileSize < 10*MB {
		return fileSize // 不分片
	} else if fileSize < 50*MB {
		return 5 * MB // 5MB 每片
	} else {
		return 10 * MB // 10MB 每片
	}
}

// generateUploadID 生成上传ID
func generateUploadID() string {
	return fmt.Sprintf("upload_%d", time.Now().UnixNano())
}

// getMimeType 获取文件MIME类型
func getMimeType(filename string) string {
	ext := filepath.Ext(filename)
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
		".zip":  "application/zip",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}
	
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
```

- [ ] **步骤 2：运行测试验证编译**

运行：`cd qim-server && go build`
预期：编译成功，无错误

- [ ] **步骤 3：Commit**

```bash
git add qim-server/service/chunk_service.go
git commit -m "feat: 添加分片服务层"
```

---

### 任务 4：创建分片处理器

**文件：**
- 创建：`qim-server/handler/chunk_handler.go`
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：创建分片处理器**

创建文件 `qim-server/handler/chunk_handler.go`：

```go
package handler

import (
	"io"
	"net/http"
	"qim-server/di"
	"qim-server/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// InitUploadRequest 初始化上传请求
type InitUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
	FileSize int64  `json:"file_size" binding:"required"`
	FileHash string `json:"file_hash"`
	FolderID *uint  `json:"folder_id"`
}

// InitUploadResponse 初始化上传响应
type InitUploadResponse struct {
	UploadID         string `json:"upload_id"`
	ChunkSize        int64  `json:"chunk_size"`
	TotalChunks      int    `json:"total_chunks"`
	UploadedChunks   []int  `json:"uploaded_chunks"`
	FileExists       bool   `json:"file_exists"`
	FileID           *uint  `json:"file_id"`
}

// UploadChunkRequest 上传分片请求
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
	userID, _ := c.Get("user_id")

	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	chunkSvc := di.GlobalContainer.ChunkService
	if chunkSvc == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	task, uploadedChunks, fileExists, err := chunkSvc.InitUpload(
		userID.(uint),
		req.Filename,
		req.FileSize,
		req.FileHash,
		req.FolderID,
	)
	if err != nil {
		response.InternalServerError(c, "初始化上传失败")
		return
	}

	// 秒传成功
	if fileExists {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": InitUploadResponse{
				FileExists: true,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": InitUploadResponse{
			UploadID:       task.UploadID,
			ChunkSize:      chunkSvc.calculateChunkSize(task.FileSize),
			TotalChunks:    task.TotalChunks,
			UploadedChunks: uploadedChunks,
			FileExists:     false,
		},
	})
}

// UploadChunk 上传分片
func UploadChunk(c *gin.Context) {
	var req UploadChunkRequest
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "文件上传失败")
		return
	}

	fileData, err := file.Open()
	if err != nil {
		response.InternalServerError(c, "读取文件失败")
		return
	}
	defer fileData.Close()

	chunkData, err := io.ReadAll(fileData)
	if err != nil {
		response.InternalServerError(c, "读取文件内容失败")
		return
	}

	chunkSvc := di.GlobalContainer.ChunkService
	if chunkSvc == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	if err := chunkSvc.UploadChunk(req.UploadID, req.ChunkIndex, chunkData, req.ChunkHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "上传分片失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"chunk_index": req.ChunkIndex,
			"uploaded":    true,
		},
	})
}

// CompleteUpload 完成上传
func CompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	chunkSvc := di.GlobalContainer.ChunkService
	if chunkSvc == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	file, err := chunkSvc.CompleteUpload(req.UploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "完成上传失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"file_id": file.ID,
			"name":    file.Name,
			"size":    file.Size,
			"url":     file.StoragePath,
		},
	})
}

// CancelUpload 取消上传
func CancelUpload(c *gin.Context) {
	var req CancelUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	chunkSvc := di.GlobalContainer.ChunkService
	if chunkSvc == nil {
		response.InternalServerError(c, "分片服务未初始化")
		return
	}

	if err := chunkSvc.CancelUpload(req.UploadID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "取消上传失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "上传已取消",
	})
}
```

- [ ] **步骤 2：注册路由**

修改文件 `qim-server/app/routes.go`，在 `setupAPIRoutes` 函数中添加：

```go
// 文件分片上传
files.POST("/upload/init", handler.InitUpload)
files.POST("/upload/chunk", handler.UploadChunk)
files.POST("/upload/complete", handler.CompleteUpload)
files.POST("/upload/cancel", handler.CancelUpload)
```

- [ ] **步骤 3：在 DI 容器中注册服务**

修改文件 `qim-server/di/container.go`，添加 ChunkService：

```go
type Container struct {
	// ... 现有字段 ...
	ChunkService *service.ChunkService
}

// 在 NewContainer 函数中初始化
func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	// ... 现有初始化 ...
	
	storagePath := "./uploads"
	if cfg.Storage.Local.Path != "" {
		storagePath = cfg.Storage.Local.Path
	}
	
	return &Container{
		// ... 现有字段 ...
		ChunkService: service.NewChunkService(db, storagePath),
	}
}
```

- [ ] **步骤 4：运行测试验证编译**

运行：`cd qim-server && go build`
预期：编译成功，无错误

- [ ] **步骤 5：Commit**

```bash
git add qim-server/handler/chunk_handler.go qim-server/app/routes.go qim-server/di/container.go
git commit -m "feat: 添加分片上传 API 接口"
```

---

## 阶段 2：前端上传功能（2-3天）

### 任务 5：创建上传状态管理

**文件：**
- 创建：`qim-client/src/stores/upload.ts`

- [ ] **步骤 1：创建上传状态 Store**

创建文件 `qim-client/src/stores/upload.ts`：

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface UploadTask {
  uploadId: string
  file: File
  folderId: number | null
  status: 'pending' | 'uploading' | 'completed' | 'failed' | 'cancelled'
  progress: number
  uploadedSize: number
  totalSize: number
  uploadedChunks: number[]
  totalChunks: number
  error?: string
  retryCount: number
  fileId?: number
}

export const useUploadStore = defineStore('upload', () => {
  const tasks = ref<UploadTask[]>([])
  const isExpanded = ref(false)

  const activeTasks = computed(() => 
    tasks.value.filter(t => t.status === 'uploading' || t.status === 'pending')
  )

  const completedTasks = computed(() => 
    tasks.value.filter(t => t.status === 'completed')
  )

  const failedTasks = computed(() => 
    tasks.value.filter(t => t.status === 'failed')
  )

  const totalProgress = computed(() => {
    if (tasks.value.length === 0) return 0
    const totalUploaded = tasks.value.reduce((sum, t) => sum + t.uploadedSize, 0)
    const totalSize = tasks.value.reduce((sum, t) => sum + t.totalSize, 0)
    return totalSize > 0 ? Math.round((totalUploaded / totalSize) * 100) : 0
  })

  function addTask(file: File, folderId?: number): string {
    const uploadId = `upload_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    
    const task: UploadTask = {
      uploadId,
      file,
      folderId: folderId || null,
      status: 'pending',
      progress: 0,
      uploadedSize: 0,
      totalSize: file.size,
      uploadedChunks: [],
      totalChunks: 0,
      retryCount: 0
    }
    
    tasks.value.push(task)
    return uploadId
  }

  function updateTask(uploadId: string, updates: Partial<UploadTask>) {
    const index = tasks.value.findIndex(t => t.uploadId === uploadId)
    if (index !== -1) {
      tasks.value[index] = { ...tasks.value[index], ...updates }
    }
  }

  function updateProgress(uploadId: string, progress: number, uploadedSize: number) {
    updateTask(uploadId, { progress, uploadedSize })
  }

  function updateChunkProgress(uploadId: string, chunkIndex: number) {
    const task = tasks.value.find(t => t.uploadId === uploadId)
    if (task && !task.uploadedChunks.includes(chunkIndex)) {
      task.uploadedChunks.push(chunkIndex)
      task.progress = Math.round((task.uploadedChunks.length / task.totalChunks) * 100)
    }
  }

  function markCompleted(uploadId: string, fileId: number) {
    updateTask(uploadId, { 
      status: 'completed', 
      progress: 100, 
      uploadedSize: tasks.value.find(t => t.uploadId === uploadId)?.totalSize || 0,
      fileId 
    })
  }

  function markFailed(uploadId: string, error: string) {
    updateTask(uploadId, { status: 'failed', error })
  }

  function cancelTask(uploadId: string) {
    updateTask(uploadId, { status: 'cancelled' })
  }

  function removeTask(uploadId: string) {
    const index = tasks.value.findIndex(t => t.uploadId === uploadId)
    if (index !== -1) {
      tasks.value.splice(index, 1)
    }
  }

  function clearCompleted() {
    tasks.value = tasks.value.filter(t => t.status !== 'completed')
  }

  function toggleExpanded() {
    isExpanded.value = !isExpanded.value
  }

  return {
    tasks,
    isExpanded,
    activeTasks,
    completedTasks,
    failedTasks,
    totalProgress,
    addTask,
    updateTask,
    updateProgress,
    updateChunkProgress,
    markCompleted,
    markFailed,
    cancelTask,
    removeTask,
    clearCompleted,
    toggleExpanded
  }
})
```

- [ ] **步骤 2：运行类型检查**

运行：`cd qim-client && npm run type-check`
预期：类型检查通过，无错误

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/stores/upload.ts
git commit -m "feat: 添加上传状态管理 Store"
```

---

### 任务 6：创建 MD5 Web Worker

**文件：**
- 创建：`qim-client/src/workers/md5.worker.ts`

- [ ] **步骤 1：创建 MD5 Worker**

创建文件 `qim-client/src/workers/md5.worker.ts`：

```typescript
import SparkMD5 from 'spark-md5'

self.onmessage = function(e: MessageEvent) {
  const { file, chunkSize = 2 * 1024 * 1024 } = e.data
  
  const blobSlice = File.prototype.slice || (File.prototype as any).mozSlice || (File.prototype as any).webkitSlice
  const chunks = Math.ceil(file.size / chunkSize)
  let currentChunk = 0
  const spark = new SparkMD5.ArrayBuffer()
  const fileReader = new FileReader()

  fileReader.onload = function(e) {
    spark.append(e.target?.result as ArrayBuffer)
    currentChunk++

    if (currentChunk < chunks) {
      loadNext()
    } else {
      const hash = spark.end()
      self.postMessage({
        hash,
        progress: 100
      })
    }
  }

  fileReader.onerror = function() {
    self.postMessage({
      error: '文件读取失败'
    })
  }

  function loadNext() {
    const start = currentChunk * chunkSize
    const end = Math.min(start + chunkSize, file.size)
    
    fileReader.readAsArrayBuffer(blobSlice.call(file, start, end))
    
    self.postMessage({
      progress: Math.round((currentChunk / chunks) * 100)
    })
  }

  loadNext()
}

export {}
```

- [ ] **步骤 2：安装依赖**

运行：`cd qim-client && npm install spark-md5 @types/spark-md5 --save`
预期：安装成功

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/workers/md5.worker.ts qim-client/package.json qim-client/package-lock.json
git commit -m "feat: 添加 MD5 计算 Web Worker"
```

---

### 任务 7：创建上传逻辑 Composable

**文件：**
- 创建：`qim-client/src/composables/useFileUpload.ts`

- [ ] **步骤 1：创建上传 Composable**

创建文件 `qim-client/src/composables/useFileUpload.ts`：

```typescript
import { ref } from 'vue'
import { useUploadStore } from '../stores/upload'
import { fileApi } from '../api/file'
import type { FileItem } from '../api/file'

export interface InitResponse {
  uploadId: string
  chunkSize: number
  totalChunks: number
  uploadedChunks: number[]
  fileExists: boolean
  fileId?: number
}

export function useFileUpload() {
  const uploadStore = useUploadStore()
  const isUploading = ref(false)

  // 计算文件 MD5
  async function calculateMD5(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const worker = new Worker(
        new URL('../workers/md5.worker.ts', import.meta.url),
        { type: 'module' }
      )

      worker.onmessage = (e) => {
        if (e.data.hash) {
          resolve(e.data.hash)
          worker.terminate()
        } else if (e.data.error) {
          reject(new Error(e.data.error))
          worker.terminate()
        }
      }

      worker.onerror = (error) => {
        reject(error)
        worker.terminate()
      }

      worker.postMessage({ file, chunkSize: 2 * 1024 * 1024 })
    })
  }

  // 智能分片策略
  function getChunkStrategy(fileSize: number): { chunkSize: number; totalChunks: number } {
    const MB = 1024 * 1024
    
    if (fileSize < 10 * MB) {
      return { chunkSize: fileSize, totalChunks: 1 }
    } else if (fileSize < 50 * MB) {
      const chunkSize = 5 * MB
      return { chunkSize, totalChunks: Math.ceil(fileSize / chunkSize) }
    } else {
      const chunkSize = 10 * MB
      return { chunkSize, totalChunks: Math.ceil(fileSize / chunkSize) }
    }
  }

  // 文件分片
  function splitFile(file: File, chunkSize: number): Blob[] {
    const chunks: Blob[] = []
    let start = 0
    
    while (start < file.size) {
      const end = Math.min(start + chunkSize, file.size)
      chunks.push(file.slice(start, end))
      start = end
    }
    
    return chunks
  }

  // 计算分片 MD5
  async function calculateChunkMD5(chunk: Blob): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => {
        const spark = require('spark-md5').default
        const hash = spark.ArrayBuffer.hash(reader.result as ArrayBuffer)
        resolve(hash)
      }
      reader.onerror = reject
      reader.readAsArrayBuffer(chunk)
    })
  }

  // 初始化上传
  async function initUpload(file: File, folderId?: number): Promise<InitResponse> {
    const fileHash = await calculateMD5(file)
    
    const response = await fileApi.initUpload({
      filename: file.name,
      file_size: file.size,
      file_hash: fileHash,
      folder_id: folderId
    })

    return response.data.data
  }

  // 上传分片
  async function uploadChunk(
    uploadId: string,
    chunk: Blob,
    chunkIndex: number,
    onProgress?: (progress: number) => void
  ): Promise<void> {
    const chunkHash = await calculateChunkMD5(chunk)
    
    const formData = new FormData()
    formData.append('upload_id', uploadId)
    formData.append('chunk_index', chunkIndex.toString())
    formData.append('chunk_hash', chunkHash)
    formData.append('file', chunk)

    await fileApi.uploadChunk(formData)
    
    if (onProgress) {
      onProgress(100)
    }
  }

  // 完成上传
  async function completeUpload(uploadId: string): Promise<FileItem> {
    const response = await fileApi.completeUpload({ upload_id: uploadId })
    return response.data.data
  }

  // 取消上传
  async function cancelUpload(uploadId: string): Promise<void> {
    await fileApi.cancelUpload({ upload_id: uploadId })
  }

  // 上传文件（完整流程）
  async function uploadFile(file: File, folderId?: number): Promise<FileItem | null> {
    isUploading.value = true
    
    try {
      // 1. 初始化上传
      const initResponse = await initUpload(file, folderId)
      
      // 秒传
      if (initResponse.fileExists) {
        uploadStore.markCompleted(file.name, initResponse.fileId!)
        return null
      }

      // 2. 创建上传任务
      const uploadId = uploadStore.addTask(file, folderId)
      uploadStore.updateTask(uploadId, {
        totalChunks: initResponse.totalChunks,
        status: 'uploading'
      })

      // 3. 分片上传
      const { chunkSize } = getChunkStrategy(file.size)
      const chunks = splitFile(file, chunkSize)
      
      // 并发上传（最多3个）
      const concurrency = 3
      const uploadedIndexes = new Set(initResponse.uploadedChunks)
      
      for (let i = 0; i < chunks.length; i += concurrency) {
        const batch = chunks.slice(i, Math.min(i + concurrency, chunks.length))
        
        await Promise.all(
          batch.map(async (chunk, batchIndex) => {
            const chunkIndex = i + batchIndex
            
            // 跳过已上传的分片
            if (uploadedIndexes.has(chunkIndex)) {
              uploadStore.updateChunkProgress(uploadId, chunkIndex)
              return
            }

            // 重试逻辑
            let retryCount = 0
            const maxRetries = 3
            
            while (retryCount < maxRetries) {
              try {
                await uploadChunk(initResponse.uploadId, chunk, chunkIndex, (progress) => {
                  // 更新分片进度
                })
                
                uploadStore.updateChunkProgress(uploadId, chunkIndex)
                break
              } catch (error) {
                retryCount++
                
                if (retryCount >= maxRetries) {
                  uploadStore.markFailed(uploadId, `分片 ${chunkIndex} 上传失败`)
                  throw error
                }
                
                // 等待后重试
                await new Promise(resolve => setTimeout(resolve, 1000 * retryCount))
              }
            }
          })
        )
      }

      // 4. 完成上传
      const fileItem = await completeUpload(initResponse.uploadId)
      uploadStore.markCompleted(uploadId, fileItem.id)
      
      return fileItem
    } catch (error: any) {
      console.error('上传失败:', error)
      throw error
    } finally {
      isUploading.value = false
    }
  }

  return {
    isUploading,
    calculateMD5,
    getChunkStrategy,
    splitFile,
    initUpload,
    uploadChunk,
    completeUpload,
    cancelUpload,
    uploadFile
  }
}
```

- [ ] **步骤 2：添加 API 方法**

修改文件 `qim-client/src/api/file.ts`，添加分片上传相关方法：

```typescript
// 在 fileApi 对象中添加以下方法

// 初始化上传
initUpload(data: {
  filename: string
  file_size: number
  file_hash?: string
  folder_id?: number
}) {
  return api.post<{ code: number; data: InitResponse }>('/api/v1/files/upload/init', data)
},

// 上传分片
uploadChunk(formData: FormData) {
  return api.post<{ code: number; data: { chunk_index: number; uploaded: boolean } }>(
    '/api/v1/files/upload/chunk',
    formData,
    {
      headers: { 'Content-Type': 'multipart/form-data' }
    }
  )
},

// 完成上传
completeUpload(data: { upload_id: string }) {
  return api.post<{ code: number; data: FileItem }>('/api/v1/files/upload/complete', data)
},

// 取消上传
cancelUpload(data: { upload_id: string }) {
  return api.post<{ code: number; message: string }>('/api/v1/files/upload/cancel', data)
}
```

同时在文件顶部添加类型定义：

```typescript
export interface InitResponse {
  upload_id: string
  chunk_size: number
  total_chunks: number
  uploaded_chunks: number[]
  file_exists: boolean
  file_id?: number
}
```

- [ ] **步骤 3：运行类型检查**

运行：`cd qim-client && npm run type-check`
预期：类型检查通过，无错误

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/composables/useFileUpload.ts qim-client/src/api/file.ts
git commit -m "feat: 添加文件上传逻辑 Composable"
```

---

### 任务 8：创建全局上传进度条组件

**文件：**
- 创建：`qim-client/src/components/common/UploadProgressBar.vue`

- [ ] **步骤 1：创建进度条组件**

创建文件 `qim-client/src/components/common/UploadProgressBar.vue`：

```vue
<template>
  <Teleport to="body">
    <Transition name="slide-down">
      <div v-if="visible && uploadStore.tasks.length > 0" class="upload-progress-bar">
        <div class="progress-header" @click="uploadStore.toggleExpanded">
          <div class="header-left">
            <i class="fas fa-cloud-upload-alt"></i>
            <span class="header-title">
              {{ uploadStore.activeTasks.length > 0 ? '上传中' : '上传完成' }}
              ({{ uploadStore.completedTasks.length }}/{{ uploadStore.tasks.length }})
            </span>
          </div>
          <div class="header-right">
            <div class="total-progress">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: uploadStore.totalProgress + '%' }"></div>
              </div>
              <span class="progress-text">{{ uploadStore.totalProgress }}%</span>
            </div>
            <button class="toggle-btn">
              <i :class="uploadStore.isExpanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'"></i>
            </button>
          </div>
        </div>

        <Transition name="expand">
          <div v-if="uploadStore.isExpanded" class="progress-body">
            <div class="task-list">
              <div
                v-for="task in uploadStore.tasks"
                :key="task.uploadId"
                class="task-item"
                :class="task.status"
              >
                <div class="task-info">
                  <i class="fas fa-file"></i>
                  <span class="task-name">{{ task.file.name }}</span>
                </div>
                
                <div class="task-status">
                  <template v-if="task.status === 'uploading'">
                    <div class="task-progress">
                      <div class="progress-bar small">
                        <div class="progress-fill" :style="{ width: task.progress + '%' }"></div>
                      </div>
                      <span class="progress-text">{{ task.progress }}%</span>
                    </div>
                    <span class="task-size">{{ formatSize(task.uploadedSize) }} / {{ formatSize(task.totalSize) }}</span>
                    <button class="cancel-btn" @click="handleCancel(task.uploadId)">
                      <i class="fas fa-times"></i>
                    </button>
                  </template>
                  
                  <template v-else-if="task.status === 'completed'">
                    <span class="status-text success">
                      <i class="fas fa-check"></i> 完成
                    </span>
                    <button class="remove-btn" @click="uploadStore.removeTask(task.uploadId)">
                      <i class="fas fa-times"></i>
                    </button>
                  </template>
                  
                  <template v-else-if="task.status === 'failed'">
                    <span class="status-text error">
                      <i class="fas fa-exclamation-circle"></i> 失败
                    </span>
                    <button class="retry-btn" @click="handleRetry(task.uploadId)">
                      <i class="fas fa-redo"></i>
                    </button>
                    <button class="remove-btn" @click="uploadStore.removeTask(task.uploadId)">
                      <i class="fas fa-times"></i>
                    </button>
                  </template>
                </div>
              </div>
            </div>

            <div v-if="uploadStore.completedTasks.length > 0" class="actions">
              <button class="clear-btn" @click="uploadStore.clearCompleted">
                清空已完成
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useUploadStore } from '../../stores/upload'
import { useFileUpload } from '../../composables/useFileUpload'

interface Props {
  visible: boolean
}

const props = defineProps<Props>()
const uploadStore = useUploadStore()
const { cancelUpload, uploadFile } = useFileUpload()

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

async function handleCancel(uploadId: string) {
  try {
    await cancelUpload(uploadId)
    uploadStore.cancelTask(uploadId)
  } catch (error) {
    console.error('取消上传失败:', error)
  }
}

async function handleRetry(uploadId: string) {
  const task = uploadStore.tasks.find(t => t.uploadId === uploadId)
  if (!task) return

  try {
    uploadStore.updateTask(uploadId, { status: 'pending', error: undefined })
    await uploadFile(task.file, task.folderId || undefined)
  } catch (error) {
    console.error('重试上传失败:', error)
  }
}
</script>

<style scoped>
.upload-progress-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  background: var(--modal-bg);
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  z-index: 1000;
}

.progress-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  cursor: pointer;
  transition: background 0.2s;
}

.progress-header:hover {
  background: var(--hover-color);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-title {
  font-weight: 500;
  color: var(--text-color);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.total-progress {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-bar {
  width: 120px;
  height: 6px;
  background: var(--border-color);
  border-radius: 3px;
  overflow: hidden;
}

.progress-bar.small {
  width: 80px;
  height: 4px;
}

.progress-fill {
  height: 100%;
  background: var(--primary-color);
  transition: width 0.3s;
}

.progress-text {
  font-size: 12px;
  color: var(--text-secondary);
  min-width: 35px;
}

.toggle-btn {
  padding: 4px 8px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
}

.progress-body {
  border-top: 1px solid var(--border-color);
  max-height: 300px;
  overflow-y: auto;
}

.task-list {
  padding: 8px 0;
}

.task-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 20px;
  transition: background 0.2s;
}

.task-item:hover {
  background: var(--hover-color);
}

.task-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.task-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-color);
}

.task-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-progress {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-size {
  font-size: 12px;
  color: var(--text-secondary);
}

.status-text {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.status-text.success {
  color: #52c41a;
}

.status-text.error {
  color: #ff4d4f;
}

.cancel-btn,
.remove-btn,
.retry-btn {
  padding: 4px 8px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.2s;
}

.cancel-btn:hover,
.remove-btn:hover {
  color: #ff4d4f;
}

.retry-btn:hover {
  color: var(--primary-color);
}

.actions {
  padding: 8px 20px;
  border-top: 1px solid var(--border-color);
}

.clear-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: none;
  color: var(--text-color);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.clear-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

/* 动画 */
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
}

.slide-down-enter-from,
.slide-down-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}

.expand-enter-active,
.expand-leave-active {
  transition: all 0.3s ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-client/src/components/common/UploadProgressBar.vue
git commit -m "feat: 添加全局上传进度条组件"
```

---

### 任务 9：集成到文件管理应用

**文件：**
- 修改：`qim-client/src/components/apps/FileManagementApp.vue`

- [ ] **步骤 1：集成上传功能**

修改文件 `qim-client/src/components/apps/FileManagementApp.vue`，在 script 部分添加：

```typescript
import { useFileUpload } from '../../composables/useFileUpload'
import { useUploadStore } from '../../stores/upload'
import UploadProgressBar from '../../components/common/UploadProgressBar.vue'

const { uploadFile } = useFileUpload()
const uploadStore = useUploadStore()

// 修改现有的 handleFileUpload 函数
async function handleFileUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files || files.length === 0) return

  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    
    try {
      await uploadFile(file, currentFolderId.value)
      // 刷新文件列表
      await refresh()
    } catch (error) {
      console.error('上传失败:', error)
    }
  }

  // 清空 input
  target.value = ''
}
```

在 template 部分添加进度条组件：

```vue
<!-- 在模板末尾添加 -->
<UploadProgressBar :visible="true" />
```

- [ ] **步骤 2：测试上传功能**

运行：`cd qim-client && npm run dev`
预期：
- 选择文件后显示全局进度条
- 进度条显示上传进度
- 上传完成后文件出现在列表中

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/apps/FileManagementApp.vue
git commit -m "feat: 集成分片上传功能到文件管理应用"
```

---

## 阶段 3：文件预览增强（1天）

### 任务 10：添加 PDF 预览支持

**文件：**
- 修改：`qim-client/src/components/apps/file/FilePreviewModal.vue`

- [ ] **步骤 1：安装 PDF.js**

运行：`cd qim-client && npm install pdfjs-dist --save`
预期：安装成功

- [ ] **步骤 2：增强预览组件**

修改文件 `qim-client/src/components/apps/file/FilePreviewModal.vue`，添加 PDF 预览功能：

```vue
<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click="handleClose">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <div class="header-info">
            <i :class="getFileIcon(file?.mime_type)" class="header-icon"></i>
            <h3 class="header-title" :title="file?.name">{{ file?.name }}</h3>
          </div>
          <button class="modal-close" @click="handleClose">&times;</button>
        </div>
        <div class="modal-body">
          <div class="preview-container">
            <!-- PDF 预览 -->
            <div v-if="isPDF(file?.mime_type)" class="preview-wrapper pdf-preview">
              <div class="pdf-controls">
                <button class="control-btn" @click="prevPage" :disabled="currentPage <= 1">
                  <i class="fas fa-chevron-left"></i>
                </button>
                <span class="page-info">第 {{ currentPage }} / {{ totalPages }} 页</span>
                <button class="control-btn" @click="nextPage" :disabled="currentPage >= totalPages">
                  <i class="fas fa-chevron-right"></i>
                </button>
                <div class="zoom-controls">
                  <button class="control-btn" @click="zoomOut" :disabled="scale <= 0.5">
                    <i class="fas fa-search-minus"></i>
                  </button>
                  <span class="zoom-info">{{ Math.round(scale * 100) }}%</span>
                  <button class="control-btn" @click="zoomIn" :disabled="scale >= 3">
                    <i class="fas fa-search-plus"></i>
                  </button>
                </div>
                <button class="control-btn" @click="toggleFullscreen">
                  <i class="fas fa-expand"></i>
                </button>
              </div>
              <div class="pdf-canvas-wrapper">
                <canvas ref="pdfCanvas"></canvas>
              </div>
            </div>

            <!-- 图片预览 -->
            <div v-else-if="isImage(file?.mime_type)" class="preview-wrapper image-preview">
              <img
                :src="previewUrl"
                :alt="file?.name"
                @error="handlePreviewError"
              />
            </div>

            <!-- 视频预览 -->
            <div v-else-if="isVideo(file?.mime_type)" class="preview-wrapper video-preview">
              <video
                :src="previewUrl"
                controls
                autoplay
                @error="handlePreviewError"
              >
                您的浏览器不支持视频播放
              </video>
            </div>

            <!-- 音频预览 -->
            <div v-else-if="isAudio(file?.mime_type)" class="preview-wrapper audio-preview">
              <div class="audio-icon-wrapper">
                <i :class="getFileIcon(file?.mime_type)" class="audio-large-icon"></i>
              </div>
              <audio
                :src="previewUrl"
                controls
                autoplay
                @error="handlePreviewError"
              >
                您的浏览器不支持音频播放
              </audio>
              <p class="audio-filename">{{ file?.name }}</p>
            </div>

            <!-- 纯文本预览 -->
            <div v-else-if="isText(file?.mime_type, file?.name)" class="preview-wrapper text-preview">
              <pre class="text-content">{{ textContent }}</pre>
            </div>

            <!-- 不支持预览 -->
            <div v-else class="preview-wrapper unsupported-preview">
              <i :class="getFileIcon(file?.mime_type)" class="unsupported-icon"></i>
              <p>此文件类型暂不支持在线预览</p>
              <button class="download-btn" @click="handleDownload">
                <i class="fas fa-download"></i> 下载查看
              </button>
            </div>

            <!-- 加载错误 -->
            <div v-if="previewError" class="error-state">
              <i class="fas fa-exclamation-circle"></i>
              <p>预览加载失败</p>
              <button class="download-btn" @click="handleDownload">
                <i class="fas fa-download"></i> 下载查看
              </button>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <div class="file-meta-info">
            <span class="meta-size">{{ formatFileSize(file?.size) }}</span>
            <span class="meta-divider">&middot;</span>
            <span class="meta-date">{{ formatFileDate(file?.created_at) }}</span>
          </div>
          <div class="footer-actions">
            <button class="action-btn" @click="handleDownload">
              <i class="fas fa-download"></i> 下载
            </button>
            <button class="action-btn" @click="handleShare">
              <i class="fas fa-share-alt"></i> 分享
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, watch, ref, nextTick } from 'vue'
import { type FileItem } from '../../../api/file'
import { API_BASE_URL } from '../../../config'
import * as pdfjsLib from 'pdfjs-dist'

// 设置 PDF.js worker
pdfjsLib.GlobalWorkerOptions.workerSrc = `//cdnjs.cloudflare.com/ajax/libs/pdf.js/${pdfjsLib.version}/pdf.worker.min.js`

interface Props {
  visible: boolean
  file?: FileItem | null
}

const props = withDefaults(defineProps<Props>(), {
  file: null
})

const emit = defineEmits<{
  close: []
  download: [file: FileItem]
  share: [file: FileItem]
}>()

const previewError = ref(false)
const previewUrl = computed(() => {
  if (!props.file) return ''
  const token = localStorage.getItem('token')
  return `${API_BASE_URL}${props.file.storage_path}?token=${token}`
})

// PDF 相关状态
const pdfCanvas = ref<HTMLCanvasElement | null>(null)
const pdfDoc = ref<any>(null)
const currentPage = ref(1)
const totalPages = ref(0)
const scale = ref(1.5)
const textContent = ref('')

// 判断文件类型
function isPDF(mimeType?: string) {
  return mimeType === 'application/pdf'
}

function isImage(mimeType?: string) {
  return mimeType?.startsWith('image/')
}

function isVideo(mimeType?: string) {
  return mimeType?.startsWith('video/')
}

function isAudio(mimeType?: string) {
  return mimeType?.startsWith('audio/')
}

function isText(mimeType?: string, filename?: string) {
  if (mimeType?.startsWith('text/')) return true
  
  const textExtensions = ['.txt', '.log', '.md', '.json', '.xml', '.csv', '.yml', '.yaml']
  const ext = filename?.toLowerCase().substring(filename.lastIndexOf('.'))
  return textExtensions.includes(ext || '')
}

// PDF 相关方法
async function loadPDF() {
  if (!previewUrl.value || !pdfCanvas.value) return

  try {
    const loadingTask = pdfjsLib.getDocument(previewUrl.value)
    pdfDoc.value = await loadingTask.promise
    totalPages.value = pdfDoc.value.numPages
    currentPage.value = 1
    
    await renderPage(currentPage.value)
  } catch (error) {
    console.error('PDF 加载失败:', error)
    previewError.value = true
  }
}

async function renderPage(pageNum: number) {
  if (!pdfDoc.value || !pdfCanvas.value) return

  const page = await pdfDoc.value.getPage(pageNum)
  const viewport = page.getViewport({ scale: scale.value })
  
  const canvas = pdfCanvas.value
  const context = canvas.getContext('2d')
  
  canvas.height = viewport.height
  canvas.width = viewport.width
  
  await page.render({
    canvasContext: context,
    viewport: viewport
  }).promise
}

async function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value--
    await renderPage(currentPage.value)
  }
}

async function nextPage() {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
    await renderPage(currentPage.value)
  }
}

async function zoomIn() {
  if (scale.value < 3) {
    scale.value += 0.25
    await renderPage(currentPage.value)
  }
}

async function zoomOut() {
  if (scale.value > 0.5) {
    scale.value -= 0.25
    await renderPage(currentPage.value)
  }
}

function toggleFullscreen() {
  const wrapper = pdfCanvas.value?.parentElement
  if (wrapper?.requestFullscreen) {
    wrapper.requestFullscreen()
  }
}

// 加载文本内容
async function loadTextContent() {
  if (!previewUrl.value) return

  try {
    const response = await fetch(previewUrl.value)
    textContent.value = await response.text()
  } catch (error) {
    console.error('文本加载失败:', error)
    previewError.value = true
  }
}

// 监听文件变化
watch(() => props.visible, async (visible) => {
  if (visible && props.file) {
    previewError.value = false
    currentPage.value = 1
    scale.value = 1.5
    
    await nextTick()
    
    if (isPDF(props.file.mime_type)) {
      await loadPDF()
    } else if (isText(props.file.mime_type, props.file.name)) {
      await loadTextContent()
    }
  }
})

// 其他方法保持不变...
function handleClose() {
  emit('close')
}

function handleDownload() {
  if (props.file) {
    emit('download', props.file)
  }
}

function handleShare() {
  if (props.file) {
    emit('share', props.file)
  }
}

function handlePreviewError() {
  previewError.value = true
}

function getFileIcon(mimeType?: string): string {
  if (!mimeType) return 'fas fa-file'
  
  if (mimeType.includes('pdf')) return 'fas fa-file-pdf'
  if (mimeType.includes('word') || mimeType.includes('document')) return 'fas fa-file-word'
  if (mimeType.includes('excel') || mimeType.includes('sheet')) return 'fas fa-file-excel'
  if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return 'fas fa-file-powerpoint'
  if (mimeType.includes('image')) return 'fas fa-file-image'
  if (mimeType.includes('video')) return 'fas fa-file-video'
  if (mimeType.includes('audio')) return 'fas fa-file-audio'
  if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('archive')) return 'fas fa-file-archive'
  if (mimeType.includes('text')) return 'fas fa-file-alt'
  if (mimeType.includes('code') || mimeType.includes('javascript') || mimeType.includes('json')) return 'fas fa-file-code'
  
  return 'fas fa-file'
}

function formatFileSize(bytes?: number): string {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

function formatFileDate(date?: string): string {
  if (!date) return ''
  return new Date(date).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
/* 保持现有样式，添加新样式 */

.pdf-preview {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.pdf-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.control-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--bg-primary);
  color: var(--text-color);
  cursor: pointer;
  transition: all 0.2s;
}

.control-btn:hover:not(:disabled) {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.control-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info,
.zoom-info {
  font-size: 14px;
  color: var(--text-secondary);
}

.zoom-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.pdf-canvas-wrapper {
  flex: 1;
  overflow: auto;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 20px;
}

.text-preview {
  padding: 20px;
  overflow: auto;
  height: 100%;
}

.text-content {
  font-family: 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  margin: 0;
  color: var(--text-color);
}
</style>
```

- [ ] **步骤 3：测试 PDF 预览**

运行：`cd qim-client && npm run dev`
预期：
- 上传 PDF 文件
- 点击预览，PDF 正常显示
- 支持翻页、缩放功能

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/components/apps/file/FilePreviewModal.vue qim-client/package.json qim-client/package-lock.json
git commit -m "feat: 添加 PDF 和纯文本预览支持"
```

---

## 阶段 4：测试和优化（1天）

### 任务 11：编写集成测试

**文件：**
- 创建：`qim-client/tests/e2e/file-upload.spec.ts`

- [ ] **步骤 1：创建上传测试**

创建文件 `qim-client/tests/e2e/file-upload.spec.ts`：

```typescript
import { test, expect } from '@playwright/test'

test.describe('文件上传功能', () => {
  test.beforeEach(async ({ page }) => {
    // 登录
    await page.goto('/login')
    await page.fill('input[placeholder="用户名"]', 'testuser')
    await page.fill('input[placeholder="密码"]', 'password123')
    await page.click('button:has-text("登录")')
    await page.waitForURL('/')
  })

  test('应该能上传小文件（< 10MB）', async ({ page }) => {
    // 打开文件管理
    await page.click('[data-testid="apps-panel"]')
    await page.click('text=文件箱')
    
    // 上传文件
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles('./tests/fixtures/small-file.txt')
    
    // 验证进度条显示
    await expect(page.locator('.upload-progress-bar')).toBeVisible()
    
    // 等待上传完成
    await expect(page.locator('.task-item.completed')).toBeVisible({ timeout: 10000 })
    
    // 验证文件出现在列表中
    await expect(page.locator('text=small-file.txt')).toBeVisible()
  })

  test('应该能上传中等文件（10-50MB）', async ({ page }) => {
    await page.click('[data-testid="apps-panel"]')
    await page.click('text=文件箱')
    
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles('./tests/fixtures/medium-file.pdf')
    
    // 验证分片上传
    await expect(page.locator('.upload-progress-bar')).toBeVisible()
    
    // 验证进度更新
    const progressBar = page.locator('.progress-fill')
    await expect(progressBar).not.toHaveCSS('width', '0%')
    
    // 等待上传完成
    await expect(page.locator('.task-item.completed')).toBeVisible({ timeout: 30000 })
  })

  test('应该能取消上传', async ({ page }) => {
    await page.click('[data-testid="apps-panel"]')
    await page.click('text=文件箱')
    
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles('./tests/fixtures/large-file.mp4')
    
    // 等待上传开始
    await expect(page.locator('.upload-progress-bar')).toBeVisible()
    
    // 点击取消按钮
    await page.click('.cancel-btn')
    
    // 验证任务被取消
    await expect(page.locator('.task-item.cancelled')).toBeVisible()
  })

  test('应该能预览 PDF 文件', async ({ page }) => {
    await page.click('[data-testid="apps-panel"]')
    await page.click('text=文件箱')
    
    // 上传 PDF
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles('./tests/fixtures/test.pdf')
    await expect(page.locator('.task-item.completed')).toBeVisible({ timeout: 10000 })
    
    // 点击预览
    await page.click('text=test.pdf')
    await page.click('.preview-btn')
    
    // 验证 PDF 预览
    await expect(page.locator('.pdf-preview')).toBeVisible()
    await expect(page.locator('canvas')).toBeVisible()
    
    // 测试翻页
    await page.click('.control-btn:has(.fa-chevron-right)')
    await expect(page.locator('.page-info')).toContainText('2')
    
    // 测试缩放
    await page.click('.control-btn:has(.fa-search-plus)')
    await expect(page.locator('.zoom-info')).toContainText('175%')
  })

  test('应该能预览纯文本文件', async ({ page }) => {
    await page.click('[data-testid="apps-panel"]')
    await page.click('text=文件箱')
    
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles('./tests/fixtures/test.txt')
    await expect(page.locator('.task-item.completed')).toBeVisible({ timeout: 10000 })
    
    // 点击预览
    await page.click('text=test.txt')
    await page.click('.preview-btn')
    
    // 验证文本预览
    await expect(page.locator('.text-preview')).toBeVisible()
    await expect(page.locator('.text-content')).toBeVisible()
  })
})
```

- [ ] **步骤 2：创建测试文件**

创建测试文件目录和文件：

```bash
mkdir -p qim-client/tests/fixtures
echo "This is a small test file" > qim-client/tests/fixtures/small-file.txt
echo "Test content for text preview" > qim-client/tests/fixtures/test.txt
```

- [ ] **步骤 3：运行测试**

运行：`cd qim-client && npm run test:e2e tests/e2e/file-upload.spec.ts`
预期：所有测试通过

- [ ] **步骤 4：Commit**

```bash
git add qim-client/tests/e2e/file-upload.spec.ts qim-client/tests/fixtures/
git commit -m "test: 添加文件上传功能集成测试"
```

---

### 任务 12：性能优化和文档

**文件：**
- 修改：多个文件（性能优化）
- 创建：`docs/file-upload-usage.md`

- [ ] **步骤 1：优化大文件上传性能**

修改文件 `qim-client/src/composables/useFileUpload.ts`，优化并发上传：

```typescript
// 在 uploadFile 函数中，优化并发逻辑
const concurrency = navigator.hardwareConcurrency || 4 // 使用 CPU 核心数
const maxConcurrency = Math.min(concurrency, 5) // 最多 5 个并发

// 使用队列管理并发
const queue: Array<() => Promise<void>> = []
let activeCount = 0

async function processQueue() {
  while (queue.length > 0 && activeCount < maxConcurrency) {
    activeCount++
    const task = queue.shift()
    if (task) {
      await task()
      activeCount--
      processQueue()
    }
  }
}

// 将分片上传任务加入队列
for (let i = 0; i < chunks.length; i++) {
  const chunkIndex = i
  
  if (uploadedIndexes.has(chunkIndex)) {
    uploadStore.updateChunkProgress(uploadId, chunkIndex)
    continue
  }

  queue.push(async () => {
    // 上传逻辑...
  })
}

processQueue()
```

- [ ] **步骤 2：编写使用文档**

创建文件 `docs/file-upload-usage.md`：

```markdown
# 文件上传功能使用指南

## 功能概述

QIM 文件管理支持以下增强功能：
- 大文件分片上传（支持 < 100MB）
- 秒传（避免重复上传）
- 断点续传（网络中断后可继续）
- 全局进度显示
- PDF 和纯文本预览

## 使用方法

### 1. 上传文件

1. 打开文件箱应用
2. 点击"上传"按钮
3. 选择要上传的文件
4. 查看全局进度条了解上传进度

### 2. 分片上传策略

系统会根据文件大小自动选择最优策略：
- **< 10MB**：直接上传，不分片
- **10-50MB**：分片上传，每片 5MB
- **50-100MB**：分片上传，每片 10MB

### 3. 秒传

如果文件已存在（通过 MD5 检测），会自动跳过上传，直接完成。

### 4. 断点续传

如果上传中断，再次上传同一文件时：
1. 系统会检测已上传的分片
2. 只上传未完成的分片
3. 节省时间和带宽

### 5. 取消上传

在上传过程中，可以点击取消按钮中止上传。

### 6. 文件预览

支持的预览格式：
- **PDF**：支持翻页、缩放、全屏
- **图片**：jpg, png, gif 等
- **视频**：mp4, webm 等
- **音频**：mp3, wav 等
- **文本**：txt, log, md, json, xml 等

## 最佳实践

1. **大文件上传**
   - 建议在网络稳定时上传
   - 上传过程中不要关闭浏览器
   - 如果失败，可以重试

2. **批量上传**
   - 支持同时上传多个文件
   - 系统会并发处理（最多 5 个并发）
   - 可以在进度条中查看每个文件的状态

3. **文件管理**
   - 上传前可以创建文件夹分类
   - 上传后可以重命名、移动、删除
   - 支持星标收藏重要文件

## 故障排查

### 上传失败

1. 检查网络连接
2. 检查文件大小是否超限（100MB）
3. 查看浏览器控制台错误信息
4. 尝试刷新页面重新上传

### 预览失败

1. 检查文件是否完整上传
2. 检查文件格式是否支持
3. 尝试下载后本地查看

## 技术细节

### 分片上传流程

```
1. 计算文件 MD5
2. 初始化上传（检查秒传）
3. 分片上传（并发）
4. 完成上传（合并分片）
```

### API 接口

- `POST /api/v1/files/upload/init` - 初始化上传
- `POST /api/v1/files/upload/chunk` - 上传分片
- `POST /api/v1/files/upload/complete` - 完成上传
- `POST /api/v1/files/upload/cancel` - 取消上传

详细 API 文档请参考：`docs/api/file-upload.md`
```

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/composables/useFileUpload.ts docs/file-upload-usage.md
git commit -m "perf: 优化文件上传性能并添加使用文档"
```

---

## 完成检查清单

- [ ] 所有数据库表已创建
- [ ] 后端 API 接口已实现
- [ ] 前端上传功能已集成
- [ ] 全局进度条已实现
- [ ] PDF 预览已实现
- [ ] 纯文本预览已实现
- [ ] 集成测试已通过
- [ ] 性能已优化
- [ ] 使用文档已完成

---

## 执行交接

计划已完成并保存到 `docs/superpowers/plans/2026-05-11-file-management-enhancement.md`。

**两种执行方式：**

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

**选哪种方式？**
