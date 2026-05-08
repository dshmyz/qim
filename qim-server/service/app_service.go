package service

import (
	"context"

	"qim-server/model"

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
	dbQuery := s.db.WithContext(ctx).Model(&model.App{}).Where("(user_id = ? OR is_global = ?) AND deleted_at IS NULL", query.UserID, true)

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
