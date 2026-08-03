package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd() *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "交互式登录获取 user_token（自动续期）",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := loadConfig()
			if err != nil || cfg.ServerURL == "" {
				die("请先执行: qim config set --server URL --token BOT_TOKEN")
			}

			if username == "" {
				fmt.Print("用户名: ")
				username, _ = bufio.NewReader(os.Stdin).ReadString('\n')
				username = strings.TrimSpace(username)
			}
			if username == "" {
				die("用户名不能为空")
			}

			fmt.Print("密码: ")
			password, err := readPassword()
			if err != nil {
				die("读取密码失败: %v", err)
			}

			body, err := json.Marshal(map[string]string{
				"username": username,
				"password": password,
			})
			if err != nil {
				die("序列化登录请求失败: %v", err)
			}
			req, err := http.NewRequest(http.MethodPost, cfg.ServerURL+"/api/v1/auth/login", bytes.NewReader(body))
			if err != nil {
				die("创建请求失败: %v", err)
			}
			req.Header.Set("Content-Type", "application/json; charset=utf-8")

			respBody, err := doReq(req)
			if err != nil {
				die("登录失败: %v", err)
			}

			var loginResp struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    struct {
					Token        string `json:"token"`
					RefreshToken string `json:"refresh_token"`
					User         struct {
						Nickname string `json:"nickname"`
						Username string `json:"username"`
					} `json:"user"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &loginResp); err != nil || loginResp.Code != 0 {
				die("登录失败: %s", loginResp.Message)
			}

			cfg.UserToken = loginResp.Data.Token
			cfg.RefreshToken = loginResp.Data.RefreshToken
			if err := saveConfig(cfg); err != nil {
				die("保存配置失败: %v", err)
			}
			nick := loginResp.Data.User.Nickname
			if nick == "" {
				nick = loginResp.Data.User.Username
			}
			fmt.Printf("✅ 登录成功：%s（token 7 天有效，自动续期）\n", nick)
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "用户名（可选，不填会交互提示）")
	return cmd
}

// readPassword 从 stdin 读取密码（关闭终端回显）。
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	b, err := term.ReadPassword(fd)
	if err != nil {
		// 非终端回退为普通读取
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return strings.TrimSpace(scanner.Text()), nil
		}
		return "", scanner.Err()
	}
	fmt.Println() // 用户按回车后换行
	return strings.TrimSpace(string(b)), nil
}
