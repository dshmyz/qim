package service

import (
	"log"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type AIProviderService struct {
	db *gorm.DB
}

func NewAIProviderService(db *gorm.DB) *AIProviderService {
	return &AIProviderService{db: db}
}

func (s *AIProviderService) GetProviders() ([]model.AIProvider, error) {
	var providers []model.AIProvider
	err := s.db.Order("created_at DESC").Find(&providers).Error
	return providers, err
}

func (s *AIProviderService) GetProviderByID(id uint) (*model.AIProvider, error) {
	var provider model.AIProvider
	err := s.db.First(&provider, id).Error
	return &provider, err
}

func (s *AIProviderService) CreateProvider(provider *model.AIProvider) error {
	return s.db.Create(provider).Error
}

func (s *AIProviderService) UpdateProvider(provider *model.AIProvider) error {
	return s.db.Save(provider).Error
}

func (s *AIProviderService) DeleteProvider(id uint) error {
	return s.db.Delete(&model.AIProvider{}, id).Error
}

// ReloadEnabledProviders 从数据库加载已启用的 AI Provider 并 reload 到 AIService。
// 返回加载的 provider 数量。查询失败时返回 error，调用方可据此决定是否告警。
func (s *AIProviderService) ReloadEnabledProviders(aiService *ai.AIService) (int, error) {
	var dbProviders []model.AIProvider
	if err := s.db.Where("enabled = ?", true).Order("priority ASC").Find(&dbProviders).Error; err != nil {
		return 0, err
	}

	infos := make([]ai.DBProviderInfo, 0, len(dbProviders))
	for _, p := range dbProviders {
		infos = append(infos, ai.DBProviderInfo{
			ID:       p.ID,
			Name:     p.Name,
			APIType:  p.APIType,
			Endpoint: p.Endpoint,
			APIKey:   p.APIKey,
			Models:   []string(p.Models),
			Enabled:  p.Enabled,
			Priority: p.Priority,
		})
	}

	aiService.ReloadProvidersFromDB(infos)
	log.Printf("[AIProviderService] Reloaded %d providers from DB", len(infos))
	return len(infos), nil
}
