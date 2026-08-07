package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newGroupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "列出当前 bot 已入群的群会话（用于按 conversation_id 主动群发）",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := mustConfig()
			respBody, err := httpGet(cfg, cfg.ServerURL+"/api/v1/bot/groups")
			if err != nil {
				die("%v", err)
			}
			if outputFmt == "json" {
				fmt.Println(string(respBody))
				return
			}
			var resp struct {
				Data []struct {
					ConversationID uint64 `json:"conversation_id"`
					GroupName      string `json:"group_name"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBody, &resp); err != nil {
				die("解析失败: %v", err)
			}
			if len(resp.Data) == 0 {
				fmt.Println("（当前 bot 尚未被拉入任何群）")
				return
			}
			for _, g := range resp.Data {
				fmt.Printf("#%-6d %s\n", g.ConversationID, g.GroupName)
			}
		},
	}
	return cmd
}
