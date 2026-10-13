package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
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
	var asUser bool
	cmd := &cobra.Command{
		Use:   "send",
		Short: "发消息（--thread 可选，自动创建会话）",
		Long: fmt.Sprintf(`发消息。默认以 bot 身份发送（需 --server/--token 配置，消息发送者显示为 bot）；
加 --as-user 则以你本人身份发送，走与桌面客户端相同的用户接口，消息以你的名义
出现在会话里（需 user_token：%s login 的 JWT，或客户端「设置 → 开发者设置」
签发的 qusr_ 长期令牌）。`, CliName),
		Run: func(cmd *cobra.Command, args []string) {
			// --as-user：以本人身份（user_token）走用户 API 发送，消息以当前用户名义出现在会话里。
			// 默认（bot 身份）走 /api/v1/bot/messages，发送者是 bot 虚拟用户。
			if asUser {
				if content == "-" {
					b, err := io.ReadAll(os.Stdin)
					if err != nil {
						die("读取 stdin 失败: %v", err)
					}
					content = string(b)
				}
				if file != "" {
					// sendFile 走 userUpload 上传，本身就是用户身份
					if content == "" {
						content = sendFile(file)
						if msgType == "text" {
							msgType = "markdown"
						}
					} else {
						content += "\n\n" + sendFile(file)
						if msgType == "text" {
							msgType = "markdown"
						}
					}
				}
				if content == "" {
					die("--content 或 --file 必填")
				}
				if thread != "" && conversation == "" {
					conversation = thread // --thread 是 --conversation 的别名
				}
				convID := resolveUserConversationID(conversation, to)
				id, err := sendMessageAsUser(convID, content, msgType, replyTo)
				if err != nil {
					die("%v", err)
				}
				out(map[string]any{"message_id": id}, "✅ 消息已发送 (ID: %d)\n", id)
				return
			}
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
	cmd.Flags().BoolVar(&asUser, "as-user", false, "以本人身份发送（需 user_token：qusr_ 长期令牌或 qim login 的 JWT）。--to 支持用户名/用户 ID（自动建/找单聊），--conversation 支持会话 ID")
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

// resolveUserConversationID 解析用户身份发送的目标会话 ID：
// --conversation 数字直接作会话 ID；否则按 --to（数字=用户 ID，否则 /users/search
// 精确匹配用户名）建/复用单聊拿到会话 ID（POST /conversations type=single 幂等）。
func resolveUserConversationID(conversation, to string) uint64 {
	if conversation != "" {
		if !isNumeric(conversation) {
			die("--as-user 下 --conversation 需为会话 ID（数字）")
		}
		id, err := strconv.ParseUint(conversation, 10, 64)
		if err != nil {
			die("无效的会话 ID: %s", conversation)
		}
		return id
	}
	if to == "" {
		die("--as-user 需要 --to（用户名/用户 ID）或 --conversation（会话 ID）")
	}
	var recipientID uint64
	if isNumeric(to) {
		id, err := strconv.ParseUint(to, 10, 64)
		if err != nil {
			die("无效的用户 ID: %s", to)
		}
		recipientID = id
	} else {
		recipientID = userSearchID(to)
	}

	// 服务端 CreateSingleConversation 只认 user_id（recipient_id 是绑定结构体里的
	// 死别名，不会被映射），字段名必须与服务端契约一致。
	respBody, err := userPost("/api/v1/conversations", map[string]any{
		"type":    "single",
		"user_id": recipientID,
	})
	if err != nil {
		die("创建/查找单聊失败: %v", err)
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil || resp.Data.ID == 0 {
		die("创建/查找单聊失败: %s", truncate(string(respBody), 200))
	}
	return resp.Data.ID
}

// userSearchID 用 /users/search 按关键词精确匹配用户名，返回用户 ID。
func userSearchID(name string) uint64 {
	respBody, err := userGet("/api/v1/users/search?q=" + url.QueryEscape(name) + "&scope=users")
	if err != nil {
		die("搜索用户失败: %v", err)
	}
	var resp struct {
		Data []struct {
			ID       uint64 `json:"id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		die("解析搜索响应失败: %s", truncate(string(respBody), 200))
	}
	for _, u := range resp.Data {
		if u.Username == name {
			return u.ID
		}
	}
	die("未找到用户: %s（请确认用户名，或改用数字用户 ID）", name)
	return 0
}

// sendMessageAsUser 以本人身份（user_token）向会话发消息：POST /api/v1/conversations/:id/messages。
// 返回消息 ID（响应 data.id，BuildMessageResponse 的主键字段）。
func sendMessageAsUser(convID uint64, content, msgType string, replyTo uint64) (uint64, error) {
	body := map[string]any{
		"type":    msgType,
		"content": content,
	}
	if replyTo > 0 {
		body["quoted_message_id"] = replyTo
	}
	respBody, err := userPost(fmt.Sprintf("/api/v1/conversations/%d/messages", convID), body)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析响应失败: %w", err)
	}
	if resp.Data.ID == 0 {
		return 0, fmt.Errorf("响应缺少 message_id: %s", truncate(string(respBody), 200))
	}
	return resp.Data.ID, nil
}
