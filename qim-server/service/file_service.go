package service

import (
	"context"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/repository"

	"gorm.io/gorm"
)

type FileService struct {
	repo repository.FileRepository
	db   *gorm.DB
}

func NewFileService(db *gorm.DB) *FileService {
	return &FileService{
		repo: repository.NewFileRepository(db),
		db:   db,
	}
}

func (s *FileService) CreateFile(file *model.File) error {
	ctx := context.Background()
	return s.repo.Create(ctx, file)
}

func (s *FileService) GetFile(userID, fileID uint) (*model.File, error) {
	ctx := context.Background()
	var file model.File
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", fileID, userID).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *FileService) GetFiles(userID uint, page, pageSize int, filters map[string]string) ([]model.File, int64, error) {
	ctx := context.Background()
	query := s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ?", userID)

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
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", fileID, userID).First(&file).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", fileID, userID).First(&file).Error; err != nil {
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
	result := s.db.WithContext(ctx).Where("id IN ? AND user_id = ?", fileIDs, userID).Delete(&model.File{})
	return result.RowsAffected, result.Error
}

func (s *FileService) BatchMove(userID uint, fileIDs []uint, targetFolderID uint) (int64, error) {
	ctx := context.Background()
	result := s.db.WithContext(ctx).Model(&model.File{}).
		Where("id IN ? AND user_id = ?", fileIDs, userID).
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
	result := s.db.WithContext(ctx).Model(&model.File{}).
		Where("id IN ? AND user_id = ?", fileIDs, userID).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (s *FileService) GetStarredFiles(userID uint, page, pageSize int) ([]model.File, int64, error) {
	ctx := context.Background()
	var total int64
	s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ? AND is_starred = ?", userID, true).Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	s.db.WithContext(ctx).Where("user_id = ? AND is_starred = ?", userID, true).
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

	s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ?", userID).Count(&stats.TotalFiles)
	s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ? AND is_starred = ?", userID, true).Count(&stats.StarredFiles)
	s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ?", userID).Select("COALESCE(SUM(size), 0)").Scan(&stats.TotalSize)
	s.db.WithContext(ctx).Model(&model.Folder{}).Where("user_id = ?", userID).Count(&stats.FolderCount)

	s.db.WithContext(ctx).Model(&model.File{}).
		Where("user_id = ?", userID).
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
	query := s.db.WithContext(ctx).Where("user_id = ?", userID)

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
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", folderID, userID).First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func (s *FileService) UpdateFolder(userID, folderID uint, updates map[string]interface{}) (*model.Folder, error) {
	ctx := context.Background()
	var folder model.Folder
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", folderID, userID).First(&folder).Error; err != nil {
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
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", folderID, userID).Delete(&model.Folder{}).Error
}

func (s *FileService) GetFolderChildCount(userID, folderID uint) (int64, error) {
	ctx := context.Background()
	var count int64
	err := s.db.WithContext(ctx).Model(&model.Folder{}).Where("user_id = ? AND parent_id = ?", userID, folderID).Count(&count).Error
	return count, err
}

func (s *FileService) GetFolderFileCount(userID, folderID uint) (int64, error) {
	ctx := context.Background()
	var count int64
	err := s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ? AND folder_id = ?", userID, folderID).Count(&count).Error
	return count, err
}

func (s *FileService) DeleteFolderRecursive(userID, folderID uint) error {
	ctx := context.Background()

	var children []model.Folder
	s.db.WithContext(ctx).Where("user_id = ? AND parent_id = ?", userID, folderID).Find(&children)

	for _, child := range children {
		s.db.WithContext(ctx).Where("user_id = ? AND folder_id = ?", userID, child.ID).Delete(&model.File{})
		s.DeleteFolderRecursive(userID, child.ID)
		s.db.WithContext(ctx).Delete(&child)
	}

	return nil
}

func (s *FileService) DeleteFolderFiles(userID, folderID uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Where("user_id = ? AND folder_id = ?", userID, folderID).Delete(&model.File{}).Error
}

func (s *FileService) GetFolderFiles(userID, folderID uint, page, pageSize int) ([]model.File, int64, error) {
	ctx := context.Background()
	var total int64
	s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ? AND folder_id = ?", userID, folderID).Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	s.db.WithContext(ctx).Where("user_id = ? AND folder_id = ?", userID, folderID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files)

	return files, total, nil
}

func (s *FileService) CreateFolder(folder *model.Folder) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(folder).Error
}

func (s *FileService) DeleteFile(userID, fileID uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", fileID, userID).Delete(&model.File{}).Error
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
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", currentID, userID).First(&folder).Error; err != nil {
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
	return s.db.WithContext(ctx).Model(&model.File{}).
		Where("id = ? AND user_id = ?", fileID, userID).
		Update("source", source).Error
}
