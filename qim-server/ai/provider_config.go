package ai

// ProviderConfig 统一的 AI 提供商配置
type ProviderConfig struct {
	// APIKey API 密钥（对于 OAuth2 模式，是 client_id）
	APIKey string

	// APISecret API 密钥/密码（用于需要双认证的提供商）
	APISecret string

	// Model 使用的模型名称
	Model string

	// BaseURL API 基础 URL
	BaseURL string

	// ExtraParams 额外的配置参数
	ExtraParams map[string]interface{}
}

// IsSet 检查 APIKey 是否已设置
func (c *ProviderConfig) IsSet() bool {
	return c.APIKey != ""
}

// IsDualKeySet 检查双密钥配置是否已设置
func (c *ProviderConfig) IsDualKeySet() bool {
	return c.APIKey != "" && c.APISecret != ""
}

func (c OpenAIConfig) ToProviderConfig() ProviderConfig {
	return ProviderConfig{
		APIKey:  c.APIKey,
		Model:   c.Model,
		BaseURL: c.BaseURL,
		ExtraParams: map[string]interface{}{
			"embedding_model": c.EmbeddingModel,
		},
	}
}

func (c BaiduConfig) ToProviderConfig() ProviderConfig {
	return ProviderConfig{
		APIKey:    c.APIKey,
		APISecret: c.SecretKey,
		Model:     c.Model,
		BaseURL:   c.BaseURL,
	}
}

func (c AlibabaConfig) ToProviderConfig() ProviderConfig {
	return ProviderConfig{
		APIKey:  c.APIKey,
		Model:   c.Model,
		BaseURL: c.BaseURL,
	}
}

func (c TencentConfig) ToProviderConfig() ProviderConfig {
	return ProviderConfig{
		APIKey:    c.SecretID,
		APISecret: c.SecretKey,
		Model:     c.Model,
		BaseURL:   c.BaseURL,
	}
}

func (c BytedanceConfig) ToProviderConfig() ProviderConfig {
	return ProviderConfig{
		APIKey:  c.APIKey,
		Model:   c.Model,
		BaseURL: c.BaseURL,
	}
}

func (c AnthropicConfig) ToProviderConfig() ProviderConfig {
	return ProviderConfig{
		APIKey:  c.APIKey,
		Model:   c.Model,
		BaseURL: c.BaseURL,
	}
}
