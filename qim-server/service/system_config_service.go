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

// UpsertConfig 幂等 upsert 一条 system_configs 配置（存在则更新 value/type，不存在则创建）。
// 语义与工具面作用域此前经 ToolScopeService.SaveRawConfig 的写入一致，供无缓存的通用
// 配置键（如推荐提示词）在 handler 层经服务读写，不再裸露 db 句柄。
func (s *SystemConfigService) UpsertConfig(key, value, configType, desc string) error {
	cfg := model.SystemConfig{ConfigKey: key, Value: value, Type: configType, Desc: desc}
	if err := s.db.Where("config_key = ?", key).
		Assign(model.SystemConfig{Value: value, Type: configType}).
		FirstOrCreate(&cfg).Error; err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}
	return nil
}

var publicConfigKeys = []string{
	"enableAI",
	"enableReadReceipt",
	"messageRecallTime",
	"messageRemindTime",
	"messageRemindRepeatCooldown",
	// 客户端更新服务器地址：随公开配置下发，客户端启动/登录即拉取，
	// 覆盖打包时烘焙的 QIM_UPDATE_URL，避免更新地址与聊天服务器脱节后无法远程校正。
	"client:update_base_url",
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
	// 发送提醒触发门槛（秒，0=禁止提醒）与同消息重复提醒冷却（秒，0=不限制）。
	// 与 messageRecallTime 同模式：缺省 3600（1 小时），保证旧语义不变。
	if _, ok := result["messageRemindTime"]; !ok {
		result["messageRemindTime"] = 3600
	}
	if _, ok := result["messageRemindRepeatCooldown"]; !ok {
		result["messageRemindRepeatCooldown"] = 3600
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
			} else if _, ok := value.(int); ok {
				cfgType = "number"
			} else if _, ok := value.(int64); ok {
				cfgType = "number"
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
