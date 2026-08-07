package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	var to, thread, msgType, content string
	var conversation string
	var file string
	var replyTo uint64
	cmd := &cobra.Command{
		Use:   "send",
		Short: "发消息（--thread 可选，自动创建会话）",
		Run: func(cmd *cobra.Command, args []string) {
			// 按会话发送（群聊/已建单聊）：--conversation <id> 时，--to 可省略
			if conversation != "" && to == "" {
				if !isNumeric(conversation) {
					die("--conversation 需为会话 ID（数字）")
				}
				convID, err := strconv.ParseUint(conversation, 10, 64)
				if err != nil {
					die("无效的会话 ID: %s", conversation)
				}
				// --content - 从 stdin 读取
				if content == "-" {
					b, err := io.ReadAll(os.Stdin)
					if err != nil {
						die("读取 stdin 失败: %v", err)
					}
					content = string(b)
				}
				if content == "" {
					die("--content 或 --file 必填")
				}
				id, err := sendMessageToConversation(convID, content, msgType, replyTo)
				if err != nil {
					die("%v", err)
				}
				out(map[string]any{"message_id": id}, "✅ 消息已发送 (ID: %d)\n", id)
				return
			}
			if to == "" {
				die("--to 必填")
			}
			// --conversation 是 --thread 的别名
			if conversation != "" && thread == "" {
				thread = conversation
			}
			// --content - 从 stdin 读取
			if content == "-" {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					die("读取 stdin 失败: %v", err)
				}
				content = string(b)
			}
			// --file 模式：上传文件并以 markdown 链接发送
			if file != "" {
				if content == "" {
					content = sendFile(file)
					if msgType == "text" {
						msgType = "markdown"
					}
				} else {
					// --file 和 --content 同时指定：内容追加文件链接
					content += "\n\n" + sendFile(file)
					if msgType == "text" {
						msgType = "markdown"
					}
				}
			}
			if content == "" {
				die("--content 或 --file 必填")
			}
			id, err := sendMessage(to, thread, content, msgType, replyTo)
			if err != nil {
				die("%v", err)
			}
			out(map[string]any{"message_id": id}, "✅ 消息已发送 (ID: %d)\n", id)
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "目标用户名或用户 ID（必填，除非用 --conversation 按群会话发送）")
	cmd.Flags().StringVar(&thread, "thread", "", "会话名或会话 ID（可选，不填自动创建/查找）")
	cmd.Flags().StringVar(&conversation, "conversation", "", "群会话/已建单聊会话 ID（传数字时按会话发送，可省略 --to）")
	cmd.Flags().Uint64Var(&replyTo, "reply-to", 0, "回复的消息 ID（可选）")
	cmd.Flags().StringVar(&msgType, "type", "text", "消息类型: text|markdown|card")
	cmd.Flags().StringVarP(&content, "content", "c", "", "消息内容（card 时为 JSON；- 表示从 stdin 读取）")
	cmd.Flags().StringVar(&file, "file", "", "上传文件并以 markdown 链接发送（需 user_token）")
	return cmd
}

// sendFile 以用户身份上传文件，返回 markdown 下载链接。
func sendFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		die("打开文件失败: %v", err)
	}
	defer f.Close()
	filename := filepath.Base(path)
	respBody, err := userUpload("/api/v1/upload", filename, f)
	if err != nil {
		die("上传文件失败: %v", err)
	}
	var resp struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil || resp.Data.URL == "" {
		die("解析上传响应失败: %s", string(respBody))
	}
	return fmt.Sprintf("📎 [%s](%s)", filename, resp.Data.URL)
}

// sendMessage 发消息。to 和 thread 均可为用户名/会话名或数字 ID。
// 数字传 to_user_id/thread_id，非数字传 to_user_name/thread_name（服务端自动解析）。
func sendMessage(to, thread, content, msgType string, replyTo uint64) (uint64, error) {
	cfg := mustConfig()
	m := map[string]any{
		"content":  content,
		"msg_type": msgType,
	}
	if isNumeric(to) {
		id, err := strconv.ParseUint(to, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("无效的用户 ID: %s", to)
		}
		m["to_user_id"] = id
	} else {
		m["to_user_name"] = to
	}
	if thread != "" {
		if isNumeric(thread) {
			id, err := strconv.ParseUint(thread, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("无效的会话 ID: %s", thread)
			}
			m["thread_id"] = id
		} else {
			m["thread_name"] = thread
		}
	}
	if replyTo > 0 {
		m["reply_to_id"] = replyTo
	}
	body, _ := json.Marshal(m)
	respBody, err := httpPost(cfg, cfg.ServerURL+"/api/v1/bot/messages", body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			MessageID uint64 `json:"message_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析响应失败: %w", err)
	}
	if resp.Data.MessageID == 0 {
		return 0, fmt.Errorf("响应缺少 message_id: %s", string(respBody))
	}
	return resp.Data.MessageID, nil
}

// sendMessageToConversation 按会话(群聊/已建单聊)发送消息。
func sendMessageToConversation(convID uint64, content, msgType string, replyTo uint64) (uint64, error) {
	cfg := mustConfig()
	m := map[string]any{
		"conversation_id": convID,
		"content":         content,
		"msg_type":        msgType,
	}
	if replyTo > 0 {
		m["reply_to_id"] = replyTo
	}
	body, _ := json.Marshal(m)
	respBody, err := httpPost(cfg, cfg.ServerURL+"/api/v1/bot/messages", body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			MessageID uint64 `json:"message_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析响应失败: %w", err)
	}
	if resp.Data.MessageID == 0 {
		return 0, fmt.Errorf("响应缺少 message_id: %s", string(respBody))
	}
	return resp.Data.MessageID, nil
}
