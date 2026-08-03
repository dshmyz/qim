package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "验证 user_token 状态，显示当前登录用户",
		Run: func(cmd *cobra.Command, args []string) {
			respBody, err := userGet("/api/v1/users/me")
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data struct {
					ID       uint64 `json:"id"`
					Username string `json:"username"`
					Nickname string `json:"nickname"`
					Avatar   string `json:"avatar"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			name := resp.Data.Nickname
			if name == "" {
				name = resp.Data.Username
			}
			fmt.Printf("用户: %s (ID: %d)\n", name, resp.Data.ID)
		},
	}
}
