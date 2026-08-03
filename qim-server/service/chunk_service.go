package service

import (
	"context"
	"crypto/md5"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
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
	fileSvc *FileService
}

// NewChunkService 创建分片服务实例
func NewChunkService(db *gorm.DB, storage string, store StorageAccessor) *ChunkService {
	return &ChunkService{
		repo:    repository.NewChunkRepository(db),
		db:      db,
		storage: storage,
		store:   store,
		fileSvc: NewFileService(db),
	}
}

// InitUpload 初始化上传
// 秒传功能已移除（存在越权风险且前端算 MD5 性能开销大、命中率低）。
// 返回值：上传任务、已上传分片索引列表、错误
// 断点续传：只有调用方显式提供原 uploadID 时才恢复任务，避免用文件名和大小猜测文件身份。
func (s *ChunkService) InitUpload(userID uint, filename string, fileSize int64, folderID *uint, resumeUploadID ...string) (*model.UploadTask, []int, error) {
	ctx := context.Background()

	// 1. 只有显式传入 uploadID 才恢复断点任务。
	if len(resumeUploadID) > 0 && resumeUploadID[0] != "" {
		existingTask, err := s.repo.GetUploadTask(ctx, resumeUploadID[0])
		if err != nil {
			return nil, nil, fmt.Errorf("断点任务不存在: %w", err)
		}
		if existingTask.UserID != userID {
			return nil, nil, ErrUploadForbidden
		}
		if existingTask.Status != "pending" && existingTask.Status != "uploading" {
			return nil, nil, fmt.Errorf("断点任务状态不可恢复: %s", existingTask.Status)
		}
		if existingTask.Filename != filename || existingTask.FileSize != fileSize || !sameFolder(existingTask.FolderID, folderID) {
			return nil, nil, errors.New("断点任务与当前文件不匹配")
		}
		uploadedIndexes, err := s.repo.GetUploadedChunkIndexes(ctx, existingTask.UploadID)
		if err != nil {
			return nil, nil, err
		}
		return existingTask, uploadedIndexes, nil
	}

	// 2. 没有显式 uploadID 时始终创建新任务，不猜测用户想恢复哪个文件。
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

func sameFolder(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// ErrUploadForbidden 表示用户无权操作该上传任务（uploadID 不属于该用户）
var ErrUploadForbidden = errors.New("无权操作该上传任务")

// ErrConcurrentComplete 表示另一个并发 CompleteUpload 请求已完成该任务
// 用于抢占式状态转换失败时的标识，防止并发重复创建文件记录
var ErrConcurrentComplete = errors.New("上传任务已被并发请求处理")

// UploadChunk 上传分片
// userID 为当前请求用户 ID，必须与任务所属用户一致，否则返回 ErrUploadForbidden
// 并发安全：使用条件更新确保同一分片不会被重复计数，使用原子 SQL 自增计数器防止竞态
func (s *ChunkService) UploadChunk(userID uint, uploadID string, chunkIndex int, chunkData []byte, chunkHash string) error {
	ctx := context.Background()

	// 1. 验证上传任务
	task, err := s.repo.GetUploadTask(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("上传任务不存在: %w", err)
	}

	// 权限校验：只能操作自己的上传任务，防止 IDOR
	if task.UserID != userID {
		return ErrUploadForbidden
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

	// 幂等检查：已上传的分片直接返回，不重复写盘、不重复计数
	if chunk.Status == "uploaded" {
		return nil
	}

	// 3. 验证分片哈希
	hash := md5.Sum(chunkData)
	actualHash := hex.EncodeToString(hash[:])
	if actualHash != chunkHash {
		// 哈希不匹配可能是因为断点续传命中了内容不同的旧任务（同名同大小但内容不同）。
		// 自动失效当前任务，让前端下次 InitUpload 创建新任务，避免用户卡在哈希错误上。
		if _, delErr := s.repo.MarkTaskCancelled(ctx, uploadID); delErr == nil {
			s.cleanupChunks(uploadID)
		}
		return fmt.Errorf("分片哈希不匹配: 期望 %s, 实际 %s（已重置上传任务，请重新发起上传）", chunkHash, actualHash)
	}

	// 4. 保存分片文件
	chunkDir := filepath.Dir(chunk.StoragePath)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return fmt.Errorf("创建分片目录失败: %w", err)
	}

	if err := os.WriteFile(chunk.StoragePath, chunkData, 0644); err != nil {
		return fmt.Errorf("保存分片文件失败: %w", err)
	}

	// 5. 更新分片 hash 和 size（非条件更新，同一分片的 hash 总是一致的）
	// 先更新这些字段，再用条件更新 status 来控制计数
	if err := s.db.WithContext(ctx).Model(&model.FileChunk{}).
		Where("upload_id = ? AND chunk_index = ?", uploadID, chunkIndex).
		Updates(map[string]interface{}{
			"chunk_hash": chunkHash,
			"chunk_size": int64(len(chunkData)),
		}).Error; err != nil {
		os.Remove(chunk.StoragePath)
		return fmt.Errorf("更新分片哈希失败: %w", err)
	}

	// 6. 条件更新分片状态：仅 pending → uploaded 才计数（防止并发重复计数）
	updated, err := s.repo.ConditionalUpdateChunkStatus(ctx, uploadID, chunkIndex, "pending", "uploaded")
	if err != nil {
		// DB 更新失败，回滚已写入的磁盘文件，避免孤儿分片
		os.Remove(chunk.StoragePath)
		return fmt.Errorf("更新分片状态失败: %w", err)
	}
	if !updated {
		// 已被其他并发请求更新为 uploaded，本次写入是冗余的，幂等返回。
		// 注意：不能删除磁盘文件！两个并发请求写的是同一条 chunk 记录的同一个 StoragePath，
		// 删除会影响成功方的文件，导致 CompleteUpload 时打开分片文件失败。
		// 文件内容相同（hash 一致），留着无害。
		return nil
	}

	// 6. 原子自增已上传分片计数 + 标记任务为 uploading
	// 使用原子 SQL 防止 read-modify-write 竞态（UploadedChunks++ 在并发下会丢失更新）
	if err := s.repo.AtomicIncrementUploadedChunks(ctx, uploadID); err != nil {
		// 计数失败不影响分片已上传的事实，CompleteUpload 会通过实际查询 chunk 状态来校验
		logger.WithModule("ChunkService").Error("原子自增已上传分片数失败", "upload_id", uploadID, "error", err)
	}
	if err := s.repo.MarkTaskUploading(ctx, uploadID); err != nil {
		// 非关键错误，任务可能已经是 uploading 状态
		logger.WithModule("ChunkService").Error("标记任务为 uploading 失败", "upload_id", uploadID, "error", err)
	}

	return nil
}

// CompleteUpload 完成上传
// userID 为当前请求用户 ID，必须与任务所属用户一致，否则返回 ErrUploadForbidden
func (s *ChunkService) CompleteUpload(userID uint, uploadID string) (*model.File, error) {
	ctx := context.Background()

	// 1. 获取上传任务
	task, err := s.repo.GetUploadTask(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("上传任务不存在: %w", err)
	}

	// 权限校验：只能操作自己的上传任务，防止 IDOR
	if task.UserID != userID {
		return nil, ErrUploadForbidden
	}

	if task.Status == "completed" {
		return nil, errors.New("上传任务已完成")
	}

	if task.Status == "cancelled" {
		return nil, errors.New("上传任务已取消")
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

	// 修正计数器：如果原子自增曾失败，task.UploadedChunks 可能小于实际已上传数。
	// CompleteUpload 以实际 chunk 状态为准，同时修正 task.UploadedChunks 确保前端展示一致。
	if task.UploadedChunks != uploadedCount {
		s.db.WithContext(ctx).Model(&model.UploadTask{}).
			Where("upload_id = ?", uploadID).
			Update("uploaded_chunks", uploadedCount)
		task.UploadedChunks = uploadedCount
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

	// 7. 在事务中创建文件记录 + 抢占式标记任务为 completed，防止并发重复完成
	// 事务失败时回滚已上传的存储文件，避免孤儿文件（与普通上传 file_handler.go 的 saved.Cleanup 对齐）
	// file 记录的 scope 由 FileService.CreateFileInTx 统一设置（"user" + UserID）
	file := &model.File{
		UserID:       task.UserID,
		Name:         safeFilename,
		OriginalName: task.Filename,
		Size:         task.FileSize,
		MimeType:     detectedMime,
		StoragePath:  storagePath,
		Checksum:     checksum,
		FolderID:     task.FolderID,
		Source:       "upload",
		SourceID:     uploadID,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 抢占式标记任务为 completed：仅当 status 为 pending/uploading 时成功
		// 防止两个并发 CompleteUpload 请求都通过前置检查后，重复创建文件记录和上传存储
		result := tx.Model(&model.UploadTask{}).
			Where("upload_id = ? AND status IN ?", uploadID, []string{"pending", "uploading"}).
			Update("status", "completed")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// 任务状态已变更（可能被另一个并发 CompleteUpload 抢占，或被 CancelUpload 取消）
			return ErrConcurrentComplete
		}

		// 走 FileService.CreateFileInTx 统一入口，确保 scope 设置和存储路径锁一致
		if err := s.fileSvc.CreateFileInTx(tx, file); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// 事务失败：DB 记录未创建，回滚已上传的存储文件，避免孤儿文件
		if delErr := s.store.DeleteByPath(ctx, storagePath); delErr != nil {
			logger.WithModule("ChunkService").Error("回滚存储文件失败", "path", storagePath, "error", delErr)
		}
		if errors.Is(err, ErrConcurrentComplete) {
			return nil, errors.New("上传任务已被并发请求处理")
		}
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	// 8. 清理临时分片文件
	utils.SafeGo(func() { s.cleanupChunks(uploadID) })

	return file, nil
}

// CancelUpload 取消上传
// userID 为当前请求用户 ID，必须与任务所属用户一致，否则返回 ErrUploadForbidden
// 并发安全：先抢占式标记 cancelled，再删 DB 记录，最后删物理文件
func (s *ChunkService) CancelUpload(userID uint, uploadID string) error {
	ctx := context.Background()

	// 1. 获取上传任务
	task, err := s.repo.GetUploadTask(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("上传任务不存在: %w", err)
	}

	// 权限校验：只能操作自己的上传任务，防止 IDOR
	if task.UserID != userID {
		return ErrUploadForbidden
	}

	if task.Status == "completed" {
		return errors.New("上传任务已完成，无法取消")
	}
	if task.Status == "cancelled" {
		return nil // 幂等：已取消的任务再次取消返回成功
	}

	// 2. 抢占式标记任务为 cancelled，防止与正在进行的 UploadChunk 冲突
	// 抢占成功后，UploadChunk 入口检查 task.Status == "cancelled" 会拒绝新分片上传
	grabbed, err := s.repo.MarkTaskCancelled(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}
	if !grabbed {
		return errors.New("任务状态已变更，无法取消")
	}

	// 3. 获取分片列表（用于后续删除物理文件）
	chunks, err := s.repo.GetChunksByUploadID(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("获取分片列表失败: %w", err)
	}

	// 4. 在事务中删除分片记录和任务记录，确保原子性
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("upload_id = ?", uploadID).Delete(&model.FileChunk{}).Error; err != nil {
			return err
		}
		if err := tx.Where("upload_id = ?", uploadID).Delete(&model.UploadTask{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("删除上传记录失败: %w", err)
	}

	// 5. 删除物理分片文件（DB 删除成功后，物理文件删除失败只是孤儿文件，可由后台 GC 兜底）
	for _, chunk := range chunks {
		if err := os.Remove(chunk.StoragePath); err != nil && !os.IsNotExist(err) {
			logger.WithModule("ChunkService").Error("删除分片文件失败", "path", chunk.StoragePath, "error", err)
		}
	}
	// 兜底：删除整个分片目录，清理可能被并发 UploadChunk 写入但未记录的孤儿分片
	// （并发场景：UploadChunk 在 GetUploadTask 时任务还存在，之后 CancelUpload 删除了 DB 记录，
	// UploadChunk 继续执行 os.WriteFile 写入磁盘，但 ConditionalUpdate 因记录已删除返回 updated=false）
	chunkDir := filepath.Join(s.storage, uploadID)
	if err := os.RemoveAll(chunkDir); err != nil {
		logger.WithModule("ChunkService").Error("删除分片目录失败", "path", chunkDir, "error", err)
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
// 使用 crypto/rand 提供 128 位随机熵，防止攻击者预测 uploadID 后利用 IDOR 接管他人上传任务
func generateUploadID() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		// 极端情况下 crypto/rand 失败，回退到时间戳+随机数（不应发生）
		timestamp := time.Now().UnixNano()
		data := fmt.Sprintf("%d-%d", timestamp, time.Now().Nanosecond())
		hash := md5.Sum([]byte(data))
		return hex.EncodeToString(hash[:])
	}
	return hex.EncodeToString(b)
}

// cleanupChunks 清理临时分片文件
func (s *ChunkService) cleanupChunks(uploadID string) {
	ctx := context.Background()

	chunks, err := s.repo.GetChunksByUploadID(ctx, uploadID)
	if err != nil {
		logger.WithModule("ChunkService").Error("清理分片：获取分片列表失败", "upload_id", uploadID, "error", err)
		return
	}

	// 删除分片文件
	for _, chunk := range chunks {
		if err := os.Remove(chunk.StoragePath); err != nil && !os.IsNotExist(err) {
			logger.WithModule("ChunkService").Error("清理分片：删除分片文件失败", "path", chunk.StoragePath, "error", err)
		}
	}

	// 删除分片目录
	chunkDir := filepath.Join(s.storage, uploadID)
	if err := os.RemoveAll(chunkDir); err != nil {
		logger.WithModule("ChunkService").Error("清理分片：删除分片目录失败", "path", chunkDir, "error", err)
	}

	// 删除分片记录
	if err := s.repo.DeleteChunksByUploadID(ctx, uploadID); err != nil {
		logger.WithModule("ChunkService").Error("清理分片：删除分片记录失败", "upload_id", uploadID, "error", err)
	}
}
