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
	var to, thread, conversation string
	cmd := &cobra.Command{
		Use:   "stream-stdin",
		Short: "stdin 逐行喂 delta，EOF finish",
		Long:  fmt.Sprintf("建一条流式消息，stdin 每行作为一个 delta 追加，EOF 时 finish。配合 `claude -p ... | %[1]s stream-stdin ...` 实现流式回复。", CliName),
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("thread") {
				warnThreadDeprecated()
			}
			target := conversation
			strict := true
			if target == "" {
				target = thread
				strict = false
			}
			var msgID uint64
			var err error
			if target != "" {
				if convID, found := lookupBotConversation(target); found {
					msgID, err = sendMessageToConversation(convID, "", "streaming", 0)
				} else if strict {
					die("未找到会话 %q（用 `%s groups` 查看已入群会话，或改用会话 ID）", target, CliName)
				} else if to != "" {
					msgID, err = sendMessage(to, target, "", "streaming", 0)
				} else {
					msgID, err = sendMessage(target, "", "", "streaming", 0)
				}
			} else {
				if to == "" {
					die("--to 必填；或用 --conversation 指定会话")
				}
				msgID, err = sendMessage(to, "", "", "streaming", 0)
			}
			if err != nil {
				die("建流式消息失败: %v", err)
			}
			if err := pipeStreamInput(os.Stdin, msgID, streamChunk); err != nil {
				die("%v", err)
			}
			fmt.Fprintf(os.Stderr, "✅ 流式消息已发送 (ID: %d)\n", msgID)
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "目标用户名或用户 ID（与 --conversation 二选一）")
	cmd.Flags().StringVar(&conversation, "conversation", "", "会话 ID 或会话名")
	cmd.Flags().StringVar(&thread, "thread", "", "已弃用，请改用 --conversation")
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

// streamChunkAbort 丢弃一条流式占位（服务端软删 + message_deleted 广播）。
// 供 agent loop 在智能体判定无需回复/调用失败时回收 typing 气泡。
func streamChunkAbort(msgID uint64) error {
	cfg := mustConfig()
	body, _ := json.Marshal(map[string]any{"abort": true})
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
