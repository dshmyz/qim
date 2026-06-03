package repository

import (
	"context"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type fileRepository struct {
	*baseRepository[model.File]
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{
		baseRepository: &baseRepository[model.File]{db: db},
		db:             db,
	}
}

func (r *fileRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.File, error) {
	var files []*model.File
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

func (r *fileRepository) FindByFolderID(ctx context.Context, folderID *uint) ([]*model.File, error) {
	var files []*model.File
	query := r.db.WithContext(ctx)

	if folderID == nil {
		query = query.Where("folder_id IS NULL")
	} else {
		query = query.Where("folder_id = ?", *folderID)
	}

	err := query.Order("created_at DESC").Find(&files).Error
	return files, err
}

func (r *fileRepository) FindByChecksum(ctx context.Context, checksum string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("checksum = ?", checksum).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *fileRepository) UpdateStarred(ctx context.Context, id uint, starred bool) error {
	return r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id = ?", id).
		Update("is_starred", starred).Error
}

func (r *fileRepository) WithTx(tx *gorm.DB) BaseRepository[model.File] {
	return &fileRepository{
		baseRepository: &baseRepository[model.File]{db: tx},
		db:             tx,
	}
}
