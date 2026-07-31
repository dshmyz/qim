package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newConversationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conversations",
		Short: "管理会话",
	}
	cmd.AddCommand(newConversationsListCmd())
	return cmd
}

func newConversationsListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出最近会话",
		Run: func(cmd *cobra.Command, args []string) {
			respBody, err := userGet("/api/v1/conversations?page_size=" + fmt.Sprint(limit))
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data struct {
					List json.RawMessage `json:"list"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			var convs []struct {
				ID              uint64 `json:"id"`
				Name            string `json:"name"`
				Type            string `json:"type"`
				UnreadCount     int    `json:"unread_count"`
				OtherMemberName string `json:"other_member_name"`
				LastMessage     *struct {
					Content string `json:"content"`
				} `json:"last_message"`
			}
			if err := json.Unmarshal(resp.Data.List, &convs); err != nil {
				die("解析会话列表失败: %v", err)
			}
			if len(convs) == 0 {
				fmt.Println("（无会话）")
				return
			}
			if len(convs) > limit {
				convs = convs[:limit]
			}
			for _, c := range convs {
				name := c.Name
				if name == "" {
					name = c.OtherMemberName
				}
				unread := ""
				if c.UnreadCount > 0 {
					unread = fmt.Sprintf(" [%d未读]", c.UnreadCount)
				}
				lastMsg := ""
				if c.LastMessage != nil && c.LastMessage.Content != "" {
					content := c.LastMessage.Content
					if len([]rune(content)) > 40 {
						content = string([]rune(content)[:40]) + "..."
					}
					lastMsg = " — " + content
				}
				fmt.Printf("#%-6d (%s) %s%s%s\n", c.ID, c.Type, name, unread, lastMsg)
			}
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "最多返回条数")
	return cmd
}
