package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	JWT     JWTConfig     `yaml:"jwt"`
	DB      DBConfig      `yaml:"database"`
	Cluster ClusterConfig `yaml:"cluster"`
	Storage StorageConfig `yaml:"storage"`
	AI      AIConfig      `yaml:"ai"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

type ClusterConfig struct {
	Enabled bool     `yaml:"enabled"`
	Nodes   []string `yaml:"nodes"`
}

type StorageConfig struct {
	Type  string             `yaml:"type"`
	Local LocalStorageConfig `yaml:"local"`
	S3    S3StorageConfig    `yaml:"s3"`
}

type LocalStorageConfig struct {
	Path string `yaml:"path"`
}

type S3StorageConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`
	UseSSL    bool   `yaml:"use_ssl"`
}

type JWTConfig struct {
	Secret            string `yaml:"secret"`
	ExpireHours       int    `yaml:"expire_hours"`
	RefreshExpireDays int    `yaml:"refresh_expire_days"`
}

type DBConfig struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type AIConfig struct {
	Provider    string          `yaml:"provider"`
	MaxTokens   int             `yaml:"max_tokens"`
	Temperature float64         `yaml:"temperature"`
	OpenAI      OpenAIConfig    `yaml:"openai"`
	Baidu       BaiduConfig     `yaml:"baidu"`
	Alibaba     AlibabaConfig   `yaml:"alibaba"`
	Tencent     TencentConfig   `yaml:"tencent"`
	Bytedance   BytedanceConfig `yaml:"bytedance"`
	Anthropic   AnthropicConfig `yaml:"anthropic"`
}

type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type BaiduConfig struct {
	APIKey    string `yaml:"api_key"`
	SecretKey string `yaml:"secret_key"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
}

type AlibabaConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
}

type TencentConfig struct {
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
}

type BytedanceConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type AnthropicConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type yamlConfig struct {
	Server  ServerConfig  `yaml:"server"`
	JWT     JWTConfig     `yaml:"jwt"`
	DB      DBConfig      `yaml:"database"`
	Cluster ClusterConfig `yaml:"cluster"`
	Storage StorageConfig `yaml:"storage"`
	AI      AIConfig      `yaml:"ai"`
}

func Load() *Config {
	var cfg yamlConfig

	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("配置文件读取失败: %v，使用默认配置\n", err)
		cfg = getDefaultConfig()
	} else {
		err = yaml.Unmarshal(yamlFile, &cfg)
		if err != nil {
			fmt.Printf("配置文件解析失败: %v，使用默认配置\n", err)
			cfg = getDefaultConfig()
		}
	}

	port := os.Getenv("PORT")
	if port != "" {
		cfg.Server.Port = port
	}

	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		cfg.JWT.Secret = secret
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath != "" {
		cfg.DB.Path = dbPath
	}

	// 加载AI配置
	aiProvider := os.Getenv("AI_PROVIDER")
	if aiProvider != "" {
		cfg.AI.Provider = aiProvider
	}

	// OpenAI配置
	openaiAPIKey := os.Getenv("AI_OPENAI_API_KEY")
	if openaiAPIKey != "" {
		cfg.AI.OpenAI.APIKey = openaiAPIKey
	}

	openaiModel := os.Getenv("AI_OPENAI_MODEL")
	if openaiModel != "" {
		cfg.AI.OpenAI.Model = openaiModel
	}

	// 百度文心一言配置
	baiduAPIKey := os.Getenv("AI_BAIDU_API_KEY")
	if baiduAPIKey != "" {
		cfg.AI.Baidu.APIKey = baiduAPIKey
	}

	baiduSecretKey := os.Getenv("AI_BAIDU_SECRET_KEY")
	if baiduSecretKey != "" {
		cfg.AI.Baidu.SecretKey = baiduSecretKey
	}

	baiduModel := os.Getenv("AI_BAIDU_MODEL")
	if baiduModel != "" {
		cfg.AI.Baidu.Model = baiduModel
	}

	// 阿里通义千问配置
	alibabaAPIKey := os.Getenv("AI_ALIBABA_API_KEY")
	if alibabaAPIKey != "" {
		cfg.AI.Alibaba.APIKey = alibabaAPIKey
	}

	alibabaAPISecret := os.Getenv("AI_ALIBABA_API_SECRET")
	if alibabaAPISecret != "" {
		cfg.AI.Alibaba.APISecret = alibabaAPISecret
	}

	alibabaModel := os.Getenv("AI_ALIBABA_MODEL")
	if alibabaModel != "" {
		cfg.AI.Alibaba.Model = alibabaModel
	}

	// 腾讯混元大模型配置
	tencentSecretID := os.Getenv("AI_TENCENT_SECRET_ID")
	if tencentSecretID != "" {
		cfg.AI.Tencent.SecretID = tencentSecretID
	}

	tencentSecretKey := os.Getenv("AI_TENCENT_SECRET_KEY")
	if tencentSecretKey != "" {
		cfg.AI.Tencent.SecretKey = tencentSecretKey
	}

	tencentModel := os.Getenv("AI_TENCENT_MODEL")
	if tencentModel != "" {
		cfg.AI.Tencent.Model = tencentModel
	}

	// 字节跳动豆包配置
	bytedanceAPIKey := os.Getenv("AI_BYTEANCE_API_KEY")
	if bytedanceAPIKey != "" {
		cfg.AI.Bytedance.APIKey = bytedanceAPIKey
	}

	bytedanceModel := os.Getenv("AI_BYTEANCE_MODEL")
	if bytedanceModel != "" {
		cfg.AI.Bytedance.Model = bytedanceModel
	}

	// Anthropic/Claude配置
	anthropicAPIKey := os.Getenv("AI_ANTHROPIC_API_KEY")
	if anthropicAPIKey != "" {
		cfg.AI.Anthropic.APIKey = anthropicAPIKey
	}

	anthropicModel := os.Getenv("AI_ANTHROPIC_MODEL")
	if anthropicModel != "" {
		cfg.AI.Anthropic.Model = anthropicModel
	}

	anthropicBaseURL := os.Getenv("AI_ANTHROPIC_BASE_URL")
	if anthropicBaseURL != "" {
		cfg.AI.Anthropic.BaseURL = anthropicBaseURL
	}

	return &Config{
		Server:  cfg.Server,
		JWT:     cfg.JWT,
		DB:      cfg.DB,
		Cluster: cfg.Cluster,
		Storage: cfg.Storage,
		AI:      cfg.AI,
	}
}

func getDefaultConfig() yamlConfig {
	return yamlConfig{
		Server: ServerConfig{
			Port: "8080",
			Mode: "debug",
		},
		JWT: JWTConfig{
			Secret:            "your-secret-key-change-in-production",
			ExpireHours:       2,
			RefreshExpireDays: 7,
		},
		DB: DBConfig{
			Path: "./qim.db",
		},
		Cluster: ClusterConfig{
			Enabled: false,
			Nodes:   []string{},
		},
		Storage: StorageConfig{
			Type: "local",
			Local: LocalStorageConfig{
				Path: "./uploads",
			},
			S3: S3StorageConfig{
				Endpoint:  "s3.amazonaws.com",
				AccessKey: "your-access-key",
				SecretKey: "your-secret-key",
				Bucket:    "qim",
				Region:    "us-east-1",
				UseSSL:    true,
			},
		},
		AI: AIConfig{
			Provider:    "openai",
			MaxTokens:   1000,
			Temperature: 0.7,
			OpenAI: OpenAIConfig{
				APIKey:  "",
				Model:   "gpt-3.5-turbo",
				BaseURL: "https://api.openai.com/v1",
			},
			Baidu: BaiduConfig{
				APIKey:    "",
				SecretKey: "",
				Model:     "ERNIE-Bot-4.0",
				BaseURL:   "https://aip.baidubce.com",
			},
			Alibaba: AlibabaConfig{
				APIKey:    "",
				APISecret: "",
				Model:     "qwen-plus",
				BaseURL:   "https://dashscope.aliyuncs.com/api/v1",
			},
			Tencent: TencentConfig{
				SecretID:  "",
				SecretKey: "",
				Model:     "hunyuan-pro",
				BaseURL:   "https://hunyuan.tencentcloudapi.com",
			},
			Bytedance: BytedanceConfig{
				APIKey:  "",
				Model:   "doubao-pro-1.0",
				BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
			},
			Anthropic: AnthropicConfig{
				APIKey:  "",
				Model:   "claude-3-5-sonnet-20241022",
				BaseURL: "https://api.anthropic.com/v1",
			},
		},
	}
}
