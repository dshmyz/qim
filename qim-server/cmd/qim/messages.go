package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// message 是消息列表接口返回的结构体。
type message struct {
	ID             uint64 `json:"id"`
	ConversationID uint64 `json:"conversation_id"`
	SenderID       uint64 `json:"sender_id"`
	SenderType     string `json:"sender_type"`
	SenderNickname string `json:"sender_nickname"`
	Content        string `json:"content"`
	Type           string `json:"type"`
	Origin         string `json:"origin"`
	CreatedAt      string `json:"created_at"`
}

func newMessagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messages",
		Short: "消息操作",
	}
	cmd.AddCommand(newMessagesListCmd(), newMessagesPollCmd(), newMessagesActionsCmd(), newMessagesEditCmd(), newMessagesSearchCmd())
	return cmd
}

func newMessagesListCmd() *cobra.Command {
	var thread string
	var afterID uint64
	var limit int
	var msgType string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "拉会话消息（JSON lines）",
		Run: func(cmd *cobra.Command, args []string) {
			if thread == "" {
				die("--thread 必填（会话名或 ID）")
			}
			msgs, err := fetchMessages(thread, afterID, limit)
			if err != nil {
				die("%v", err)
			}
			msgs = filterByType(msgs, msgType)
			emitMessages(msgs)
		},
	}
	cmd.Flags().StringVar(&thread, "thread", "", "会话名或会话 ID（必填）")
	cmd.Flags().Uint64Var(&afterID, "after-id", 0, "只返回 id 大于该值的消息")
	cmd.Flags().IntVar(&limit, "limit", 50, "最多返回条数")
	cmd.Flags().StringVar(&msgType, "type", "", "按消息类型过滤: text|markdown|card|card_action|streaming")
	return cmd
}

func newMessagesPollCmd() *cobra.Command {
	var thread string
	var interval time.Duration
	var msgType string
	var once bool
	cmd := &cobra.Command{
		Use:   "poll",
		Short: "轮询新消息（JSON lines）",
		Long: `轮询新消息，输出 JSON lines。
默认持续运行直到 SIGINT/SIGTERM。

示例:
  # 持续监听用户输入
  qim messages poll --thread alice

  # 只等一次卡片点击结果就退出
  qim messages poll --thread alice --type card_action --once

  # 等待任何新消息，一轮就退出
  qim messages poll --thread alice --once`,
		Run: func(cmd *cobra.Command, args []string) {
			if thread == "" {
				die("--thread 必填（会话名或 ID）")
			}
			// 监听 SIGINT/SIGTERM 优雅退出
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			var afterID uint64
			afterID = latestMessageID(thread)
			for {
				msgs, err := fetchMessages(thread, afterID, 100)
				if err != nil {
					fmt.Fprintf(os.Stderr, "poll 错误: %v\n", err)
				}
				filtered, nextAfterID := filterMessagesAndAdvance(msgs, msgType, afterID)
				for _, m := range filtered {
					emitMessage(m)
				}
				afterID = nextAfterID
				if once && len(filtered) > 0 {
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(interval):
				}
			}
		},
	}
	cmd.Flags().StringVar(&thread, "thread", "", "会话名或会话 ID（必填）")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "轮询间隔")
	cmd.Flags().StringVar(&msgType, "type", "", "按消息类型过滤: text|markdown|card|card_action|streaming")
	cmd.Flags().BoolVar(&once, "once", false, "收到一批新消息后退出（适合 agent 一次性等待）")
	return cmd
}

// newMessagesActionsCmd 是 "messages list --type card_action" 的语义别名。
func newMessagesActionsCmd() *cobra.Command {
	var thread string
	var afterID uint64
	var limit int
	cmd := &cobra.Command{
		Use:   "actions",
		Short: "查看卡片点击事件（card_action 消息）",
		Long: `查看卡片按钮点击事件。

等价于 messages list --type card_action。
agent 发送交互卡片后，用 poll --type card_action --once 等待用户点击结果。

示例:
  # 查看历史卡片点击
  qim messages actions --thread alice

  # 等待下一次卡片点击
  qim messages poll --thread alice --type card_action --once`,
		Run: func(cmd *cobra.Command, args []string) {
			if thread == "" {
				die("--thread 必填（会话名或 ID）")
			}
			msgs, err := fetchMessages(thread, afterID, limit)
			if err != nil {
				die("%v", err)
			}
			msgs = filterByType(msgs, "card_action")
			emitMessages(msgs)
		},
	}
	cmd.Flags().StringVar(&thread, "thread", "", "会话名或会话 ID（必填）")
	cmd.Flags().Uint64Var(&afterID, "after-id", 0, "只返回 id 大于该值的消息")
	cmd.Flags().IntVar(&limit, "limit", 50, "最多返回条数")
	return cmd
}

// fetchMessages 拉取会话消息。thread 可以是数字 ID 或会话名/用户名。
// 数字传 thread_id，非数字传 thread_name（服务端自动解析）。
func fetchMessages(thread string, afterID uint64, limit int) ([]message, error) {
	cfg := mustConfig()
	u := fmt.Sprintf("%s/api/v1/bot/messages?limit=%d", cfg.ServerURL, clamp(limit))
	if isNumeric(thread) {
		u += "&thread_id=" + thread
	} else {
		u += "&thread_name=" + url.QueryEscape(thread)
	}
	if afterID > 0 {
		u += fmt.Sprintf("&after_id=%d", afterID)
	}
	body, err := httpGet(cfg, u)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Messages []message `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return resp.Data.Messages, nil
}

// filterByType 按消息类型过滤。typeStr 为空时不过滤。
func filterByType(msgs []message, typeStr string) []message {
	if typeStr == "" {
		return msgs
	}
	var filtered []message
	for _, m := range msgs {
		if m.Type == typeStr {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func filterMessagesAndAdvance(msgs []message, typeStr string, afterID uint64) ([]message, uint64) {
	filtered := filterByType(msgs, typeStr)
	for _, m := range msgs {
		if m.ID > afterID {
			afterID = m.ID
		}
	}
	return filtered, afterID
}

// latestMessageID 分页拉到会话最新消息的 id，作为 poll 水位线（不输出历史）。
// 最多翻 50 页（5000 条消息），避免会话消息过多时卡死。
func latestMessageID(thread string) uint64 {
	var after, max uint64
	for pages := 0; pages < 50; pages++ {
		msgs, err := fetchMessages(thread, after, 100)
		if err != nil {
			return max
		}
		if len(msgs) == 0 {
			return max
		}
		for _, m := range msgs {
			if m.ID > max {
				max = m.ID
			}
			after = m.ID
		}
		if len(msgs) < 100 {
			return max
		}
	}
	return max
}

func emitMessages(msgs []message) {
	for _, m := range msgs {
		emitMessage(m)
	}
}

func emitMessage(m message) {
	if outputFmt == "id" {
		fmt.Println(m.ID)
		return
	}
	b, _ := json.Marshal(m)
	fmt.Println(string(b))
}

// isNumeric 判断字符串是否全数字。
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
