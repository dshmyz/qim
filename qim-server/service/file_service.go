package service

import (
	"context"
	"errors"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/repository"

	"gorm.io/gorm"
)

type FileService struct {
	repo  repository.FileRepository
	db    *gorm.DB
	store StorageAccessor
}

func NewFileService(db *gorm.DB) *FileService {
	return &FileService{
		repo: repository.NewFileRepository(db),
		db:   db,
	}
}

// SetStorageAccessor 注入存储访问器，用于批量删除时清理物理文件。
func (s *FileService) SetStorageAccessor(store StorageAccessor) {
	s.store = store
}

// legacyUserFiles limits the legacy personal-file API to records that belong
// to the user's own file space. Group-scoped records may share UserID with the
// uploading actor, but must remain reachable only through FileSpaceService.
func (s *FileService) legacyUserFiles(ctx context.Context, userID uint) *gorm.DB {
	return s.db.WithContext(ctx).Where("user_id = ? AND scope_type = ? AND scope_id = ?", userID, "user", userID)
}

func (s *FileService) legacyUserFolders(ctx context.Context, userID uint) *gorm.DB {
	return s.db.WithContext(ctx).Where("user_id = ? AND scope_type = ? AND scope_id = ?", userID, "user", userID)
}

func (s *FileService) CreateFile(file *model.File) error {
	ctx := context.Background()
	file.ScopeType = "user"
	file.ScopeID = file.UserID
	return withStoragePathLock(file.StoragePath, func() error {
		return s.repo.Create(ctx, file)
	})
}

func (s *FileService) GetFile(userID, fileID uint) (*model.File, error) {
	ctx := context.Background()
	var file model.File
	err := s.legacyUserFiles(ctx, userID).Where("id = ?", fileID).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *FileService) GetFiles(userID uint, page, pageSize int, filters map[string]string) ([]model.File, int64, error) {
	ctx := context.Background()
	query := s.legacyUserFiles(ctx, userID).Model(&model.File{})

	if folderIDStr, ok := filters["folder_id"]; ok && folderIDStr != "" {
		query = query.Where("folder_id = ?", folderIDStr)
	}
	if source, ok := filters["source"]; ok && source != "" {
		query = query.Where("source = ?", source)
	}
	if starred, ok := filters["starred"]; ok {
		if starred == "true" {
			query = query.Where("is_starred = ?", true)
		} else if starred == "false" {
			query = query.Where("is_starred = ?", false)
		}
	}
	if fileType, ok := filters["type"]; ok && fileType != "" {
		switch fileType {
		case "image":
			query = query.Where("mime_type LIKE ?", "image/%")
		case "video":
			query = query.Where("mime_type LIKE ?", "video/%")
		case "audio":
			query = query.Where("mime_type LIKE ?", "audio/%")
		case "document":
			query = query.Where("mime_type LIKE ? OR mime_type LIKE ? OR mime_type LIKE ? OR mime_type LIKE ?",
				"application/pdf", "application/msword", "application/vnd.ms-excel", "text/%")
		default:
			query = query.Where("mime_type LIKE ?", fileType+"/%")
		}
	}
	if search, ok := filters["search"]; ok && search != "" {
		query = query.Where("name LIKE ? OR original_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if dateFrom, ok := filters["date_from"]; ok && dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if dateTo, ok := filters["date_to"]; ok && dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			query = query.Where("created_at <= ?", t.Add(24*time.Hour-1*time.Second))
		}
	}

	sortBy := filters["sort_by"]
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := filters["sort_order"]
	if sortOrder == "" {
		sortOrder = "desc"
	}
	allowedSortFields := map[string]bool{"created_at": true, "name": true, "size": true, "updated_at": true}
	if !allowedSortFields[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	var total int64
	query.Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	query.Order(sortBy + " " + sortOrder).Offset(offset).Limit(pageSize).Find(&files)

	return files, total, nil
}

func (s *FileService) UpdateFile(userID, fileID uint, updates map[string]interface{}) (*model.File, error) {
	ctx := context.Background()
	var file model.File
	if err := s.legacyUserFiles(ctx, userID).Where("id = ?", fileID).First(&file).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&file).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.WithContext(ctx).First(&file, file.ID)
	return &file, nil
}

func (s *FileService) ToggleStar(userID, fileID uint) (*model.File, error) {
	ctx := context.Background()
	var file model.File
	if err := s.legacyUserFiles(ctx, userID).Where("id = ?", fileID).First(&file).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	if file.IsStarred {
		file.IsStarred = false
		file.StarredAt = nil
	} else {
		file.IsStarred = true
		file.StarredAt = &now
	}

	if err := s.db.WithContext(ctx).Save(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *FileService) BatchDelete(userID uint, fileIDs []uint) (int64, error) {
	ctx := context.Background()
	var files []model.File
	if err := s.legacyUserFiles(ctx, userID).Where("id IN ?", fileIDs).Find(&files).Error; err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, nil
	}
	var deleted int64
	for _, f := range files {
		removed, err := s.deleteLegacyFile(ctx, userID, f.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return deleted, err
		}
		if removed {
			deleted++
		}
	}
	return deleted, nil
}

// deleteStoragePathIfUnreferenced performs storage cleanup only while the
// caller holds the path lock and has already established that its deletion left
// no active record using the path.
func deleteStoragePathIfUnreferenced(ctx context.Context, db *gorm.DB, store StorageAccessor, file model.File) error {
	if store == nil || file.StoragePath == "" {
		return nil
	}

	var remaining int64
	if err := db.WithContext(ctx).Model(&model.File{}).Where("storage_path = ?", file.StoragePath).Count(&remaining).Error; err != nil {
		return err
	}
	if remaining != 0 {
		return nil
	}
	return store.DeleteByPath(ctx, file.StoragePath)
}

func (s *FileService) deleteLegacyFile(ctx context.Context, userID, fileID uint) (bool, error) {
	var candidate model.File
	if err := s.legacyUserFiles(ctx, userID).Where("id = ?", fileID).First(&candidate).Error; err != nil {
		return false, err
	}

	removed := false
	err := withStoragePathLock(candidate.StoragePath, func() error {
		var deleted model.File
		var deleteStorage bool
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("id = ? AND user_id = ? AND scope_type = ? AND scope_id = ?", fileID, userID, "user", userID).First(&deleted).Error; err != nil {
				return err
			}
			if err := tx.Delete(&deleted).Error; err != nil {
				return err
			}
			removed = true

			var remaining int64
			if err := tx.Model(&model.File{}).Where("storage_path = ?", deleted.StoragePath).Count(&remaining).Error; err != nil {
				return err
			}
			deleteStorage = remaining == 0
			return nil
		}); err != nil {
			return err
		}
		if deleteStorage {
			if err := deleteStoragePathIfUnreferenced(ctx, s.db, s.store, deleted); err != nil {
				logger.WithModule("FileService").Warn("删除物理文件失败", "file_id", deleted.ID, "error", err)
			}
		}
		return nil
	})
	return removed, err
}

func (s *FileService) BatchMove(userID uint, fileIDs []uint, targetFolderID uint) (int64, error) {
	ctx := context.Background()
	result := s.legacyUserFiles(ctx, userID).Model(&model.File{}).
		Where("id IN ?", fileIDs).
		Update("folder_id", targetFolderID)
	return result.RowsAffected, result.Error
}

func (s *FileService) BatchStar(userID uint, fileIDs []uint, starred bool) (int64, error) {
	ctx := context.Background()
	updates := map[string]interface{}{"is_starred": starred}
	if starred {
		updates["starred_at"] = time.Now()
	} else {
		updates["starred_at"] = nil
	}
	result := s.legacyUserFiles(ctx, userID).Model(&model.File{}).
		Where("id IN ?", fileIDs).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (s *FileService) GetStarredFiles(userID uint, page, pageSize int) ([]model.File, int64, error) {
	ctx := context.Background()
	var total int64
	s.legacyUserFiles(ctx, userID).Model(&model.File{}).Where("is_starred = ?", true).Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	s.legacyUserFiles(ctx, userID).Where("is_starred = ?", true).
		Order("starred_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files)

	return files, total, nil
}

type FileStats struct {
	TotalFiles   int64
	StarredFiles int64
	TotalSize    int64
	FolderCount  int64
	TypeStats    []FileTypeStat
}

type FileTypeStat struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
	Size  int64  `json:"size"`
}

func (s *FileService) GetFileStats(userID uint) (*FileStats, error) {
	ctx := context.Background()
	stats := &FileStats{}

	s.legacyUserFiles(ctx, userID).Model(&model.File{}).Count(&stats.TotalFiles)
	s.legacyUserFiles(ctx, userID).Model(&model.File{}).Where("is_starred = ?", true).Count(&stats.StarredFiles)
	s.legacyUserFiles(ctx, userID).Model(&model.File{}).Select("COALESCE(SUM(size), 0)").Scan(&stats.TotalSize)
	s.legacyUserFolders(ctx, userID).Model(&model.Folder{}).Count(&stats.FolderCount)

	s.legacyUserFiles(ctx, userID).Model(&model.File{}).
		Select(`
			CASE
				WHEN mime_type LIKE 'image/%' THEN 'image'
				WHEN mime_type LIKE 'video/%' THEN 'video'
				WHEN mime_type LIKE 'audio/%' THEN 'audio'
				WHEN mime_type LIKE 'application/%' OR mime_type LIKE 'text/%' THEN 'document'
				ELSE 'other'
			END as type,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size
		`).
		Group(`
			CASE
				WHEN mime_type LIKE 'image/%' THEN 'image'
				WHEN mime_type LIKE 'video/%' THEN 'video'
				WHEN mime_type LIKE 'audio/%' THEN 'audio'
				WHEN mime_type LIKE 'application/%' OR mime_type LIKE 'text/%' THEN 'document'
				ELSE 'other'
			END
		`).
		Scan(&stats.TypeStats)

	return stats, nil
}

func (s *FileService) GetFolderTree(userID uint, parentID *uint) ([]model.Folder, error) {
	ctx := context.Background()
	var folders []model.Folder
	query := s.legacyUserFolders(ctx, userID)

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	err := query.Order("sort_order ASC, created_at ASC").Find(&folders).Error
	return folders, err
}

func (s *FileService) GetFolder(userID, folderID uint) (*model.Folder, error) {
	ctx := context.Background()
	var folder model.Folder
	err := s.legacyUserFolders(ctx, userID).Where("id = ?", folderID).First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func (s *FileService) UpdateFolder(userID, folderID uint, updates map[string]interface{}) (*model.Folder, error) {
	ctx := context.Background()
	var folder model.Folder
	if err := s.legacyUserFolders(ctx, userID).Where("id = ?", folderID).First(&folder).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&folder).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.WithContext(ctx).First(&folder, folder.ID)
	return &folder, nil
}

func (s *FileService) DeleteFolder(userID, folderID uint) error {
	ctx := context.Background()
	return s.legacyUserFolders(ctx, userID).Where("id = ?", folderID).Delete(&model.Folder{}).Error
}

func (s *FileService) GetFolderChildCount(userID, folderID uint) (int64, error) {
	ctx := context.Background()
	var count int64
	err := s.legacyUserFolders(ctx, userID).Model(&model.Folder{}).Where("parent_id = ?", folderID).Count(&count).Error
	return count, err
}

func (s *FileService) GetFolderFileCount(userID, folderID uint) (int64, error) {
	ctx := context.Background()
	var count int64
	err := s.legacyUserFiles(ctx, userID).Model(&model.File{}).Where("folder_id = ?", folderID).Count(&count).Error
	return count, err
}

func (s *FileService) DeleteFolderRecursive(userID, folderID uint) error {
	ctx := context.Background()

	var children []model.Folder
	s.legacyUserFolders(ctx, userID).Where("parent_id = ?", folderID).Find(&children)

	for _, child := range children {
		s.legacyUserFiles(ctx, userID).Where("folder_id = ?", child.ID).Delete(&model.File{})
		s.DeleteFolderRecursive(userID, child.ID)
		s.db.WithContext(ctx).Delete(&child)
	}

	return nil
}

func (s *FileService) DeleteFolderFiles(userID, folderID uint) error {
	ctx := context.Background()
	return s.legacyUserFiles(ctx, userID).Where("folder_id = ?", folderID).Delete(&model.File{}).Error
}

func (s *FileService) GetFolderFiles(userID, folderID uint, page, pageSize int) ([]model.File, int64, error) {
	ctx := context.Background()
	var total int64
	s.legacyUserFiles(ctx, userID).Model(&model.File{}).Where("folder_id = ?", folderID).Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	s.legacyUserFiles(ctx, userID).Where("folder_id = ?", folderID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files)

	return files, total, nil
}

func (s *FileService) CreateFolder(folder *model.Folder) error {
	ctx := context.Background()
	folder.ScopeType = "user"
	folder.ScopeID = folder.UserID
	return s.db.WithContext(ctx).Create(folder).Error
}

func (s *FileService) DeleteFile(userID, fileID uint) error {
	ctx := context.Background()
	_, err := s.deleteLegacyFile(ctx, userID, fileID)
	return err
}

func (s *FileService) IsDescendant(userID, targetID, ancestorID uint) bool {
	ctx := context.Background()
	currentID := targetID
	visited := make(map[uint]bool)

	for currentID != 0 {
		if visited[currentID] {
			return false
		}
		visited[currentID] = true

		if currentID == ancestorID {
			return true
		}

		var folder model.Folder
		if err := s.legacyUserFolders(ctx, userID).Where("id = ?", currentID).First(&folder).Error; err != nil {
			return false
		}

		if folder.ParentID == nil {
			return false
		}
		currentID = *folder.ParentID
	}

	return false
}

func (s *FileService) UpdateFileSource(fileID, userID uint, source string) error {
	ctx := context.Background()
	return s.legacyUserFiles(ctx, userID).Model(&model.File{}).
		Where("id = ?", fileID).
		Update("source", source).Error
}
