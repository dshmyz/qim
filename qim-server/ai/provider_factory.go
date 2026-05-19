package ai

import (
	"fmt"
)

type ProviderFactory struct{}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

func (f *ProviderFactory) CreateProvider(cfg *AIConfig) (Provider, error) {
	providers := cfg.AllProviders()
	for name, providerCfg := range providers {
		return f.CreateProviderByName(name, providerCfg)
	}
	return nil, fmt.Errorf("no AI provider configured")
}

func (f *ProviderFactory) createOpenAIProvider(cfg *AIConfig) Provider {
	extraParams := map[string]interface{}{
		"max_tokens":  cfg.MaxTokens,
		"temperature": cfg.Temperature,
	}
	if cfg.OpenAI.EmbeddingModel != "" {
		extraParams["embedding_model"] = cfg.OpenAI.EmbeddingModel
	}
	return NewOpenAIProvider(ProviderConfig{
		APIKey:  cfg.OpenAI.APIKey,
		Model:   cfg.OpenAI.Model,
		BaseURL: cfg.OpenAI.BaseURL,
		ExtraParams: extraParams,
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

func (f *ProviderFactory) CreateProviderByName(name string, cfg ProviderConfig) (Provider, error) {
	switch name {
	case "openai", "deepseek":
		return f.createGenericOpenAIProvider(name, cfg), nil
	case "anthropic":
		return f.createAnthropicProviderFromConfig(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", name)
	}
}

func (f *ProviderFactory) createGenericOpenAIProvider(name string, cfg ProviderConfig) Provider {
	extraParams := cfg.ExtraParams
	if extraParams == nil {
		extraParams = make(map[string]interface{})
	}
	if _, ok := extraParams["max_tokens"]; !ok {
		extraParams["max_tokens"] = 1000
	}
	if _, ok := extraParams["temperature"]; !ok {
		extraParams["temperature"] = 0.7
	}
	return NewOpenAIProvider(ProviderConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		ExtraParams: extraParams,
	})
}

func (f *ProviderFactory) createAnthropicProviderFromConfig(cfg ProviderConfig) Provider {
	return NewAnthropicProvider(ProviderConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens":  1000,
			"temperature": 0.7,
		},
	})
}
