package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const configDir = ConfigDirName

type config struct {
	ServerURL    string `json:"server_url"`
	BotToken     string `json:"bot_token"`
	UserToken    string `json:"user_token"`    // 用户令牌（JWT 或 qusr_ 长期令牌），用于以用户身份调 /api/v1/*（任务/日历等）
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
		Short: fmt.Sprintf("写配置（~/%s/config.json）", ConfigDirName),
		Run: func(cmd *cobra.Command, args []string) {
			// 合并语义：--server/--token 更新连接配置，--user-token 更新用户令牌；
			// 未指定的字段保留旧值（避免重设 bot 令牌时清掉已配置的 user_token）。
			old, _ := loadConfig() // 首次配置时无旧文件，零值即可
			if server == "" && token == "" {
				if userToken == "" {
					die("--server 与 --token 必填（或先配置后再用 --user-token 追加）")
				}
				if old.ServerURL == "" || old.BotToken == "" {
					die("先 %s config set --server URL --token T 配置服务端，再追加 --user-token", CliName)
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
				ServerURL:    strings.TrimRight(server, "/"),
				BotToken:     token,
				UserToken:    old.UserToken,
				RefreshToken: old.RefreshToken,
			}
			// 显式指定 --user-token 视为整体替换：旧 refresh_token 属于旧令牌，一并作废
			if userToken != "" {
				cfg.UserToken = userToken
				cfg.RefreshToken = ""
			}
			if err := saveConfig(cfg); err != nil {
				die("保存配置失败: %v", err)
			}
			fmt.Println("配置已保存到", configPath())
		},
	}
	cmd.Flags().StringVar(&server, "server", "", fmt.Sprintf("%s 服务器地址，如 http://localhost:8080", AppName))
	cmd.Flags().StringVar(&token, "token", "", "bot 访问令牌 qbot_...")
	cmd.Flags().StringVar(&userToken, "user-token", "", "用户令牌（可选）：qim login 的 JWT，或客户端「设置 → 开发者设置」签发的 qusr_ 长期令牌")
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
		die("读取配置失败（先 %s config set）: %v", CliName, err)
	}
	if cfg.ServerURL == "" || cfg.BotToken == "" {
		die("配置不完整，先 %s config set --server URL --token T", CliName)
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
