package repository

import (
	"context"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type chunkRepository struct {
	db *gorm.DB
}

func NewChunkRepository(db *gorm.DB) ChunkRepository {
	return &chunkRepository{db: db}
}

// UploadTask 相关操作

func (r *chunkRepository) CreateUploadTask(ctx context.Context, task *model.UploadTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *chunkRepository) GetUploadTask(ctx context.Context, uploadID string) (*model.UploadTask, error) {
	var task model.UploadTask
	err := r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *chunkRepository) UpdateUploadTask(ctx context.Context, task *model.UploadTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// AtomicIncrementUploadedChunks 原子自增已上传分片数
// 使用 SQL 的 uploaded_chunks = uploaded_chunks + 1 避免并发 read-modify-write 竞态
func (r *chunkRepository) AtomicIncrementUploadedChunks(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).
		Model(&model.UploadTask{}).
		Where("upload_id = ?", uploadID).
		UpdateColumn("uploaded_chunks", gorm.Expr("uploaded_chunks + 1")).Error
}

// MarkTaskUploading 将任务从 pending 标记为 uploading
func (r *chunkRepository) MarkTaskUploading(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).
		Model(&model.UploadTask{}).
		Where("upload_id = ? AND status = ?", uploadID, "pending").
		Update("status", "uploading").Error
}

// MarkTaskCancelled 抢占式将任务标记为 cancelled
// 仅在当前状态为 pending/uploading 时成功，防止与 UploadChunk 并发冲突
func (r *chunkRepository) MarkTaskCancelled(ctx context.Context, uploadID string) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&model.UploadTask{}).
		Where("upload_id = ? AND status IN ?", uploadID, []string{"pending", "uploading"}).
		Update("status", "cancelled")
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// MarkTaskCompleted 抢占式将任务标记为 completed
func (r *chunkRepository) MarkTaskCompleted(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).
		Model(&model.UploadTask{}).
		Where("upload_id = ?", uploadID).
		Update("status", "completed").Error
}

func (r *chunkRepository) DeleteUploadTask(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Delete(&model.UploadTask{}).Error
}

// FileChunk 相关操作

func (r *chunkRepository) CreateChunk(ctx context.Context, chunk *model.FileChunk) error {
	return r.db.WithContext(ctx).Create(chunk).Error
}

func (r *chunkRepository) GetChunk(ctx context.Context, uploadID string, chunkIndex int) (*model.FileChunk, error) {
	var chunk model.FileChunk
	err := r.db.WithContext(ctx).
		Where("upload_id = ? AND chunk_index = ?", uploadID, chunkIndex).
		First(&chunk).Error
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

func (r *chunkRepository) GetChunksByUploadID(ctx context.Context, uploadID string) ([]model.FileChunk, error) {
	var chunks []model.FileChunk
	err := r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Order("chunk_index ASC").
		Find(&chunks).Error
	return chunks, err
}

func (r *chunkRepository) GetUploadedChunkIndexes(ctx context.Context, uploadID string) ([]int, error) {
	var indexes []int
	err := r.db.WithContext(ctx).
		Model(&model.FileChunk{}).
		Where("upload_id = ? AND status = ?", uploadID, "uploaded").
		Pluck("chunk_index", &indexes).Error
	return indexes, err
}

func (r *chunkRepository) UpdateChunkStatus(ctx context.Context, uploadID string, chunkIndex int, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.FileChunk{}).
		Where("upload_id = ? AND chunk_index = ?", uploadID, chunkIndex).
		Update("status", status).Error
}

// ConditionalUpdateChunkStatus 条件更新分片状态
// 仅当当前状态等于 expectFrom 时才更新为 newStatus，返回是否实际更新了
// 用于幂等控制：防止并发上传同一分片时重复计数
func (r *chunkRepository) ConditionalUpdateChunkStatus(ctx context.Context, uploadID string, chunkIndex int, expectFrom, newStatus string) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&model.FileChunk{}).
		Where("upload_id = ? AND chunk_index = ? AND status = ?", uploadID, chunkIndex, expectFrom).
		Update("status", newStatus)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *chunkRepository) DeleteChunksByUploadID(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Delete(&model.FileChunk{}).Error
}
