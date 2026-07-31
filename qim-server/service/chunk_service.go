package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/upload"
	"github.com/dshmyz/qim/qim-server/repository"
	"github.com/dshmyz/qim/qim-server/utils"

	"gorm.io/gorm"
)

// ChunkService 分片上传服务
type ChunkService struct {
	repo    repository.ChunkRepository
	db      *gorm.DB
	storage string // 分片存储目录
	store   StorageAccessor
}

// NewChunkService 创建分片服务实例
func NewChunkService(db *gorm.DB, storage string, store StorageAccessor) *ChunkService {
	return &ChunkService{
		repo:    repository.NewChunkRepository(db),
		db:      db,
		storage: storage,
		store:   store,
	}
}

// InitUpload 初始化上传
// 秒传功能已移除（存在越权风险且前端算 MD5 性能开销大、命中率低）。
// 返回值：上传任务、已上传分片索引列表、错误
// 断点续传：通过 user_id + filename + file_size 匹配未完成任务（不再依赖 file_hash）
func (s *ChunkService) InitUpload(userID uint, filename string, fileSize int64, folderID *uint) (*model.UploadTask, []int, error) {
	ctx := context.Background()

	// 1. 检查断点续传：是否有未完成的上传任务
	// 不再依赖 file_hash 匹配（前端不再算 MD5），改用 user_id + filename + file_size 三元组定位
	var existingTask model.UploadTask
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND filename = ? AND file_size = ? AND status IN ?",
			userID, filename, fileSize, []string{"pending", "uploading"}).
		First(&existingTask).Error

	if err == nil {
		// 断点续传：返回已有任务和已上传分片列表
		uploadedIndexes, err := s.repo.GetUploadedChunkIndexes(ctx, existingTask.UploadID)
		if err != nil {
			return nil, nil, err
		}
		return &existingTask, uploadedIndexes, nil
	}

	// 2. 新上传：创建新的上传任务
	chunkSize := s.calculateChunkSize(fileSize)
	totalChunks := int((fileSize + chunkSize - 1) / chunkSize)
	if totalChunks == 0 {
		totalChunks = 1
	}

	task := &model.UploadTask{
		UploadID:       generateUploadID(),
		UserID:         userID,
		Filename:       filename,
		FileSize:       fileSize,
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
		UploadedChunks: 0,
		FolderID:       folderID,
		Status:         "pending",
	}

	// 使用事务确保数据一致性
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建上传任务
		if err := tx.Create(task).Error; err != nil {
			return err
		}

		// 创建分片记录
		for i := 0; i < totalChunks; i++ {
			start := int64(i) * chunkSize
			end := start + chunkSize
			if end > fileSize {
				end = fileSize
			}

			chunk := &model.FileChunk{
				UploadID:    task.UploadID,
				ChunkIndex:  i,
				ChunkHash:   "",
				ChunkSize:   end - start,
				StoragePath: filepath.Join(s.storage, task.UploadID, fmt.Sprintf("chunk-%d", i)),
				Status:      "pending",
			}

			if err := tx.Create(chunk).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("创建上传任务失败: %w", err)
	}

	return task, []int{}, nil
}

// UploadChunk 上传分片
func (s *ChunkService) UploadChunk(uploadID string, chunkIndex int, chunkData []byte, chunkHash string) error {
	ctx := context.Background()

	// 1. 验证上传任务
	task, err := s.repo.GetUploadTask(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("上传任务不存在: %w", err)
	}

	if task.Status == "completed" {
		return errors.New("上传任务已完成")
	}

	if task.Status == "cancelled" {
		return errors.New("上传任务已取消")
	}

	// 2. 获取分片记录
	chunk, err := s.repo.GetChunk(ctx, uploadID, chunkIndex)
	if err != nil {
		return fmt.Errorf("分片记录不存在: %w", err)
	}

	// 3. 验证分片哈希
	hash := md5.Sum(chunkData)
	actualHash := hex.EncodeToString(hash[:])
	if actualHash != chunkHash {
		return fmt.Errorf("分片哈希不匹配: 期望 %s, 实际 %s", chunkHash, actualHash)
	}

	// 4. 保存分片文件
	chunkDir := filepath.Dir(chunk.StoragePath)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return fmt.Errorf("创建分片目录失败: %w", err)
	}

	if err := os.WriteFile(chunk.StoragePath, chunkData, 0644); err != nil {
		return fmt.Errorf("保存分片文件失败: %w", err)
	}

	// 5. 更新分片状态
	chunk.ChunkHash = chunkHash
	chunk.ChunkSize = int64(len(chunkData))
	chunk.Status = "uploaded"
	if err := s.db.WithContext(ctx).Save(chunk).Error; err != nil {
		return fmt.Errorf("更新分片状态失败: %w", err)
	}

	// 6. 更新上传任务进度
	task.UploadedChunks++
	task.Status = "uploading"

	if err := s.repo.UpdateUploadTask(ctx, task); err != nil {
		return fmt.Errorf("更新上传任务失败: %w", err)
	}

	return nil
}

// CompleteUpload 完成上传
func (s *ChunkService) CompleteUpload(uploadID string) (*model.File, error) {
	ctx := context.Background()

	// 1. 获取上传任务
	task, err := s.repo.GetUploadTask(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("上传任务不存在: %w", err)
	}

	if task.Status == "completed" {
		return nil, errors.New("上传任务已完成")
	}

	// 2. 获取所有分片
	chunks, err := s.repo.GetChunksByUploadID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("获取分片列表失败: %w", err)
	}

	// 3. 验证所有分片已上传
	uploadedCount := 0
	for _, chunk := range chunks {
		if chunk.Status == "uploaded" {
			uploadedCount++
		}
	}

	if uploadedCount != task.TotalChunks {
		return nil, fmt.Errorf("分片未全部上传: %d/%d", uploadedCount, task.TotalChunks)
	}

	// 4. 合并分片到本地临时文件
	// task.Filename 来自客户端，必须清洗防止路径遍历（如 ../逃逸 files/<uploadID>/ 子目录）
	safeFilename := upload.SanitizeFilename(task.Filename)
	finalPath := filepath.Join(s.storage, "files", uploadID, safeFilename)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return nil, fmt.Errorf("创建文件目录失败: %w", err)
	}

	outFile, err := os.Create(finalPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}

	hash := md5.New()
	for _, chunk := range chunks {
		inFile, err := os.Open(chunk.StoragePath)
		if err != nil {
			outFile.Close()
			os.Remove(finalPath)
			return nil, fmt.Errorf("打开分片文件失败: %w", err)
		}
		if _, err := io.Copy(outFile, io.TeeReader(inFile, hash)); err != nil {
			inFile.Close()
			outFile.Close()
			os.Remove(finalPath)
			return nil, fmt.Errorf("合并分片失败: %w", err)
		}
		inFile.Close()
	}
	outFile.Close()

	// 5. 计算最终文件哈希
	checksum := hex.EncodeToString(hash.Sum(nil))

	// 6. 上传合并文件到存储后端（local/S3），key 与普通上传一致
	// 统一使用 safeFilename 提取扩展名，避免与上面清洗逻辑不一致
	now := time.Now()
	ext := filepath.Ext(safeFilename)
	keyFilename := fmt.Sprintf("%s%03d_%d%s", now.Format("20060102150405"), now.UnixMilli()%1000, task.UserID, ext)
	key := fmt.Sprintf("uploads/%s/%s", now.Format("2006/01"), keyFilename)

	mergedFile, err := os.Open(finalPath)
	if err != nil {
		os.Remove(finalPath)
		return nil, fmt.Errorf("打开合并文件失败: %w", err)
	}

	// 读前 512 字节检测真实 MIME（不信任客户端声明的扩展名）
	headBytes := make([]byte, 512)
	n, _ := mergedFile.Read(headBytes)
	headBytes = headBytes[:n]
	detectedMime := upload.DetectMimeType(headBytes)
	if _, err := mergedFile.Seek(0, io.SeekStart); err != nil {
		mergedFile.Close()
		os.Remove(finalPath)
		return nil, fmt.Errorf("重置文件指针失败: %w", err)
	}

	storagePath, err := s.store.Put(ctx, key, mergedFile, task.FileSize, detectedMime)
	mergedFile.Close()
	os.Remove(finalPath) // 清理本地合并临时文件
	if err != nil {
		return nil, fmt.Errorf("上传合并文件失败: %w", err)
	}

	// 7. 创建文件记录
	// 与 FileService.CreateFile 保持一致：分片上传的文件同样落入上传者的个人文件空间
	// （scope_type="user", scope_id=userID），否则 FileSpaceService.AttachUpload 在
	// 用户空间内查不到该文件，会以 ErrFileSpaceForbidden（403 无权访问群文件）拒绝挂载到群。
	// file 记录的 Name/OriginalName 使用清洗后的 safeFilename，与普通上传入口保持一致，
	// 避免用户提交的原始文件名（可能含特殊字符或路径分隔符）落库后被前端/存储层误用。
	file := &model.File{
		UserID:       task.UserID,
		ScopeType:    "user",
		ScopeID:      task.UserID,
		Name:         safeFilename,
		OriginalName: safeFilename,
		Size:         task.FileSize,
		MimeType:     detectedMime,
		StoragePath:  storagePath,
		Checksum:     checksum,
		FolderID:     task.FolderID,
		Source:       "upload",
		SourceID:     uploadID,
	}

	if err := s.db.WithContext(ctx).Create(file).Error; err != nil {
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	// 7. 更新上传任务状态
	task.Status = "completed"
	if err := s.repo.UpdateUploadTask(ctx, task); err != nil {
		return nil, fmt.Errorf("更新上传任务状态失败: %w", err)
	}

	// 8. 清理临时分片文件
	utils.SafeGo(func() { s.cleanupChunks(uploadID) })

	return file, nil
}

// CancelUpload 取消上传
func (s *ChunkService) CancelUpload(uploadID string) error {
	ctx := context.Background()

	// 1. 获取上传任务
	task, err := s.repo.GetUploadTask(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("上传任务不存在: %w", err)
	}

	if task.Status == "completed" {
		return errors.New("上传任务已完成，无法取消")
	}

	// 2. 删除分片文件
	chunks, err := s.repo.GetChunksByUploadID(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("获取分片列表失败: %w", err)
	}

	for _, chunk := range chunks {
		os.Remove(chunk.StoragePath)
	}

	// 3. 删除分片目录
	chunkDir := filepath.Join(s.storage, uploadID)
	os.RemoveAll(chunkDir)

	// 4. 删除分片记录
	if err := s.repo.DeleteChunksByUploadID(ctx, uploadID); err != nil {
		return fmt.Errorf("删除分片记录失败: %w", err)
	}

	// 5. 删除上传任务
	if err := s.repo.DeleteUploadTask(ctx, uploadID); err != nil {
		return fmt.Errorf("删除上传任务失败: %w", err)
	}

	return nil
}

// calculateChunkSize 计算分片大小
// <10MB不分片，10-50MB用5MB，50-100MB用10MB
func (s *ChunkService) calculateChunkSize(fileSize int64) int64 {
	const (
		MB = 1024 * 1024
	)

	if fileSize < 10*MB {
		// 小于10MB不分片
		return fileSize
	} else if fileSize < 50*MB {
		// 10-50MB用5MB分片
		return 5 * MB
	} else {
		// 50MB以上用10MB分片
		return 10 * MB
	}
}

// generateUploadID 生成唯一上传ID
func generateUploadID() string {
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("%d-%d", timestamp, time.Now().Nanosecond())
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// cleanupChunks 清理临时分片文件
func (s *ChunkService) cleanupChunks(uploadID string) {
	ctx := context.Background()

	chunks, err := s.repo.GetChunksByUploadID(ctx, uploadID)
	if err != nil {
		return
	}

	// 删除分片文件
	for _, chunk := range chunks {
		os.Remove(chunk.StoragePath)
	}

	// 删除分片目录
	chunkDir := filepath.Join(s.storage, uploadID)
	os.RemoveAll(chunkDir)

	// 删除分片记录
	s.repo.DeleteChunksByUploadID(ctx, uploadID)
}
