package ai

import (
	"fmt"
)

type ProviderFactory struct{}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

func (f *ProviderFactory) CreateProvider(cfg *AIConfig) (Provider, error) {
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

func (f *ProviderFactory) createOpenAIProvider(cfg *AIConfig) Provider {
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

func (f *ProviderFactory) createBaiduProvider(cfg *AIConfig) Provider {
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

func (f *ProviderFactory) createAlibabaProvider(cfg *AIConfig) Provider {
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

func (f *ProviderFactory) createTencentProvider(cfg *AIConfig) Provider {
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

func (f *ProviderFactory) createBytedanceProvider(cfg *AIConfig) Provider {
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

func (f *ProviderFactory) createAnthropicProvider(cfg *AIConfig) Provider {
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
