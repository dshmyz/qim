package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newMessagesEditCmd() *cobra.Command {
	var msgID uint64
	var content, msgType string
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "更新消息内容（卡片状态回写等）",
		Run: func(cmd *cobra.Command, args []string) {
			if msgID == 0 {
				die("--message-id 必填")
			}
			if content == "" {
				die("--content 必填")
			}
			// 支持 --content - 从 stdin 读取
			if content == "-" {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					die("读取 stdin 失败: %v", err)
				}
				content = string(b)
			}
			if err := editMessage(msgID, content, msgType); err != nil {
				die("%v", err)
			}
			fmt.Println("✅ 消息已更新")
		},
	}
	cmd.Flags().Uint64Var(&msgID, "message-id", 0, "消息 ID（必填）")
	cmd.Flags().StringVarP(&content, "content", "c", "", "新内容（card 时为 JSON；- 表示从 stdin 读取）")
	cmd.Flags().StringVar(&msgType, "type", "", "消息类型（可选，留空保持原类型）")
	return cmd
}

// editMessage 调 PUT /api/v1/bot/messages/:id 更新消息内容。
func editMessage(msgID uint64, content, msgType string) error {
	cfg := mustConfig()
	m := map[string]any{"content": content}
	if msgType != "" {
		m["msg_type"] = msgType
	}
	body, _ := json.Marshal(m)
	_, err := httpPut(cfg, fmt.Sprintf("%s/api/v1/bot/messages/%d", cfg.ServerURL, msgID), body)
	return err
}
