package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const configDir = ".qim"

type config struct {
	ServerURL    string `json:"server_url"`
	BotToken     string `json:"bot_token"`
	UserToken    string `json:"user_token"`    // 用户 JWT，用于以用户身份调 /api/v1/*（任务/日历等）
	RefreshToken string `json:"refresh_token"` // 用于自动续期 user_token
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "管理 CLI 配置",
	}
	cmd.AddCommand(newConfigSetCmd(), newConfigShowCmd())
	return cmd
}

func newConfigSetCmd() *cobra.Command {
	var server, token, userToken string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "写配置（~/.qim/config.json）",
		Run: func(cmd *cobra.Command, args []string) {
			if server == "" && token == "" {
				if userToken == "" {
					die("--server 与 --token 必填（或先配置后再用 --user-token 追加）")
				}
				old, err := loadConfig()
				if err != nil {
					die("读取旧配置失败: %v", err)
				}
				old.UserToken = userToken
				if err := saveConfig(old); err != nil {
					die("保存配置失败: %v", err)
				}
				fmt.Println("配置已保存到", configPath())
				return
			}
			if server == "" || token == "" {
				die("--server 与 --token 必填")
			}
			cfg := config{
				ServerURL: strings.TrimRight(server, "/"),
				BotToken:  token,
				UserToken: userToken,
			}
			if err := saveConfig(cfg); err != nil {
				die("保存配置失败: %v", err)
			}
			fmt.Println("配置已保存到", configPath())
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "QIM 服务器地址，如 http://localhost:8080")
	cmd.Flags().StringVar(&token, "token", "", "bot 访问令牌 qbot_...")
	cmd.Flags().StringVar(&userToken, "user-token", "", "用户 JWT，用于以用户身份建任务/日历（可选）")
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "显示配置（token 脱敏）",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := loadConfig()
			if err != nil {
				die("读取配置失败: %v", err)
			}
			mask := cfg.BotToken
			if len(mask) > 12 {
				mask = mask[:8] + "..." + mask[len(mask)-4:]
			}
			utMask := cfg.UserToken
			if len(utMask) > 20 {
				utMask = utMask[:8] + "..." + utMask[len(utMask)-4:]
			}
			fmt.Printf("server_url:  %s\nbot_token:   %s\nuser_token:  %s\n", cfg.ServerURL, mask, utMask)
		},
	}
}

// ---------- config io ----------

func mustConfig() config {
	cfg, err := loadConfig()
	if err != nil {
		die("读取配置失败（先 qim config set）: %v", err)
	}
	if cfg.ServerURL == "" || cfg.BotToken == "" {
		die("配置不完整，先 qim config set --server URL --token T")
	}
	return cfg
}

func loadConfig() (config, error) {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	return os.WriteFile(configPath(), b, 0o600)
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, configDir, "config.json")
	}
	return filepath.Join(configDir, "config.json")
}
