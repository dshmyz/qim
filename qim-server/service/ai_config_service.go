package service

import (
	"encoding/json"
	"errors"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/utils"
	"log"
	"time"

	"gorm.io/gorm"
)

var ErrConfigNotFound = errors.New("config not found")
var ErrConfigLimitExceeded = errors.New("config limit exceeded")
var ErrConfigInUse = errors.New("config is in use")
var ErrUnsupportedProvider = errors.New("unsupported provider")

type AIConfigService struct {
	db              *gorm.DB
	providerFactory *ai.ProviderFactory
}

func NewAIConfigService(db *gorm.DB, factory *ai.ProviderFactory) *AIConfigService {
	return &AIConfigService{
		db:              db,
		providerFactory: factory,
	}
}

func (s *AIConfigService) GetDefaultConfig(userID uint) (*model.AIConfig, error) {
	var config model.AIConfig
	err := s.db.Where("user_id = ? AND is_default = ?", userID, true).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

func (s *AIConfigService) UpdateDefaultConfig(userID uint, provider string, apiKey string, secretKey string, secretID string, modelName string, baseURL string, maxTokens int, temperature float64) (*model.AIConfig, error) {
	var config model.AIConfig
	err := s.db.Where("user_id = ? AND is_default = ?", userID, true).First(&config).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		config = model.AIConfig{
			UserID:    userID,
			IsDefault: true,
		}
	}

	var providerConfig model.AIProviderConfig
	switch provider {
	case "openai":
		providerConfig = model.OpenAIProviderConfig{
			Provider: provider,
			APIKey:   apiKey,
			Model:    modelName,
			BaseURL:  baseURL,
		}
	case "baidu":
		providerConfig = model.BaiduProviderConfig{
			Provider:  provider,
			APIKey:    apiKey,
			SecretKey: secretKey,
			Model:     modelName,
			BaseURL:   baseURL,
		}
	case "alibaba":
		providerConfig = model.AlibabaProviderConfig{
			Provider: provider,
			APIKey:   apiKey,
			Model:    modelName,
			BaseURL:  baseURL,
		}
	case "tencent":
		providerConfig = model.TencentProviderConfig{
			Provider:  provider,
			SecretID:  secretID,
			SecretKey: secretKey,
			Model:     modelName,
			BaseURL:   baseURL,
		}
	case "bytedance":
		providerConfig = model.BytedanceProviderConfig{
			Provider: provider,
			APIKey:   apiKey,
			Model:    modelName,
			BaseURL:  baseURL,
		}
	case "anthropic":
		providerConfig = model.AnthropicProviderConfig{
			Provider: provider,
			APIKey:   apiKey,
			Model:    modelName,
			BaseURL:  baseURL,
		}
	default:
		return nil, ErrUnsupportedProvider
	}

	if err := config.SetProviderConfig(providerConfig); err != nil {
		return nil, err
	}

	if modelName != "" {
		config.ModelName = modelName
	}
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	if maxTokens > 0 {
		config.MaxTokens = maxTokens
	}
	if temperature > 0 {
		config.Temperature = temperature
	}

	if config.ID == 0 {
		if err := s.db.Create(&config).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Save(&config).Error; err != nil {
			return nil, err
		}
	}

	return &config, nil
}

func (s *AIConfigService) ListUserConfigs(userID uint, page int, pageSize int) ([]model.AIConfig, int64, error) {
	var configs []model.AIConfig
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&configs).Error
	if err != nil {
		return nil, 0, err
	}

	var total int64
	s.db.Model(&model.AIConfig{}).Where("user_id = ?", userID).Count(&total)

	return configs, total, nil
}

func (s *AIConfigService) CreateConfig(userID uint, configName string, provider string, apiKey string, modelName string, baseURL string, overridesJSON string) (*model.AIConfig, error) {
	var count int64
	s.db.Model(&model.AIConfig{}).Where("user_id = ?", userID).Count(&count)
	if count >= 5 {
		return nil, ErrConfigLimitExceeded
	}

	encryptedKey, err := utils.EncryptAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	verified := s.TestConnection(provider, apiKey, modelName, baseURL)

	config := model.AIConfig{
		UserID:          userID,
		ConfigName:      configName,
		Provider:        provider,
		APIKeyEncrypted: encryptedKey,
		ModelName:       modelName,
		BaseURL:         baseURL,
		IsVerified:      verified,
		Overrides:       overridesJSON,
	}

	now := time.Now()
	config.LastTestedAt = &now

	if err := s.db.Create(&config).Error; err != nil {
		return nil, err
	}

	return &config, nil
}

func (s *AIConfigService) UpdateConfig(userID uint, configID uint, configName string, provider string, apiKey string, modelName string, baseURL string) (*model.AIConfig, error) {
	var config model.AIConfig
	err := s.db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	// apiKey 为空时保留原密钥，测试连接也用原密钥（避免空 key 导致 openai 测试必然失败）
	keyForTest := apiKey
	if apiKey != "" {
		encryptedKey, err := utils.EncryptAPIKey(apiKey)
		if err != nil {
			return nil, err
		}
		config.APIKeyEncrypted = encryptedKey
	} else {
		// 解密原密钥用于测试连接
		decrypted, err := utils.DecryptAPIKey(config.APIKeyEncrypted)
		if err == nil {
			keyForTest = decrypted
		}
	}

	config.ConfigName = configName
	config.Provider = provider
	config.ModelName = modelName
	config.BaseURL = baseURL

	verified := s.TestConnection(provider, keyForTest, modelName, baseURL)
	config.IsVerified = verified
	now := time.Now()
	config.LastTestedAt = &now

	if err := s.db.Save(&config).Error; err != nil {
		return nil, err
	}

	return &config, nil
}

func (s *AIConfigService) DeleteConfig(userID uint, configID uint) error {
	var config model.AIConfig
	err := s.db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrConfigNotFound
		}
		return err
	}

	var botCount int64
	s.db.Model(&model.Bot{}).Where("user_config_id = ?", configID).Count(&botCount)
	if botCount > 0 {
		return ErrConfigInUse
	}

	return s.db.Delete(&config).Error
}

func (s *AIConfigService) TestConfig(userID uint, configID uint) (bool, error) {
	var config model.AIConfig
	err := s.db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, ErrConfigNotFound
		}
		return false, err
	}

	apiKey, err := utils.DecryptAPIKey(config.APIKeyEncrypted)
	if err != nil {
		return false, err
	}

	verified := s.TestConnection(config.Provider, apiKey, config.ModelName, config.BaseURL)

	now := time.Now()
	s.db.Model(&config).Updates(map[string]interface{}{
		"is_verified":    verified,
		"last_tested_at": now,
	})

	return verified, nil
}

func (s *AIConfigService) TestConnection(provider string, apiKey string, modelName string, baseURL string) bool {
	switch provider {
	case "openai":
		return s.testOpenAIConnection(apiKey, modelName, baseURL)
	default:
		return true
	}
}

func (s *AIConfigService) testOpenAIConnection(apiKey string, modelName string, baseURL string) bool {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	cfg := &ai.AIConfig{
		OpenAI: ai.OpenAIConfig{
			APIKey:  apiKey,
			Model:   modelName,
			BaseURL: baseURL,
		},
	}

	provider, err := s.providerFactory.CreateProvider(cfg)
	if err != nil {
		return false
	}

	if !provider.IsConfigured() {
		return false
	}

	_, err = provider.Chat([]ai.Message{
		{Role: "user", Content: "Hi"},
	})

	return err == nil
}

func (s *AIConfigService) GetUserOverrides(userID uint) []ai.Override {
	config, err := s.GetDefaultConfig(userID)
	if err != nil || config == nil || config.Overrides == "" {
		return nil
	}
	var overrides []ai.Override
	if err := json.Unmarshal([]byte(config.Overrides), &overrides); err != nil {
		log.Printf("[AIConfigService] Failed to unmarshal overrides for user %d: %v", userID, err)
		return nil
	}
	return overrides
}

// customProvider 解析出的「用户自选模型」provider，供采用显式 ProviderConfig 的调用路径使用。
// ProviderName 为空表示未命中/解析失败，调用方应回退系统默认配置。
type customProvider struct {
	ProviderName string
	Config       ai.ProviderConfig
}

// resolveUserAIConfigProvider 按 configID（归属 userID）解析用户自选的 AIConfig → provider 配置。
// 供分身（AvatarReplyGraph）与 bot internal_ai 回复共用；任何一步失败返回 nil 表示回退系统默认。
func resolveUserAIConfigProvider(db *gorm.DB, userID, configID uint) *customProvider {
	var cfg model.AIConfig
	if err := db.Where("id = ? AND user_id = ?", configID, userID).First(&cfg).Error; err != nil {
		log.Printf("[resolveUserAIConfigProvider] 自选模型配置不存在或非本人所有，回退系统默认: userID=%d configID=%d err=%v", userID, configID, err)
		return nil
	}
	if !cfg.AIEnabled {
		log.Printf("[resolveUserAIConfigProvider] 自选模型配置已禁用，回退系统默认: userID=%d configID=%d", userID, configID)
		return nil
	}
	apiKey, err := utils.DecryptAPIKey(cfg.APIKeyEncrypted)
	if err != nil {
		log.Printf("[resolveUserAIConfigProvider] 自选模型密钥解密失败，回退系统默认: userID=%d configID=%d err=%v", userID, configID, err)
		return nil
	}
	return &customProvider{
		ProviderName: cfg.Provider,
		Config: ai.ProviderConfig{
			APIKey:      apiKey,
			Model:       cfg.ModelName,
			BaseURL:     cfg.BaseURL,
			ExtraParams: buildCustomProviderExtraParams(cfg.MaxTokens, cfg.Temperature),
		},
	}
}
