package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newStreamCmd() *cobra.Command {
	var msgID uint64
	var delta string
	var finish bool
	cmd := &cobra.Command{
		Use:   "stream",
		Short: "追加流式分段",
		Run: func(cmd *cobra.Command, args []string) {
			if msgID == 0 {
				die("--message-id 必填")
			}
			if err := streamChunk(msgID, delta, finish); err != nil {
				die("%v", err)
			}
		},
	}
	cmd.Flags().Uint64Var(&msgID, "message-id", 0, "流式消息 ID")
	cmd.Flags().StringVar(&delta, "delta", "", "追加内容")
	cmd.Flags().BoolVar(&finish, "finish", false, "结束流式并转 markdown")
	return cmd
}

func newStreamStdinCmd() *cobra.Command {
	var to, thread string
	cmd := &cobra.Command{
		Use:   "stream-stdin",
		Short: "stdin 逐行喂 delta，EOF finish",
		Long:  "建一条流式消息，stdin 每行作为一个 delta 追加，EOF 时 finish。配合 `claude -p ... | qim stream-stdin ...` 实现流式回复。",
		Run: func(cmd *cobra.Command, args []string) {
			if to == "" {
				die("--to 必填")
			}
			msgID, err := sendMessage(to, thread, "", "streaming", 0)
			if err != nil {
				die("建流式消息失败: %v", err)
			}
			if err := pipeStreamInput(os.Stdin, msgID, streamChunk); err != nil {
				die("%v", err)
			}
			fmt.Fprintln(os.Stderr, "streamed message", msgID)
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "目标用户名或用户 ID（必填）")
	cmd.Flags().StringVar(&thread, "thread", "", "会话名或会话 ID（可选）")
	return cmd
}

func streamChunk(msgID uint64, delta string, finish bool) error {
	cfg := mustConfig()
	body, _ := json.Marshal(map[string]any{
		"content_delta": delta,
		"finish":        finish,
	})
	_, err := httpPost(cfg, fmt.Sprintf("%s/api/v1/bot/messages/%d/stream", cfg.ServerURL, msgID), body)
	return err
}

func pipeStreamInput(r io.Reader, msgID uint64, send func(uint64, string, bool) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if err := send(msgID, line+"\n", false); err != nil {
			return fmt.Errorf("追加 delta 失败: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 stdin 失败: %w", err)
	}
	if err := send(msgID, "", true); err != nil {
		return fmt.Errorf("finish 失败: %w", err)
	}
	return nil
}
