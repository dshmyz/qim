package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"gorm.io/gorm"
)

var (
	// ErrFileSpaceForbidden is returned when the actor cannot use the requested space.
	ErrFileSpaceForbidden = errors.New("file space access forbidden")
	// ErrFileSpaceInvalid is returned for malformed spaces and invalid folder relationships.
	ErrFileSpaceInvalid = errors.New("invalid file space request")
)

// FileSpace identifies the owner of a file tree. Group IDs refer to model.Group IDs,
// not conversation IDs.
type FileSpace struct {
	Type string
	ID   uint
}

type FileSpaceAction string

const (
	FileSpaceActionList     FileSpaceAction = "list"
	FileSpaceActionUpload   FileSpaceAction = "upload"
	FileSpaceActionDownload FileSpaceAction = "download"
	FileSpaceActionManage   FileSpaceAction = "manage"
)

// FileSpaceQuery restricts a listing to a folder and optionally filters its files.
// Files and folders are always selected from the same scoped tree.
type FileSpaceQuery struct {
	FolderID *uint
	Search   string
	Page     int
	PageSize int
}

type FileSpaceList struct {
	Files   []model.File
	Folders []model.Folder
	Total   int64
}

// FileSpaceService is the sole service boundary for reusable, scoped file trees.
// It keeps authorization, scope predicates, and shared-storage lifetime decisions together.
type FileSpaceService struct {
	db    *gorm.DB
	store StorageAccessor
}

func NewFileSpaceService(db *gorm.DB) *FileSpaceService {
	return &FileSpaceService{db: db}
}

func (s *FileSpaceService) SetStorageAccessor(store StorageAccessor) {
	s.store = store
}

func (s *FileSpaceService) authorize(ctx context.Context, actorID uint, space FileSpace, action FileSpaceAction) error {
	if actorID == 0 || space.ID == 0 {
		return ErrFileSpaceInvalid
	}

	switch space.Type {
	case "user":
		if actorID != space.ID {
			return ErrFileSpaceForbidden
		}
		return nil
	case "group":
		var group model.Group
		if err := s.db.WithContext(ctx).Select("id", "conversation_id").First(&group, space.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFileSpaceForbidden
			}
			return err
		}

		var member model.ConversationMember
		if err := s.db.WithContext(ctx).
			Where("conversation_id = ? AND user_id = ?", group.ConversationID, actorID).
			First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFileSpaceForbidden
			}
			return err
		}
		if action == FileSpaceActionManage && member.Role != "owner" && member.Role != "admin" {
			return ErrFileSpaceForbidden
		}
		return nil
	default:
		return ErrFileSpaceInvalid
	}
}

func (s *FileSpaceService) List(ctx context.Context, actorID uint, space FileSpace, query FileSpaceQuery) (*FileSpaceList, error) {
	if err := s.authorize(ctx, actorID, space, FileSpaceActionList); err != nil {
		return nil, err
	}
	if err := s.validateFolder(ctx, space, query.FolderID); err != nil {
		return nil, err
	}

	result := &FileSpaceList{}
	folderQuery := s.db.WithContext(ctx).
		Where("scope_type = ? AND scope_id = ?", space.Type, space.ID)
	if query.FolderID == nil {
		folderQuery = folderQuery.Where("parent_id IS NULL")
	} else {
		folderQuery = folderQuery.Where("parent_id = ?", *query.FolderID)
	}
	if err := folderQuery.Order("sort_order ASC, created_at ASC").Find(&result.Folders).Error; err != nil {
		return nil, err
	}

	fileQuery := s.db.WithContext(ctx).Model(&model.File{}).
		Where("scope_type = ? AND scope_id = ?", space.Type, space.ID)
	if query.FolderID == nil {
		fileQuery = fileQuery.Where("folder_id IS NULL")
	} else {
		fileQuery = fileQuery.Where("folder_id = ?", *query.FolderID)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		fileQuery = fileQuery.Where("name LIKE ? OR original_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := fileQuery.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	if err := fileQuery.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&result.Files).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (s *FileSpaceService) CreateFolder(ctx context.Context, actorID uint, space FileSpace, name string, parentID *uint) (*model.Folder, error) {
	if err := s.authorize(ctx, actorID, space, FileSpaceActionManage); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrFileSpaceInvalid
	}
	if err := s.validateFolder(ctx, space, parentID); err != nil {
		return nil, err
	}

	folder := &model.Folder{UserID: actorID, ScopeType: space.Type, ScopeID: space.ID, Name: name, ParentID: parentID}
	if err := s.db.WithContext(ctx).Create(folder).Error; err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *FileSpaceService) Move(ctx context.Context, actorID uint, space FileSpace, fileIDs []uint, folderID *uint) error {
	if err := s.authorize(ctx, actorID, space, FileSpaceActionManage); err != nil {
		return err
	}
	if len(fileIDs) == 0 {
		return ErrFileSpaceInvalid
	}
	if err := s.validateFolder(ctx, space, folderID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.File{}).
			Where("id IN ? AND scope_type = ? AND scope_id = ?", fileIDs, space.Type, space.ID).
			Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(fileIDs)) {
			return ErrFileSpaceForbidden
		}
		return tx.Model(&model.File{}).Where("id IN ?", fileIDs).Update("folder_id", folderID).Error
	})
}

func (s *FileSpaceService) Delete(ctx context.Context, actorID uint, space FileSpace, fileID uint) error {
	if err := s.authorize(ctx, actorID, space, FileSpaceActionManage); err != nil {
		return err
	}
	var file model.File
	if err := s.db.WithContext(ctx).
		Where("id = ? AND scope_type = ? AND scope_id = ?", fileID, space.Type, space.ID).
		First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileSpaceForbidden
		}
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&file).Error; err != nil {
		return err
	}
	if err := deleteStoragePathIfUnreferenced(ctx, s.db, s.store, file); err != nil {
		logger.WithModule("FileSpaceService").Warn("删除物理文件失败", "file_id", file.ID, "error", err)
	}
	return nil
}

func (s *FileSpaceService) ShareReference(ctx context.Context, actorID uint, space FileSpace, sourceFileID uint, folderID *uint) (*model.File, error) {
	if err := s.authorize(ctx, actorID, space, FileSpaceActionManage); err != nil {
		return nil, err
	}
	if err := s.validateFolder(ctx, space, folderID); err != nil {
		return nil, err
	}

	var source model.File
	if err := s.db.WithContext(ctx).First(&source, sourceFileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileSpaceForbidden
		}
		return nil, err
	}
	if err := s.authorize(ctx, actorID, FileSpace{Type: source.ScopeType, ID: source.ScopeID}, FileSpaceActionDownload); err != nil {
		return nil, err
	}

	shared := &model.File{
		UserID:       actorID,
		ScopeType:    space.Type,
		ScopeID:      space.ID,
		Name:         source.Name,
		OriginalName: source.OriginalName,
		Size:         source.Size,
		MimeType:     source.MimeType,
		StoragePath:  source.StoragePath,
		Checksum:     source.Checksum,
		FolderID:     folderID,
		Source:       "shared_reference",
		SourceID:     strconv.FormatUint(uint64(source.ID), 10),
		Tags:         source.Tags,
	}
	if err := s.db.WithContext(ctx).Create(shared).Error; err != nil {
		return nil, err
	}
	return shared, nil
}

// AttachUpload moves an actor's freshly uploaded personal file into a group
// space. Unlike ShareReference, it keeps the same file record and storage
// object so the upload has a single owner after attachment.
func (s *FileSpaceService) AttachUpload(ctx context.Context, actorID uint, space FileSpace, fileID uint, folderID *uint) (*model.File, error) {
	if fileID == 0 {
		return nil, ErrFileSpaceInvalid
	}
	if err := s.authorize(ctx, actorID, space, FileSpaceActionUpload); err != nil {
		return nil, err
	}
	if err := s.validateFolder(ctx, space, folderID); err != nil {
		return nil, err
	}

	file := &model.File{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ? AND scope_type = ? AND scope_id = ?", fileID, actorID, "user", actorID).First(file).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFileSpaceForbidden
			}
			return err
		}

		if err := tx.Model(file).Updates(map[string]interface{}{
			"scope_type": space.Type,
			"scope_id":   space.ID,
			"folder_id":  folderID,
		}).Error; err != nil {
			return err
		}
		file.ScopeType = space.Type
		file.ScopeID = space.ID
		file.FolderID = folderID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return file, nil
}

// OpenDownload returns a group-scoped file only after the requesting user has
// been authorized as a current member of that group.
func (s *FileSpaceService) OpenDownload(ctx context.Context, actorID uint, space FileSpace, fileID uint) (*model.File, error) {
	if fileID == 0 {
		return nil, ErrFileSpaceInvalid
	}
	if err := s.authorize(ctx, actorID, space, FileSpaceActionDownload); err != nil {
		return nil, err
	}

	file := &model.File{}
	if err := s.db.WithContext(ctx).Where("id = ? AND scope_type = ? AND scope_id = ?", fileID, space.Type, space.ID).First(file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileSpaceForbidden
		}
		return nil, err
	}
	return file, nil
}

func (s *FileSpaceService) validateFolder(ctx context.Context, space FileSpace, folderID *uint) error {
	if folderID == nil {
		return nil
	}
	var folder model.Folder
	if err := s.db.WithContext(ctx).First(&folder, *folderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileSpaceForbidden
		}
		return err
	}
	if folder.ScopeType != space.Type || folder.ScopeID != space.ID {
		return ErrFileSpaceForbidden
	}

	visited := make(map[uint]struct{})
	for {
		if _, seen := visited[folder.ID]; seen {
			return ErrFileSpaceInvalid
		}
		visited[folder.ID] = struct{}{}
		if folder.ParentID == nil {
			return nil
		}
		var parent model.Folder
		if err := s.db.WithContext(ctx).First(&parent, *folder.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFileSpaceInvalid
			}
			return err
		}
		folder = parent
		if folder.ScopeType != space.Type || folder.ScopeID != space.ID {
			return ErrFileSpaceForbidden
		}
	}
}
