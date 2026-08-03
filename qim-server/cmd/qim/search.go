package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

func newMessagesSearchCmd() *cobra.Command {
	var keyword, conv string
	var limit int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "搜索消息（需 user_token）",
		Run: func(cmd *cobra.Command, args []string) {
			if keyword == "" {
				die("--keyword 必填")
			}
			u := fmt.Sprintf("/api/v1/messages/search?keyword=%s&pageSize=%d", url.QueryEscape(keyword), limit)
			if conv != "" {
				u += "&conv_id=" + conv
			}
			respBody, err := userGet(u)
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data struct {
					List []struct {
						ID         uint64 `json:"id"`
						Content    string `json:"content"`
						Type       string `json:"type"`
						ConvID     uint64 `json:"conversation_id"`
						SenderName string `json:"sender_name"`
						CreatedAt  string `json:"created_at"`
					} `json:"list"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			if len(resp.Data.List) == 0 {
				fmt.Println("（无匹配消息）")
				return
			}
			for _, m := range resp.Data.List {
				content := m.Content
				if len([]rune(content)) > 60 {
					content = string([]rune(content)[:60]) + "..."
				}
				sender := m.SenderName
				if sender == "" {
					sender = "?"
				}
				fmt.Printf("#%-6d [%s] %s: %s\n", m.ID, m.Type, sender, content)
			}
		},
	}
	cmd.Flags().StringVarP(&keyword, "keyword", "k", "", "搜索关键词（必填）")
	cmd.Flags().StringVar(&conv, "conv", "", "限定会话 ID（可选）")
	cmd.Flags().IntVar(&limit, "limit", 20, "最多返回条数")
	return cmd
}
