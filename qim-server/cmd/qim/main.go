// Command qim 是 agent/人驱动 QIM 的命令行客户端（pull 底座）。
//
// 用 Bash 调用本 CLI，即可让 Claude Code/OpenCode 等 agent 在 QIM 内收发消息：
//
//	qim config set --server http://localhost:8080 --token qbot_...
//	qim messages list --thread <conv_id>
//	qim send --to <user_id> --thread <conv_id> --type markdown --content "hi"
//	echo "增量" | qim stream-stdin --to <user_id> --thread <conv_id>
//
// 纯 HTTP 客户端，不耦合 server 内部（service/handler），只依赖 REST 契约。
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const configDir = ".qim"

type config struct {
	ServerURL string `json:"server_url"`
	BotToken  string `json:"bot_token"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "config":
		cmdConfig(os.Args[2:])
	case "messages":
		cmdMessages(os.Args[2:])
	case "send":
		cmdSend(os.Args[2:])
	case "stream":
		cmdStream(os.Args[2:])
	case "stream-stdin":
		cmdStreamStdin(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `qim - QIM agent CLI

命令:
  config set --server URL --token T   写配置（~/.qim/config.json）
  config show                          显示配置（token 脱敏）
  messages list --thread ID [--after-id N] [--limit 50]   拉会话消息（JSON lines）
  messages poll --thread ID [--interval 2s]               轮询新消息（JSON lines）
  send --to USER --thread ID --type text|markdown|card --content ...   发消息，输出 message_id
  stream --message-id ID --delta "..." [--finish]         追加流式分段
  stream-stdin --to USER --thread ID                       stdin 逐行喂 delta，EOF finish
`)
}

// ---------- config ----------

func cmdConfig(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim config set|show")
		os.Exit(2)
	}
	switch args[0] {
	case "set":
		fs := flag.NewFlagSet("config set", flag.ExitOnError)
		server := fs.String("server", "", "QIM 服务器地址，如 http://localhost:8080")
		token := fs.String("token", "", "bot 访问令牌 qbot_...")
		_ = fs.Parse(args[1:])
		if *server == "" || *token == "" {
			fmt.Fprintln(os.Stderr, "--server 与 --token 必填")
			os.Exit(2)
		}
		cfg := config{ServerURL: strings.TrimRight(*server, "/"), BotToken: *token}
		if err := saveConfig(cfg); err != nil {
			die("保存配置失败: %v", err)
		}
		fmt.Println("配置已保存到", configPath())
	case "show":
		cfg, err := loadConfig()
		if err != nil {
			die("读取配置失败: %v", err)
		}
		mask := cfg.BotToken
		if len(mask) > 12 {
			mask = mask[:8] + "..." + mask[len(mask)-4:]
		}
		fmt.Printf("server_url: %s\nbot_token:  %s\n", cfg.ServerURL, mask)
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: config "+args[0])
		os.Exit(2)
	}
}

// ---------- messages ----------

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

func cmdMessages(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim messages list|poll")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("messages list", flag.ExitOnError)
		thread := fs.Uint64("thread", 0, "会话 ID（thread_id）")
		afterID := fs.Uint64("after-id", 0, "只返回 id 大于该值的消息")
		limit := fs.Int("limit", 50, "最多返回条数")
		_ = fs.Parse(args[1:])
		if *thread == 0 {
			fmt.Fprintln(os.Stderr, "--thread 必填")
			os.Exit(2)
		}
		msgs, err := fetchMessages(*thread, *afterID, *limit)
		if err != nil {
			die("%v", err)
		}
		emitMessages(msgs)
	case "poll":
		fs := flag.NewFlagSet("messages poll", flag.ExitOnError)
		thread := fs.Uint64("thread", 0, "会话 ID（thread_id）")
		interval := fs.Duration("interval", 2*time.Second, "轮询间隔")
		_ = fs.Parse(args[1:])
		if *thread == 0 {
			fmt.Fprintln(os.Stderr, "--thread 必填")
			os.Exit(2)
		}
		var afterID uint64
		// 首次建立水位线：拉到最新消息的 id（分页向前推进），不输出任何历史。
		// 之前用 limit=1 取首条（ASC 即最老），导致每次启动回放几乎全部历史。
		afterID = latestMessageID(*thread)
		for {
			msgs, err := fetchMessages(*thread, afterID, 100)
			if err != nil {
				fmt.Fprintf(os.Stderr, "poll 错误: %v\n", err)
			}
			for _, m := range msgs {
				emitMessage(m)
				afterID = m.ID
			}
			time.Sleep(*interval)
		}
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: messages "+args[0])
		os.Exit(2)
	}
}

func fetchMessages(thread, afterID uint64, limit int) ([]message, error) {
	cfg := mustConfig()
	u := fmt.Sprintf("%s/api/v1/bot/messages?thread_id=%d&limit=%d", cfg.ServerURL, thread, clamp(limit))
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

// latestMessageID 分页拉到会话最新消息的 id，作为 poll 水位线（不输出历史）。
// 列表接口按 id ASC 返回，故持续用上一批最大 id 作 after_id 向前推进，直到不足一页。
func latestMessageID(thread uint64) uint64 {
	var after, max uint64
	for {
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
}

func emitMessages(msgs []message) {
	for _, m := range msgs {
		emitMessage(m)
	}
}

func emitMessage(m message) {
	b, _ := json.Marshal(m)
	fmt.Println(string(b))
}

// ---------- send ----------

func cmdSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	to := fs.Uint64("to", 0, "目标用户 ID")
	thread := fs.Uint64("thread", 0, "会话 ID（thread_id）")
	msgType := fs.String("type", "text", "消息类型: text|markdown|card")
	content := fs.String("content", "", "消息内容（card 时为 JSON）")
	fs.StringVar(content, "c", "", "（--content 简写）")
	_ = fs.Parse(args)
	if *to == 0 || *thread == 0 || *content == "" {
		fmt.Fprintln(os.Stderr, "--to/--thread/--content 必填")
		os.Exit(2)
	}
	id, err := sendMessage(*to, *thread, *content, *msgType)
	if err != nil {
		die("%v", err)
	}
	fmt.Println(id)
}

func sendMessage(to, thread uint64, content, msgType string) (uint64, error) {
	cfg := mustConfig()
	body, _ := json.Marshal(map[string]any{
		"to_user_id": to,
		"content":    content,
		"msg_type":   msgType,
		"thread_id":  thread,
	})
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

// ---------- stream ----------

func cmdStream(args []string) {
	fs := flag.NewFlagSet("stream", flag.ExitOnError)
	msgID := fs.Uint64("message-id", 0, "流式消息 ID")
	delta := fs.String("delta", "", "追加内容")
	finish := fs.Bool("finish", false, "结束流式并转 markdown")
	_ = fs.Parse(args)
	if *msgID == 0 {
		fmt.Fprintln(os.Stderr, "--message-id 必填")
		os.Exit(2)
	}
	if err := streamChunk(*msgID, *delta, *finish); err != nil {
		die("%v", err)
	}
}

// cmdStreamStdin 建一条流式消息，stdin 每行作为一个 delta 追加，EOF 时 finish。
// 配合 `claude -p ... | qim stream-stdin ...` 实现流式回复。
func cmdStreamStdin(args []string) {
	fs := flag.NewFlagSet("stream-stdin", flag.ExitOnError)
	to := fs.Uint64("to", 0, "目标用户 ID")
	thread := fs.Uint64("thread", 0, "会话 ID（thread_id）")
	_ = fs.Parse(args)
	if *to == 0 || *thread == 0 {
		fmt.Fprintln(os.Stderr, "--to/--thread 必填")
		os.Exit(2)
	}
	msgID, err := sendMessage(*to, *thread, "", "streaming")
	if err != nil {
		die("建流式消息失败: %v", err)
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if err := streamChunk(msgID, line+"\n", false); err != nil {
			die("追加 delta 失败: %v", err)
		}
	}
	if err := streamChunk(msgID, "", true); err != nil {
		die("finish 失败: %v", err)
	}
	fmt.Fprintln(os.Stderr, "streamed message", msgID)
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

// ---------- HTTP ----------

func httpGet(cfg config, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setAuth(req, cfg)
	return do(req)
}

func httpPost(cfg config, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	setAuth(req, cfg)
	return do(req)
}

func setAuth(req *http.Request, cfg config) {
	if cfg.BotToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.BotToken)
	}
}

func do(req *http.Request) ([]byte, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

// ---------- config io ----------

func mustConfig() config {
	cfg, err := loadConfig()
	if err != nil {
		die("读取配置失败（先 qim config set）: %v", err)
	}
	if cfg.ServerURL == "" || cfg.BotToken == "" {
		die("配置不完整，先 qim config set --server URL --token T")
	}
	return cfg
}

func loadConfig() (config, error) {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configPath(), b, 0o600)
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, configDir, "config.json")
	}
	return filepath.Join(configDir, "config.json")
}

// ---------- utils ----------

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func clamp(n int) int {
	if n <= 0 || n > 100 {
		return 50
	}
	return n
}
