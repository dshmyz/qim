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

func (r *chunkRepository) DeleteChunksByUploadID(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).
		Where("upload_id = ?", uploadID).
		Delete(&model.FileChunk{}).Error
}

// File 相关操作

func (r *chunkRepository) GetFileByHash(ctx context.Context, fileHash string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("checksum = ?", fileHash).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}
