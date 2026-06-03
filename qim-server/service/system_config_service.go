package service

import (
	"fmt"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type SystemConfigService struct {
	db *gorm.DB
}

func NewSystemConfigService(db *gorm.DB) *SystemConfigService {
	return &SystemConfigService{db: db}
}

func (s *SystemConfigService) GetConfig(key string) (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := s.db.Where("config_key = ?", key).First(&config).Error
	return &config, err
}

func (s *SystemConfigService) GetAllConfigs() (map[string]interface{}, error) {
	var configs []model.SystemConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, cfg := range configs {
		switch cfg.Type {
		case "number":
			var val int
			fmt.Sscanf(cfg.Value, "%d", &val)
			result[cfg.ConfigKey] = val
		case "boolean":
			result[cfg.ConfigKey] = cfg.Value == "true"
		case "json":
			result[cfg.ConfigKey] = cfg.Value
		default:
			result[cfg.ConfigKey] = cfg.Value
		}
	}
	return result, nil
}

func (s *SystemConfigService) UpdateConfig(config *model.SystemConfig) error {
	return s.db.Save(config).Error
}

func (s *SystemConfigService) CreateConfig(config *model.SystemConfig) error {
	return s.db.Create(config).Error
}

var publicConfigKeys = []string{
	"enableAI",
	"enableReadReceipt",
	"messageRecallTime",
}

func (s *SystemConfigService) GetPublicConfigs() (map[string]interface{}, error) {
	var configs []model.SystemConfig
	if err := s.db.Where("config_key IN ?", publicConfigKeys).Find(&configs).Error; err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, cfg := range configs {
		switch cfg.Type {
		case "number":
			var val int
			fmt.Sscanf(cfg.Value, "%d", &val)
			result[cfg.ConfigKey] = val
		case "boolean":
			result[cfg.ConfigKey] = cfg.Value == "true"
		default:
			result[cfg.ConfigKey] = cfg.Value
		}
	}

	if _, ok := result["enableAI"]; !ok {
		result["enableAI"] = true
	}
	if _, ok := result["enableReadReceipt"]; !ok {
		result["enableReadReceipt"] = true
	}
	if _, ok := result["messageRecallTime"]; !ok {
		result["messageRecallTime"] = 120
	}

	return result, nil
}

func (s *SystemConfigService) BatchUpdate(configs map[string]interface{}) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range configs {
			var cfg model.SystemConfig
			result := tx.Where("config_key = ?", key).First(&cfg)

			strValue := fmt.Sprintf("%v", value)
			cfgType := "string"

			if _, ok := value.(float64); ok {
				cfgType = "number"
			} else if _, ok := value.(bool); ok {
				cfgType = "boolean"
			}

			if result.Error != nil {
				cfg = model.SystemConfig{
					ConfigKey: key,
					Value:     strValue,
					Type:      cfgType,
				}
				if err := tx.Create(&cfg).Error; err != nil {
					return err
				}
			} else {
				cfg.Value = strValue
				cfg.Type = cfgType
				if err := tx.Save(&cfg).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
