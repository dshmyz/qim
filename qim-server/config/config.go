package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Log      LogConfig
	Cluster  ClusterConfig
	Storage  StorageConfig
	AI       ai.AIConfig
	CORS     CORSConfig
	WS       WSConfig
	Vector   VectorConfig
	DataInit DataInitConfig
	Node     NodeConfig
	Static   StaticConfig
	Knowledge KnowledgeConfig
}

// KnowledgeConfig 知识检索配置
type KnowledgeConfig struct {
	// ScoreThreshold 知识来源分数门槛（0-1），低于此分数的召回结果不注入 prompt 也不展示在徽章。
	// 默认 0.6；可经 config.yaml 的 knowledge.score_threshold 覆盖。
	ScoreThreshold float64 `yaml:"score_threshold"`
}

// AiThresholdConfig AI 阈值配置项定义：key、默认值、取值范围说明。
type AiThresholdConfig struct {
	Key         string
	Default     float64
	Min         float64
	Max         float64
	Label       string // 前端展示用的中文标签
	Description string // 前端展示用的说明
}

// Thresholds 所有 AI 阈值的定义列表，供前端渲染表单和后端校验。
var Thresholds = []AiThresholdConfig{
	{Key: "ai.knowledge_score_threshold", Default: 0.6, Min: 0, Max: 1, Label: "知识来源分数门槛", Description: "低于此分数的检索结果不注入 prompt 也不展示在知识来源徽章（0-1）"},
	{Key: "ai.memory_recall_threshold", Default: 0.5, Min: 0, Max: 1, Label: "记忆召回门槛", Description: "分身记忆 Recall 分数门槛，低于此分数时视为知识范围外静默（0-1）"},
	{Key: "ai.conflict_detection_threshold", Default: 0.7, Min: 0, Max: 1, Label: "冲突检测门槛", Description: "新旧记忆分数达到此值时才触发语义冲突检测（0-1）"},
	{Key: "ai.context_history_limit", Default: 20, Min: 5, Max: 100, Label: "上下文历史条数", Description: "注入到 AI prompt 的最近对话消息条数上限"},
	{Key: "ai.recent_ai_messages_limit", Default: 5, Min: 1, Max: 20, Label: "近期 AI 回复条数", Description: "上下文中保留的近期 AI 回复消息条数上限（防自我复制）"},
}

// StaticConfig 静态资源路径配置，避免在 routes.go 中硬编码工作目录相对路径
type StaticConfig struct {
	UploadsDir  string `yaml:"uploads_dir"`  // 上传文件根目录，默认 "uploads"
	MiniAppsDir string `yaml:"miniapps_dir"` // 内置小程序根目录，默认 "static/miniapps"
}

type LogConfig struct {
	Dir   string `yaml:"dir"`
	Level string `yaml:"level"` // debug / info / warn / error，默认 info

	// 日志轮转（lumberjack）策略
	MaxSizeMB  int  `yaml:"max_size_mb"`  // 单文件最大体积（MB），达到后轮转；<=0 用 lumberjack 默认 100
	MaxBackups int  `yaml:"max_backups"`  // 保留的旧文件份数；0 表示保留全部（但受 MaxAgeDays 限制）
	MaxAgeDays int  `yaml:"max_age_days"` // 旧文件最大保留天数；0 表示不按天数清理
	Compress   bool `yaml:"compress"`     // 是否压缩归档的旧文件（gzip）
}

type DataInitConfig struct {
	PresetData   bool `yaml:"preset_data"`
	BotTemplates bool `yaml:"bot_templates"`
	TestData     bool `yaml:"test_data"`
	SeedForce    bool `yaml:"seed_force"`
}

type VectorConfig struct {
	Path string `yaml:"path"`
}

type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

type WSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// ValidateCORS 校验 CORS 配置的合法性。
// 浏览器规范不允许 AllowCredentials=true 且 AllowedOrigins 包含 "*"，
// 此时会将 Origins 设为空切片，由应用层根据请求 Origin 动态设置。
// 返回是否使用了通配符模式（需要动态 Origin 校验）。
func (c *Config) ValidateCORS() bool {
	hasWildcard := false
	for _, o := range c.CORS.AllowedOrigins {
		if o == "*" {
			hasWildcard = true
			break
		}
	}
	// 默认 AllowCredentials 为 true（与 routes.go 中的配置一致）
	if hasWildcard && c.CORS.AllowCredentials {
		c.CORS.AllowedOrigins = []string{}
		return true
	}
	return false
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

type ClusterConfig struct {
	Enabled bool     `yaml:"enabled"`
	Nodes   []string `yaml:"nodes"`
	Scheme  string   `yaml:"scheme"` // 节点间通信协议：http 或 https，默认 http
}

type NodeConfig struct {
	Secret string `yaml:"secret"`
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
	Log      LogConfig      `yaml:"log"`
	Cluster  ClusterConfig  `yaml:"cluster"`
	Storage  StorageConfig  `yaml:"storage"`
	AI       ai.AIConfig    `yaml:"ai"`
	CORS     CORSConfig     `yaml:"cors"`
	WS       WSConfig       `yaml:"ws"`
	Vector   VectorConfig   `yaml:"vector"`
	DataInit DataInitConfig `yaml:"data_init"`
	Node     NodeConfig     `yaml:"node"`
	Static   StaticConfig   `yaml:"static"`
	Knowledge KnowledgeConfig `yaml:"knowledge"`
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
			cfg.JWT.Secret = "qim-dev-default-secret-key-2024"
			fmt.Println("[WARN] ============================================================")
			fmt.Println("[WARN] JWT_SECRET 未配置，已使用固定开发密钥。")
			fmt.Println("[WARN] 请设置 JWT_SECRET 环境变量以使用自定义密钥。")
			fmt.Println("[WARN] 生产环境未设置 JWT_SECRET 将拒绝启动。")
			fmt.Println("[WARN] ============================================================")
		}
	}

	// 数据初始化策略：如果配置未显式设置，回退到基于 mode 的默认值
	if cfg.DataInit == (DataInitConfig{}) {
		cfg.DataInit = getDefaultDataInitConfig(cfg.Server.Mode)
	}

	// 日志目录（环境变量优先，其次 config.yaml 的 log.dir）
	logDir := os.Getenv("LOG_DIR")
	if logDir != "" {
		cfg.Log.Dir = logDir
	}
	// 日志级别（环境变量优先，其次 config.yaml 的 log.level，默认 info）
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Log.Level = level
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}

	// 静态资源路径：未配置时使用与 routes.go 既有硬编码等价的默认值
	if cfg.Static.UploadsDir == "" {
		cfg.Static.UploadsDir = "uploads"
	}
	if cfg.Static.MiniAppsDir == "" {
		cfg.Static.MiniAppsDir = "static/miniapps"
	}
	if v := os.Getenv("QIM_STATIC_UPLOADS_DIR"); v != "" {
		cfg.Static.UploadsDir = v
	}
	if v := os.Getenv("QIM_STATIC_MINIAPPS_DIR"); v != "" {
		cfg.Static.MiniAppsDir = v
	}

	return &Config{
		Server:   cfg.Server,
		Database: cfg.DB,
		JWT:      cfg.JWT,
		Log:      cfg.Log,
		Cluster:  cfg.Cluster,
		Storage:  cfg.Storage,
		AI:       cfg.AI,
		CORS:     cfg.CORS,
		WS:       cfg.WS,
		Vector:   cfg.Vector,
		DataInit: cfg.DataInit,
		Node:     cfg.Node,
		Static:   cfg.Static,
		Knowledge: cfg.Knowledge,
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
			TestData:     false,
			SeedForce:    false,
		}
	}
	return DataInitConfig{
		PresetData:   true,
		BotTemplates: true,
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
			Expire:            604800,
			RefreshExpireDays: 7,
		},
		DB: DatabaseConfig{
			Path: "./qim.db",
		},
		Log: newDefaultLogConfig(),
		Cluster: ClusterConfig{
			Enabled: false,
			Nodes:   []string{},
			Scheme:  "http",
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
			Path: "./data/gracedb",
		},
		Static: StaticConfig{
			UploadsDir:  "uploads",
			MiniAppsDir: "static/miniapps",
		},
		Knowledge: KnowledgeConfig{
			ScoreThreshold: 0.6,
		},
	}
}

// newDefaultLogConfig 构造默认日志配置。
//
// 轮转字段直接引用 logger.DefaultRotateConfig()，避免默认值在 config 与 logger
// 两处分别维护导致不同步（如修改 logger 默认值后忘了同步 config）。
// Dir/Level 在 logger 包中没有对应概念，仍在此处指定。
func newDefaultLogConfig() LogConfig {
	rc := logger.DefaultRotateConfig()
	return LogConfig{
		Dir:        "./logs",
		Level:      "info",
		MaxSizeMB:  rc.MaxSizeMB,
		MaxBackups: rc.MaxBackups,
		MaxAgeDays: rc.MaxAgeDays,
		Compress:   rc.Compress,
	}
}
