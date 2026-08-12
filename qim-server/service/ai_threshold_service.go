package service

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// AiThresholdService AI 阈值读写服务。
// 从 system_configs 表读取阈值，支持运行时热更新（改完即生效）。
type AiThresholdService struct {
	db         *gorm.DB
	cache      map[string]float64
	cacheExpiry time.Time
	mu         sync.RWMutex
	cacheTTL   time.Duration
}

func NewAiThresholdService(db *gorm.DB) *AiThresholdService {
	return &AiThresholdService{
		db:       db,
		cache:    make(map[string]float64),
		cacheTTL: 5 * time.Minute,
	}
}

// SeedFromYAML 读取 config.yaml 中已废弃的 knowledge.score_threshold 字段，
// 若 DB 中尚无 ai.knowledge_score_threshold 记录则自动迁移（兼容老版本升级）。
// 已在 DB 中有记录时不做任何写入（DB 优先）。
func (s *AiThresholdService) SeedFromYAML() {
	if s.db == nil {
		return
	}

	// 检查 DB 是否已有记录
	var count int64
	s.db.Model(&model.SystemConfig{}).Where("config_key = ?", "ai.knowledge_score_threshold").Count(&count)
	if count > 0 {
		return // DB 已有值，不覆盖
	}

	// 读取 config.yaml 的原始 YAML，查找已废弃的 knowledge.score_threshold
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return // 文件不存在或不可读，跳过
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}
	knowledge, ok := raw["knowledge"].(map[string]interface{})
	if !ok {
		return
	}
	val, ok := knowledge["score_threshold"].(float64)
	if !ok {
		return
	}

	// 写入 DB
	cfg := model.SystemConfig{
		ConfigKey: "ai.knowledge_score_threshold",
		Value:     fmt.Sprintf("%g", val),
		Type:      "number",
		Desc:      "从 config.yaml knowledge.score_threshold 自动迁移",
	}
	if err := s.db.Create(&cfg).Error; err != nil {
		logger.WithModule("AiThreshold").Warn("迁移 knowledge.score_threshold 失败", "error", err)
		return
	}
	s.mu.Lock()
	s.cache["ai.knowledge_score_threshold"] = val
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	s.mu.Unlock()
	logger.WithModule("AiThreshold").Info("已从 config.yaml 迁移 knowledge.score_threshold", "value", val)
}

// GetFloat 读取指定阈值：先查缓存，缓存未命中则查 DB，DB 无记录返回默认值。
func (s *AiThresholdService) GetFloat(key string, defaultVal float64) float64 {
	s.mu.RLock()
	if v, ok := s.cache[key]; ok && time.Now().Before(s.cacheExpiry) {
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	// 缓存过期，从 DB 读取
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查：另一个 goroutine 可能刚刷新过缓存
	if v, ok := s.cache[key]; ok && time.Now().Before(s.cacheExpiry) {
		return v
	}

	if s.db == nil {
		return defaultVal
	}
	var cfg model.SystemConfig
	if err := s.db.Where("config_key = ?", key).First(&cfg).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			logger.WithModule("AiThreshold").Warn("读取阈值失败", "key", key, "error", err)
		}
		return defaultVal
	}

	var val float64
	fmt.Sscanf(cfg.Value, "%f", &val)
	s.cache[key] = val
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	return val
}

// GetInt 读取整数型阈值。
func (s *AiThresholdService) GetInt(key string, defaultVal int) int {
	return int(s.GetFloat(key, float64(defaultVal)))
}

// GetAll 返回所有阈值的当前值（含默认值兜底），供管理后台展示。
func (s *AiThresholdService) GetAll() map[string]interface{} {
	result := make(map[string]interface{})
	for _, t := range config.Thresholds {
		result[t.Key] = s.GetFloat(t.Key, t.Default)
	}
	return result
}

// BatchUpdate 批量更新阈值并刷新缓存。
func (s *AiThresholdService) BatchUpdate(configs map[string]interface{}) error {
	// 校验所有 key 是否合法
	for key := range configs {
		valid := false
		for _, t := range config.Thresholds {
			if t.Key == key {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("未知的阈值 key: %s", key)
		}
	}

	// 校验范围
	for key, val := range configs {
		f, ok := val.(float64)
		if !ok {
			if i, ok := val.(int); ok {
				f = float64(i)
			} else {
				return fmt.Errorf("阈值 %s 的值类型不正确", key)
			}
		}
		for _, t := range config.Thresholds {
			if t.Key == key && (f < t.Min || f > t.Max) {
				return fmt.Errorf("阈值 %s 的值 %.2f 超出范围 [%.2f, %.2f]", key, f, t.Min, t.Max)
			}
		}
	}

	// 写入 DB（复用 SystemConfigService 的 BatchUpdate 逻辑）
	if s.db == nil {
		return fmt.Errorf("数据库未初始化，无法保存阈值")
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range configs {
			var cfg model.SystemConfig
			result := tx.Where("config_key = ?", key).First(&cfg)
			strValue := fmt.Sprintf("%v", value)
			if result.Error != nil {
				cfg = model.SystemConfig{ConfigKey: key, Value: strValue, Type: "number"}
				if err := tx.Create(&cfg).Error; err != nil {
					return err
				}
			} else {
				cfg.Value = strValue
				cfg.Type = "number"
				if err := tx.Save(&cfg).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 刷新缓存
	s.mu.Lock()
	for key, val := range configs {
		f, _ := val.(float64)
		s.cache[key] = f
	}
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	s.mu.Unlock()

	logger.WithModule("AiThreshold").Info("阈值已更新", "count", len(configs))
	return nil
}
