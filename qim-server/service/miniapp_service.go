package service

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type MiniAppService struct {
	db *gorm.DB
}

func NewMiniAppService(db *gorm.DB) *MiniAppService {
	return &MiniAppService{db: db}
}

type MiniAppQuery struct {
	Page     int
	PageSize int
	Name     string
	Status   string
}

type MiniAppResult struct {
	List     []model.MiniApp
	Total    int64
	Page     int
	PageSize int
}

func (s *MiniAppService) GetMiniApps(query MiniAppQuery) (*MiniAppResult, error) {
	ctx := context.Background()
	dbQuery := s.db.WithContext(ctx).Model(&model.MiniApp{})

	if query.Name != "" {
		dbQuery = dbQuery.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Status != "" && query.Status != "all" {
		dbQuery = dbQuery.Where("status = ?", query.Status)
	}

	var total int64
	dbQuery.Count(&total)

	var miniApps []model.MiniApp
	offset := (query.Page - 1) * query.PageSize
	dbQuery.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&miniApps)

	return &MiniAppResult{
		List:     miniApps,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *MiniAppService) GetMiniAppByAppID(appID string) (*model.MiniApp, error) {
	ctx := context.Background()
	var miniApp model.MiniApp
	err := s.db.WithContext(ctx).Where("app_id = ?", appID).First(&miniApp).Error
	if err != nil {
		return nil, err
	}
	return &miniApp, nil
}

func (s *MiniAppService) GetMiniApp(id string) (*model.MiniApp, error) {
	ctx := context.Background()
	var miniApp model.MiniApp
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&miniApp).Error
	if err != nil {
		return nil, err
	}
	return &miniApp, nil
}

func (s *MiniAppService) IsAppIDExists(appID string) (bool, error) {
	ctx := context.Background()
	var count int64
	err := s.db.WithContext(ctx).Model(&model.MiniApp{}).Where("app_id = ?", appID).Count(&count).Error
	return count > 0, err
}

func (s *MiniAppService) CreateMiniApp(miniApp *model.MiniApp) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(miniApp).Error
}

func (s *MiniAppService) UpdateMiniApp(miniApp *model.MiniApp) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(miniApp).Error
}

func (s *MiniAppService) DeleteMiniApp(id string) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&model.MiniApp{}).Error
}
