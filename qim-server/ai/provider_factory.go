package ai

import (
	"fmt"
	"qim-server/config"
)

// ProviderFactory 根据配置创建对应的 Provider
type ProviderFactory struct{}

// NewProviderFactory 创建提供商工厂
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider 根据配置创建对应的 AI 提供商
func (f *ProviderFactory) CreateProvider(cfg *config.AIConfig) (Provider, error) {
	switch cfg.Provider {
	case "openai":
		return f.createOpenAIProvider(cfg), nil
	case "baidu":
		return f.createBaiduProvider(cfg), nil
	case "alibaba":
		return f.createAlibabaProvider(cfg), nil
	case "tencent":
		return f.createTencentProvider(cfg), nil
	case "bytedance":
		return f.createBytedanceProvider(cfg), nil
	case "anthropic":
		return f.createAnthropicProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}

func (f *ProviderFactory) createOpenAIProvider(cfg *config.AIConfig) Provider {
	return NewOpenAIProvider(ProviderConfig{
		APIKey:  cfg.OpenAI.APIKey,
		Model:   cfg.OpenAI.Model,
		BaseURL: cfg.OpenAI.BaseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens":  cfg.MaxTokens,
			"temperature": cfg.Temperature,
		},
	})
}

func (f *ProviderFactory) createBaiduProvider(cfg *config.AIConfig) Provider {
	return NewBaiduProvider(ProviderConfig{
		APIKey:    cfg.Baidu.APIKey,
		APISecret: cfg.Baidu.SecretKey,
		Model:     cfg.Baidu.Model,
		BaseURL:   cfg.Baidu.BaseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens":  cfg.MaxTokens,
			"temperature": cfg.Temperature,
		},
	})
}

func (f *ProviderFactory) createAlibabaProvider(cfg *config.AIConfig) Provider {
	return NewAlibabaProvider(ProviderConfig{
		APIKey:  cfg.Alibaba.APIKey,
		Model:   cfg.Alibaba.Model,
		BaseURL: cfg.Alibaba.BaseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens":  cfg.MaxTokens,
			"temperature": cfg.Temperature,
		},
	})
}

func (f *ProviderFactory) createTencentProvider(cfg *config.AIConfig) Provider {
	return NewTencentProvider(ProviderConfig{
		APIKey:    cfg.Tencent.SecretID,
		APISecret: cfg.Tencent.SecretKey,
		Model:     cfg.Tencent.Model,
		BaseURL:   cfg.Tencent.BaseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens":  cfg.MaxTokens,
			"temperature": cfg.Temperature,
		},
	})
}

func (f *ProviderFactory) createBytedanceProvider(cfg *config.AIConfig) Provider {
	return NewBytedanceProvider(ProviderConfig{
		APIKey:  cfg.Bytedance.APIKey,
		Model:   cfg.Bytedance.Model,
		BaseURL: cfg.Bytedance.BaseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens":  cfg.MaxTokens,
			"temperature": cfg.Temperature,
		},
	})
}

func (f *ProviderFactory) createAnthropicProvider(cfg *config.AIConfig) Provider {
	return NewAnthropicProvider(ProviderConfig{
		APIKey:  cfg.Anthropic.APIKey,
		Model:   cfg.Anthropic.Model,
		BaseURL: cfg.Anthropic.BaseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens":  cfg.MaxTokens,
			"temperature": cfg.Temperature,
		},
	})
}
