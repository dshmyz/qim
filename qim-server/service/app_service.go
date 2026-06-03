package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type AppService struct {
	db *gorm.DB
}

func NewAppService(db *gorm.DB) *AppService {
	return &AppService{db: db}
}

type AppQuery struct {
	UserID   uint
	Page     int
	PageSize int
	Name     string
	Status   string
	Category string
	IsGlobal string
}

type AppResult struct {
	List     []model.App
	Total    int64
	Page     int
	PageSize int
}

func (s *AppService) GetUserApps(query AppQuery) (*AppResult, error) {
	ctx := context.Background()
	dbQuery := s.db.WithContext(ctx).Model(&model.App{}).Where("user_id = ? AND is_global = ? AND deleted_at IS NULL", query.UserID, false)

	if query.Name != "" {
		dbQuery = dbQuery.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Category != "" {
		dbQuery = dbQuery.Where("category = ?", query.Category)
	}
	if query.Status != "" && query.Status != "all" {
		dbQuery = dbQuery.Where("status = ?", query.Status)
	}

	var total int64
	dbQuery.Count(&total)

	var apps []model.App
	offset := (query.Page - 1) * query.PageSize
	dbQuery.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&apps)

	return &AppResult{
		List:     apps,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *AppService) GetAllApps(query AppQuery) (*AppResult, error) {
	ctx := context.Background()
	dbQuery := s.db.WithContext(ctx).Model(&model.App{})

	if query.Name != "" {
		dbQuery = dbQuery.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.IsGlobal != "" {
		if query.IsGlobal == "true" {
			dbQuery = dbQuery.Where("is_global = ?", true)
		} else {
			dbQuery = dbQuery.Where("is_global = ?", false)
		}
	}
	if query.Status != "" && query.Status != "all" {
		dbQuery = dbQuery.Where("status = ?", query.Status)
	}

	var total int64
	dbQuery.Count(&total)

	var apps []model.App
	offset := (query.Page - 1) * query.PageSize
	dbQuery.Order("is_global DESC, created_at DESC").Offset(offset).Limit(query.PageSize).Find(&apps)

	return &AppResult{
		List:     apps,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *AppService) GetApp(appID uint) (*model.App, error) {
	ctx := context.Background()
	var app model.App
	err := s.db.WithContext(ctx).First(&app, appID).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *AppService) GetUserApp(appID, userID uint) (*model.App, error) {
	ctx := context.Background()
	var app model.App
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", appID, userID).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *AppService) GetGlobalApp(appID uint) (*model.App, error) {
	ctx := context.Background()
	var app model.App
	err := s.db.WithContext(ctx).Where("id = ? AND is_global = ?", appID, true).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetBuiltInApps 获取内置应用列表（is_global = true 且 status = 'active'）
func (s *AppService) GetBuiltInApps() ([]model.App, error) {
	ctx := context.Background()
	var apps []model.App
	err := s.db.WithContext(ctx).
		Where("is_global = ? AND status = ? AND deleted_at IS NULL", true, "active").
		Order("created_at ASC").
		Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}

// GetBuiltInAppsForUser 获取用户可见的内置应用列表
func (s *AppService) GetBuiltInAppsForUser(userID uint) ([]model.App, error) {
	ctx := context.Background()

	var apps []model.App
	err := s.db.WithContext(ctx).
		Where("is_global = ? AND status = ? AND deleted_at IS NULL", true, "active").
		Order("created_at ASC").
		Find(&apps).Error
	if err != nil {
		return nil, err
	}

	// 根据权限范围过滤应用
	result := make([]model.App, 0)
	for _, app := range apps {
		if s.isAppVisibleToUser(app, userID) {
			result = append(result, app)
		}
	}

	return result, nil
}

// isAppVisibleToUser 检查应用是否对用户可见
func (s *AppService) isAppVisibleToUser(app model.App, userID uint) bool {
	switch app.ScopeType {
	case "all":
		return true
	case "users":
		if app.ScopeValue == "" {
			return true
		}
		// 检查用户ID是否在允许列表中
		userIDs := strings.Split(app.ScopeValue, ",")
		for _, id := range userIDs {
			if strings.TrimSpace(id) == fmt.Sprintf("%d", userID) {
				return true
			}
		}
		return false
	case "organizations":
		if app.ScopeValue == "" {
			return true
		}
		// 组织范围暂时不实现，留待后续扩展
		return false
	case "roles":
		if app.ScopeValue == "" {
			return true
		}
		// 角色范围暂时不实现，留待后续扩展
		return true
	default:
		return true
	}
}

func (s *AppService) CreateApp(app *model.App) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(app).Error
}

func (s *AppService) UpdateApp(app *model.App) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(app).Error
}

func (s *AppService) DeleteApp(appID, userID uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", appID, userID).Delete(&model.App{}).Error
}
