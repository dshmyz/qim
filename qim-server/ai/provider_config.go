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

	// EmbeddingBaseURL embedding 接口的基础 URL（与 BaseURL 不同时显式配置）。
	// 火山引擎等平台的 embedding 路径与 chat 不同（/api/v3/embeddings vs /api/plan/v3/chat/completions）。
	// 空值时回退到 BaseURL。
	EmbeddingBaseURL string

	// EmbeddingModel embedding 模型名称（与 chat Model 不同时显式配置）。
	// 空值时回退到 Model（chat 模型，多数场景不适用 embedding）。
	EmbeddingModel string

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
		APIKey:           c.APIKey,
		Model:            c.Model,
		BaseURL:          c.BaseURL,
		EmbeddingBaseURL: c.EmbeddingBaseURL,
		EmbeddingModel:   c.EmbeddingModel,
		ExtraParams:      map[string]interface{}{},
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
