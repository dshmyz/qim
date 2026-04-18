package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	JWT    JWTConfig    `yaml:"jwt"`
	DB     DBConfig     `yaml:"database"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

type JWTConfig struct {
	Secret           string `yaml:"secret"`
	ExpireHours      int    `yaml:"expire_hours"`
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

type yamlConfig struct {
	Server ServerConfig `yaml:"server"`
	JWT    JWTConfig    `yaml:"jwt"`
	DB     DBConfig     `yaml:"database"`
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

	return &Config{
		Server: cfg.Server,
		JWT:    cfg.JWT,
		DB:     cfg.DB,
	}
}

func getDefaultConfig() yamlConfig {
	return yamlConfig{
		Server: ServerConfig{
			Port: "8080",
			Mode: "debug",
		},
		JWT: JWTConfig{
			Secret:           "your-secret-key-change-in-production",
			ExpireHours:      2,
			RefreshExpireDays: 7,
		},
		DB: DBConfig{
			Path: "./qim.db",
		},
	}
}
