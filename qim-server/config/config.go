package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"qim-server/ai"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Cluster  ClusterConfig
	Storage  StorageConfig
	AI       ai.AIConfig
	CORS     CORSConfig
	WS       WSConfig
	Vector   VectorConfig
	DataInit DataInitConfig
}

type DataInitConfig struct {
	PresetData   bool `yaml:"preset_data"`
	BotTemplates bool `yaml:"bot_templates"`
	DemoMiniApps bool `yaml:"demo_mini_apps"`
	TestData     bool `yaml:"test_data"`
	SeedForce    bool `yaml:"seed_force"`
}

type VectorConfig struct {
	Path string `yaml:"path"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type WSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
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
	Expire            int    `yaml:"expire"`
	RefreshExpireDays int    `yaml:"refresh_expire_days"`
}

type DatabaseConfig struct {
	Type         string `yaml:"type"`
	Path         string `yaml:"path"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxLifetime  int    `yaml:"max_lifetime"`
}

type yamlConfig struct {
	Server   ServerConfig   `yaml:"server"`
	JWT      JWTConfig      `yaml:"jwt"`
	DB       DatabaseConfig `yaml:"database"`
	Cluster  ClusterConfig  `yaml:"cluster"`
	Storage  StorageConfig  `yaml:"storage"`
	AI       ai.AIConfig    `yaml:"ai"`
	CORS     CORSConfig     `yaml:"cors"`
	WS       WSConfig       `yaml:"ws"`
	Vector   VectorConfig   `yaml:"vector"`
	DataInit DataInitConfig `yaml:"data_init"`
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

	// 确保CORS配置有默认值
	if len(cfg.CORS.AllowedOrigins) == 0 {
		defaultCfg := getDefaultConfig()
		cfg.CORS = defaultCfg.CORS
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
		cfg.AI.Router.Routes[ai.TaskTypeChat] = ai.Route{Provider: aiProvider}
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
	bytedanceAPIKey := os.Getenv("AI_BYTEDANCE_API_KEY")
	if bytedanceAPIKey != "" {
		cfg.AI.Bytedance.APIKey = bytedanceAPIKey
	}

	bytedanceModel := os.Getenv("AI_BYTEDANCE_MODEL")
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

	// JWT 密钥安全校验
	if cfg.JWT.Secret == "" || strings.HasPrefix(cfg.JWT.Secret, "${QIM_JWT_SECRET:") || cfg.JWT.Secret == "change-me-to-random-string" {
		envSecret := os.Getenv("JWT_SECRET")
		if envSecret != "" {
			cfg.JWT.Secret = envSecret
		} else if cfg.Server.Mode == "release" {
			fmt.Fprintln(os.Stderr, "[FATAL] JWT_SECRET 未配置！生产环境必须设置 JWT_SECRET 环境变量或配置文件中的 jwt.secret")
			os.Exit(1)
		} else {
			cfg.JWT.Secret = generateRandomSecret()
			fmt.Println("[WARN] ============================================================")
			fmt.Println("[WARN] JWT_SECRET 未配置，已自动生成随机密钥。")
			fmt.Println("[WARN] 重启后密钥将变化，所有已颁发的 Token 将失效！")
			fmt.Println("[WARN] 请设置 JWT_SECRET 环境变量以使用固定密钥。")
			fmt.Println("[WARN] 生产环境未设置 JWT_SECRET 将拒绝启动。")
			fmt.Println("[WARN] ============================================================")
		}
	}

	// 数据初始化策略：如果配置未显式设置，回退到基于 mode 的默认值
	if cfg.DataInit == (DataInitConfig{}) {
		cfg.DataInit = getDefaultDataInitConfig(cfg.Server.Mode)
	}

	return &Config{
		Server:   cfg.Server,
		Database: cfg.DB,
		JWT:      cfg.JWT,
		Cluster:  cfg.Cluster,
		Storage:  cfg.Storage,
		AI:       cfg.AI,
		CORS:     cfg.CORS,
		WS:       cfg.WS,
		Vector:   cfg.Vector,
		DataInit: cfg.DataInit,
	}
}

func generateRandomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("fallback-insecure-secret-" + fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return hex.EncodeToString(b)
}

func getDefaultDataInitConfig(mode string) DataInitConfig {
	if mode == "release" {
		return DataInitConfig{
			PresetData:   true,
			BotTemplates: true,
			DemoMiniApps: false,
			TestData:     false,
			SeedForce:    false,
		}
	}
	return DataInitConfig{
		PresetData:   true,
		BotTemplates: true,
		DemoMiniApps: true,
		TestData:     true,
		SeedForce:    false,
	}
}

func getDefaultConfig() yamlConfig {
	return yamlConfig{
		Server: ServerConfig{
			Port: "8080",
			Mode: "debug",
		},
		JWT: JWTConfig{
			Secret:            "${QIM_JWT_SECRET:change-me-to-random-string}",
			Expire:            7200,
			RefreshExpireDays: 7,
		},
		DB: DatabaseConfig{
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
		AI: ai.AIConfig{
			MaxTokens:   1000,
			Temperature: 0.7,
			OpenAI: ai.OpenAIConfig{
				APIKey:  "",
				Model:   "gpt-3.5-turbo",
				BaseURL: "https://api.openai.com/v1",
			},
			Baidu: ai.BaiduConfig{
				APIKey:    "",
				SecretKey: "",
				Model:     "ERNIE-Bot-4.0",
				BaseURL:   "https://aip.baidubce.com",
			},
			Alibaba: ai.AlibabaConfig{
				APIKey:    "",
				APISecret: "",
				Model:     "qwen-plus",
				BaseURL:   "https://dashscope.aliyuncs.com/api/v1",
			},
			Tencent: ai.TencentConfig{
				SecretID:  "",
				SecretKey: "",
				Model:     "hunyuan-pro",
				BaseURL:   "https://hunyuan.tencentcloudapi.com",
			},
			Bytedance: ai.BytedanceConfig{
				APIKey:  "",
				Model:   "doubao-pro-1.0",
				BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
			},
			Anthropic: ai.AnthropicConfig{
				APIKey:  "",
				Model:   "claude-3-5-sonnet-20241022",
				BaseURL: "https://api.anthropic.com/v1",
			},
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		WS: WSConfig{
			AllowedOrigins: nil,
		},
		Vector: VectorConfig{
			Path: "./data/vector.db",
		},
	}
}
