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
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const configDir = ".qim"

// outputFmt 控制输出格式："" (默认人类可读) 或 "json" (原始 JSON)。
var outputFmt string

type config struct {
	ServerURL    string `json:"server_url"`
	BotToken     string `json:"bot_token"`
	UserToken    string `json:"user_token"`     // 用户 JWT，用于以用户身份调 /api/v1/*（任务/日历等）
	RefreshToken string `json:"refresh_token"`  // 用于自动续期 user_token
}

func main() {
	args := os.Args[1:]
	args, outputFmt = extractOutputFlag(args)
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	switch args[0] {
	case "config":
		cmdConfig(args[1:])
	case "login":
		cmdLogin(args[1:])
	case "conversations":
		cmdConversations(args[1:])
	case "messages":
		cmdMessages(args[1:])
	case "send":
		cmdSend(args[1:])
	case "stream":
		cmdStream(args[1:])
	case "stream-stdin":
		cmdStreamStdin(args[1:])
	case "task":
		cmdTask(args[1:])
	case "event":
		cmdEvent(args[1:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", args[0])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `qim - QIM agent CLI

全局选项:
  --output json   输出原始 JSON（默认人类可读）

命令:
  config set --server URL --token T   写配置（~/.qim/config.json）
  config show                          显示配置（token 脱敏）
  login [-u username]                  交互式登录获取 user_token（自动续期）
  conversations list [--limit 50]      列出最近会话
  messages list --thread ID [--after-id N] [--limit 50]   拉会话消息（JSON lines）
  messages poll --thread ID [--interval 2s]               轮询新消息（JSON lines）
  send --to USER [--thread CONV] [--reply-to MSG_ID] --type text|markdown|card --content ...   发消息（--thread 可选，自动创建会话）
  stream --message-id ID --delta "..." [--finish]         追加流式分段
  stream-stdin --to USER --thread ID                       stdin 逐行喂 delta，EOF finish
  task list [--status todo|doing|done] [--limit 50]        列出待办
  task create --title "..." [--due 2026-08-01] [--priority low|medium|high] [--desc "..."]   建待办
  task update ID [--status done] [--priority high] [--title "..."] [--due 2026-08-01]        改待办
  event list [--limit 50]                                   列出日历事件
  event create --title "..." --start "2026-08-01 14:00" --end "2026-08-01 15:00" [--reminder 15] [--desc "..."]
  event update ID [--title "..."] [--reminder 30]           改事件
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
		userToken := fs.String("user-token", "", "用户 JWT，用于以用户身份建任务/日历（可选）")
		_ = fs.Parse(args[1:])
		// --user-token 单独 set 时复用已有 server/bot_token
		if *server == "" || *token == "" {
			if *userToken == "" {
				fmt.Fprintln(os.Stderr, "--server 与 --token 必填（或先配置后再用 --user-token 追加）")
				os.Exit(2)
			}
			// 只追加 user-token：读取现有配置补充
			old, err := loadConfig()
			if err != nil {
				die("读取旧配置失败: %v", err)
			}
			old.UserToken = *userToken
			if err := saveConfig(old); err != nil {
				die("保存配置失败: %v", err)
			}
			fmt.Println("user_token 已保存到", configPath())
			return
		}
		cfg := config{
			ServerURL: strings.TrimRight(*server, "/"),
			BotToken:  *token,
			UserToken: *userToken,
		}
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
		utMask := cfg.UserToken
		if len(utMask) > 20 {
			utMask = utMask[:8] + "..." + utMask[len(utMask)-4:]
		}
		fmt.Printf("server_url:  %s\nbot_token:   %s\nuser_token:  %s\n", cfg.ServerURL, mask, utMask)
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: config "+args[0])
		os.Exit(2)
	}
}

// ---------- login ----------

func cmdLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	username := fs.String("u", "", "用户名（可选，不填会交互提示）")
	_ = fs.Parse(args)

	// 需要先有 server 配置
	cfg, err := loadConfig()
	if err != nil || cfg.ServerURL == "" {
		die("请先执行: qim config set --server URL --token BOT_TOKEN")
	}

	if *username == "" {
		fmt.Print("用户名: ")
		*username, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		*username = strings.TrimSpace(*username)
	}
	if *username == "" {
		die("用户名不能为空")
	}

	fmt.Print("密码: ")
	password, err := readPassword()
	fmt.Println()
	if err != nil {
		die("读取密码失败: %v", err)
	}

	// 调登录接口
	body, _ := json.Marshal(map[string]string{
		"username": *username,
		"password": password,
	})
	req, _ := http.NewRequest(http.MethodPost, cfg.ServerURL+"/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	respBody, err := do(req)
	if err != nil {
		die("登录失败: %v", err)
	}

	var loginResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
			User         struct {
				Nickname string `json:"nickname"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &loginResp); err != nil || loginResp.Code != 0 {
		die("登录失败: %s", loginResp.Message)
	}

	cfg.UserToken = loginResp.Data.Token
	cfg.RefreshToken = loginResp.Data.RefreshToken
	if err := saveConfig(cfg); err != nil {
		die("保存配置失败: %v", err)
	}
	nick := loginResp.Data.User.Nickname
	if nick == "" {
		nick = loginResp.Data.User.Username
	}
	fmt.Printf("✅ 登录成功：%s（token 7 天有效，自动续期）\n", nick)
}

// readPassword 从 stdin 读取密码（关闭终端回显）。
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		// 非终端回退为普通读取
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return strings.TrimSpace(scanner.Text()), nil
		}
		return "", scanner.Err()
	}
	old := *termios
	newState := old
	newState.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &newState); err != nil {
		return "", err
	}
	defer unix.IoctlSetTermios(fd, unix.TIOCSETA, &old)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", scanner.Err()
}

// jwtExpired 解析 JWT payload 检查是否过期（预留 60s 余量）。
func jwtExpired(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return true // 格式错误视为过期
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return true
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return true
	}
	return time.Now().Unix() >= claims.Exp-60
}

// ensureUserToken 确保 user_token 有效：过期则自动用 refresh_token 续期。
// 失败返回 error（需要重新 qim login）。
func ensureUserToken(cfg *config) error {
	if cfg.UserToken == "" {
		return fmt.Errorf("未登录，请先执行: qim login")
	}
	if !jwtExpired(cfg.UserToken) {
		return nil // 还没过期
	}
	if cfg.RefreshToken == "" {
		return fmt.Errorf("token 已过期且无 refresh_token，请重新登录: qim login")
	}
	if jwtExpired(cfg.RefreshToken) {
		return fmt.Errorf("refresh_token 也已过期，请重新登录: qim login")
	}

	// 自动续期
	body, _ := json.Marshal(map[string]string{"refresh_token": cfg.RefreshToken})
	req, _ := http.NewRequest(http.MethodPost, cfg.ServerURL+"/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+cfg.RefreshToken)

	respBody, err := do(req)
	if err != nil {
		return fmt.Errorf("刷新 token 失败: %w（请重新登录: qim login）", err)
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil || resp.Code != 0 {
		return fmt.Errorf("刷新 token 失败，请重新登录: qim login")
	}

	cfg.UserToken = resp.Data.Token
	cfg.RefreshToken = resp.Data.RefreshToken
	if err := saveConfig(*cfg); err != nil {
		return fmt.Errorf("保存新 token 失败: %w", err)
	}
	fmt.Fprintln(os.Stderr, "🔄 token 已自动续期")
	return nil
}

// ---------- conversations ----------

func cmdConversations(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim conversations list")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		listConversationsCmd(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: conversations "+args[0])
		os.Exit(2)
	}
}

func listConversationsCmd(args []string) {
	fs := flag.NewFlagSet("conversations list", flag.ExitOnError)
	limit := fs.Int("limit", 50, "最多返回条数")
	_ = fs.Parse(args)

	respBody, err := userGet("/api/v1/conversations?limit=" + fmt.Sprint(*limit))
	if err != nil {
		die("%v", err)
	}
	if outputFmt == "json" {
		fmt.Println(string(respBody))
		return
	}
	// API 返回 {data: {has_more: bool, list: [...]}}
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
	if len(convs) > *limit {
		convs = convs[:*limit]
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
	to := fs.String("to", "", "目标用户名或用户 ID（必填）")
	thread := fs.String("thread", "", "会话名或会话 ID（可选，不填自动创建/查找）")
	fs.String("conversation", "", "（--thread 别名）")
	replyTo := fs.Uint64("reply-to", 0, "回复的消息 ID（可选）")
	msgType := fs.String("type", "text", "消息类型: text|markdown|card")
	content := fs.String("content", "", "消息内容（card 时为 JSON）")
	fs.StringVar(content, "c", "", "（--content 简写）")
	_ = fs.Parse(args)
	if *to == "" || *content == "" {
		fmt.Fprintln(os.Stderr, "--to/--content 必填")
		os.Exit(2)
	}
	// --conversation 是 --thread 的别名
	if conv := fs.Lookup("conversation"); conv != nil && conv.Value.String() != "" && *thread == "" {
		*thread = conv.Value.String()
	}
	toID, err := resolveUser(*to)
	if err != nil {
		die("解析 --to 失败: %v", err)
	}
	var threadID uint64
	if *thread != "" {
		threadID, err = resolveConversation(*thread)
		if err != nil {
			die("解析 --thread 失败: %v", err)
		}
	}
	id, err := sendMessage(toID, threadID, *content, *msgType, *replyTo)
	if err != nil {
		die("%v", err)
	}
	out(map[string]any{"message_id": id}, "✅ 消息已发送 (ID: %d)\n", id)
}

func sendMessage(to, thread uint64, content, msgType string, replyTo uint64) (uint64, error) {
	cfg := mustConfig()
	m := map[string]any{
		"to_user_id": to,
		"content":    content,
		"msg_type":   msgType,
		"thread_id":  thread,
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

// ---------- task ----------

func cmdTask(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim task list|create|update")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		listTaskCmd(args[1:])
	case "create":
		createTaskCmd(args[1:])
	case "update":
		updateTaskCmd(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: task "+args[0])
		os.Exit(2)
	}
}

func listTaskCmd(args []string) {
	fs := flag.NewFlagSet("task list", flag.ExitOnError)
	status := fs.String("status", "", "筛选状态: todo|doing|done")
	limit := fs.Int("limit", 50, "最多返回条数")
	_ = fs.Parse(args)

	respBody, err := userGet("/api/v1/tasks")
	if err != nil {
		die("%v", err)
	}
	if outputFmt == "json" {
		fmt.Println(string(respBody))
		return
	}
	// API 返回 {data: [...]}
	var resp struct {
		Data []struct {
			ID       uint64  `json:"id"`
			Title    string  `json:"title"`
			Status   string  `json:"status"`
			Priority string  `json:"priority"`
			DueDate  *string `json:"due_date"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		die("解析失败: %v", err)
	}
	tasks := resp.Data
	// 客户端按 status 过滤（服务端不支持 query param）
	if *status != "" {
		var filtered []struct {
			ID       uint64  `json:"id"`
			Title    string  `json:"title"`
			Status   string  `json:"status"`
			Priority string  `json:"priority"`
			DueDate  *string `json:"due_date"`
		}
		for _, t := range tasks {
			if t.Status == *status {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	if len(tasks) > *limit {
		tasks = tasks[:*limit]
	}
	if len(tasks) == 0 {
		fmt.Println("（无待办）")
		return
	}
	for _, t := range tasks {
		due := "-"
		if t.DueDate != nil && *t.DueDate != "" {
			due = (*t.DueDate)[:10]
		}
		fmt.Printf("#%-4d [%s] %s  (priority=%s, due=%s)\n", t.ID, t.Status, t.Title, t.Priority, due)
	}
}

func createTaskCmd(args []string) {
	fs := flag.NewFlagSet("task create", flag.ExitOnError)
	title := fs.String("title", "", "待办标题（必填）")
	due := fs.String("due", "", "截止日期 YYYY-MM-DD（可选）")
	priority := fs.String("priority", "medium", "优先级: low|medium|high")
	desc := fs.String("desc", "", "描述（可选）")
	_ = fs.Parse(args)
	if *title == "" {
		fmt.Fprintln(os.Stderr, "--title 必填")
		os.Exit(2)
	}

	body := map[string]any{"title": *title, "priority": *priority}
	if *due != "" {
		body["due_date"] = *due
	}
	if *desc != "" {
		body["description"] = *desc
	}
	respBody, err := userPost("/api/v1/tasks", body)
	if err != nil {
		die("%v", err)
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respBody, &resp)
	if resp.Data.ID == 0 {
		fmt.Fprintln(os.Stderr, "创建失败:", string(respBody))
		os.Exit(1)
	}
	outRaw(respBody, "✅ 待办已创建 (ID: %d)\n", resp.Data.ID)
}

func updateTaskCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim task update ID [--status done] [--priority high] [--title ...] [--due ...]")
		os.Exit(2)
	}
	id := args[0]
	fs := flag.NewFlagSet("task update", flag.ExitOnError)
	status := fs.String("status", "", "新状态: todo|doing|done")
	priority := fs.String("priority", "", "新优先级: low|medium|high")
	title := fs.String("title", "", "新标题")
	due := fs.String("due", "", "新截止日期 YYYY-MM-DD")
	_ = fs.Parse(args[1:])

	body := map[string]any{}
	if *status != "" {
		body["status"] = *status
	}
	if *priority != "" {
		body["priority"] = *priority
	}
	if *title != "" {
		body["title"] = *title
	}
	if *due != "" {
		body["due_date"] = *due
	}
	if len(body) == 0 {
		die("至少指定一个要修改的字段")
	}
	respBody, err := userPut("/api/v1/tasks/"+id, body)
	if err != nil {
		die("%v", err)
	}
	outRaw(respBody, "✅ 待办 #%s 已更新\n", id)
}

// ---------- event ----------

func cmdEvent(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim event list|create|update")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		listEventCmd(args[1:])
	case "create":
		createEventCmd(args[1:])
	case "update":
		updateEventCmd(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: event "+args[0])
		os.Exit(2)
	}
}

func listEventCmd(args []string) {
	fs := flag.NewFlagSet("event list", flag.ExitOnError)
	limit := fs.Int("limit", 50, "最多返回条数")
	_ = fs.Parse(args)

	respBody, err := userGet("/api/v1/events")
	if err != nil {
		die("%v", err)
	}
	if outputFmt == "json" {
		fmt.Println(string(respBody))
		return
	}
	var resp struct {
		Data []struct {
			ID       uint64 `json:"id"`
			Title    string `json:"title"`
			Start    string `json:"start"`
			End      string `json:"end"`
			Reminder int    `json:"reminder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		die("解析失败: %v", err)
	}
	events := resp.Data
	if len(events) > *limit {
		events = events[:*limit]
	}
	if len(events) == 0 {
		fmt.Println("（无事件）")
		return
	}
	for _, e := range events {
		remind := ""
		if e.Reminder > 0 {
			remind = fmt.Sprintf(" (提醒提前%d分钟)", e.Reminder)
		}
		start := e.Start
		if len(start) > 16 {
			start = start[:16]
		}
		end := e.End
		if len(end) > 16 {
			end = end[:16]
		}
		fmt.Printf("#%-4d %s  %s ~ %s%s\n", e.ID, e.Title, start, end, remind)
	}
}

func createEventCmd(args []string) {
	fs := flag.NewFlagSet("event create", flag.ExitOnError)
	title := fs.String("title", "", "事件标题（必填）")
	start := fs.String("start", "", "开始时间，如 \"2026-08-01 14:00\"（必填，本地时间）")
	end := fs.String("end", "", "结束时间，如 \"2026-08-01 15:00\"（必填，本地时间）")
	reminder := fs.Int("reminder", 0, "提前提醒分钟数（0=不提醒）")
	desc := fs.String("desc", "", "描述（可选）")
	_ = fs.Parse(args)
	if *title == "" || *start == "" || *end == "" {
		fmt.Fprintln(os.Stderr, "--title/--start/--end 必填")
		os.Exit(2)
	}
	// API 要求 RFC3339 time.Time，本地时间字符串转 RFC3339（带本地时区偏移）
	startRFC, err := localToRFC3339(*start)
	if err != nil {
		die("--start 格式错误: %v（示例: \"2026-08-01 14:00\"）", err)
	}
	endRFC, err := localToRFC3339(*end)
	if err != nil {
		die("--end 格式错误: %v", err)
	}

	body := map[string]any{
		"title":    *title,
		"start":    startRFC,
		"end":      endRFC,
		"reminder": *reminder,
	}
	if *desc != "" {
		body["description"] = *desc
	}
	respBody, err := userPost("/api/v1/events", body)
	if err != nil {
		die("%v", err)
	}
	var resp struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(respBody, &resp)
	if resp.Data.ID == 0 {
		fmt.Fprintln(os.Stderr, "创建失败:", string(respBody))
		os.Exit(1)
	}
	outRaw(respBody, "✅ 事件已创建 (ID: %d)\n", resp.Data.ID)
}

func updateEventCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim event update ID [--title ...] [--reminder 30]")
		os.Exit(2)
	}
	id := args[0]
	fs := flag.NewFlagSet("event update", flag.ExitOnError)
	title := fs.String("title", "", "新标题")
	start := fs.String("start", "", "新开始时间")
	end := fs.String("end", "", "新结束时间")
	reminder := fs.Int("reminder", -1, "新提醒分钟数（-1=不改）")
	desc := fs.String("desc", "", "新描述")
	_ = fs.Parse(args[1:])

	// API 要求 title/start/end 全部必填，先拉取现有事件合并
	currentBody, err := userGet("/api/v1/events/" + id)
	if err != nil {
		die("获取事件失败: %v", err)
	}
	var curResp struct {
		Data struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Start       string `json:"start"`
			End         string `json:"end"`
			Reminder    int    `json:"reminder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(currentBody, &curResp); err != nil {
		die("解析事件失败: %v", err)
	}

	ev := curResp.Data
	body := map[string]any{
		"title":    ev.Title,
		"start":    ev.Start,
		"end":      ev.End,
		"reminder": ev.Reminder,
	}
	if ev.Description != "" {
		body["description"] = ev.Description
	}
	// 覆盖用户指定的字段
	if *title != "" {
		body["title"] = *title
	}
	if *start != "" {
		s, err := localToRFC3339(*start)
		if err != nil {
			die("--start 格式错误: %v", err)
		}
		body["start"] = s
	}
	if *end != "" {
		e, err := localToRFC3339(*end)
		if err != nil {
			die("--end 格式错误: %v", err)
		}
		body["end"] = e
	}
	if *reminder >= 0 {
		body["reminder"] = *reminder
	}
	if *desc != "" {
		body["description"] = *desc
	}
	respBody, err := userPut("/api/v1/events/"+id, body)
	if err != nil {
		die("%v", err)
	}
	outRaw(respBody, "✅ 事件 #%s 已更新\n", id)
}

// localToRFC3339 把 "2006-01-02 15:04" 本地时间字符串转为 RFC3339（带本地时区偏移）。
func localToRFC3339(s string) (string, error) {
	layouts := []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02T15:04", time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("无法解析 %q", s)
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
	to := fs.String("to", "", "目标用户名或用户 ID（必填）")
	thread := fs.String("thread", "", "会话名或会话 ID（可选）")
	_ = fs.Parse(args)
	if *to == "" {
		fmt.Fprintln(os.Stderr, "--to 必填")
		os.Exit(2)
	}
	toID, err := resolveUser(*to)
	if err != nil {
		die("解析 --to 失败: %v", err)
	}
	var threadID uint64
	if *thread != "" {
		threadID, err = resolveConversation(*thread)
		if err != nil {
			die("解析 --thread 失败: %v", err)
		}
	}
	msgID, err := sendMessage(toID, threadID, "", "streaming", 0)
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

// ---------- resolve ----------

// resolveUser 把用户名或数字 ID 解析为用户 ID。
// 数字直接返回，否则调 users/search?q= 按昵称/用户名搜索。
func resolveUser(nameOrID string) (uint64, error) {
	var id uint64
	if _, err := fmt.Sscanf(nameOrID, "%d", &id); err == nil && id > 0 {
		return id, nil
	}
	respBody, err := userGet("/api/v1/users/search?q=" + nameOrID)
	if err != nil {
		return 0, fmt.Errorf("搜索用户 %q 失败: %w", nameOrID, err)
	}
	var resp struct {
		Data []struct {
			ID   uint64 `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析搜索结果失败: %w", err)
	}
	// 精确匹配优先
	for _, u := range resp.Data {
		if u.Name == nameOrID {
			return u.ID, nil
		}
	}
	// 模糊匹配第一个
	if len(resp.Data) > 0 {
		return resp.Data[0].ID, nil
	}
	// 搜索结果为空：可能是搜索自己（API 排除当前用户）
	me, err := userGet("/api/v1/users/me")
	if err == nil {
		var meResp struct {
			Data struct {
				ID       uint64 `json:"id"`
				Username string `json:"username"`
				Nickname string `json:"nickname"`
			} `json:"data"`
		}
		if json.Unmarshal(me, &meResp) == nil {
			if meResp.Data.Username == nameOrID || meResp.Data.Nickname == nameOrID {
				return meResp.Data.ID, nil
			}
		}
	}
	return 0, fmt.Errorf("未找到用户 %q", nameOrID)
}

// resolveConversation 把会话名或数字 ID 解析为会话 ID。
// 数字直接返回，否则调 conversations/search?query= 按名字搜索。
func resolveConversation(nameOrID string) (uint64, error) {
	var id uint64
	if _, err := fmt.Sscanf(nameOrID, "%d", &id); err == nil && id > 0 {
		return id, nil
	}
	respBody, err := userGet("/api/v1/conversations/search?query=" + nameOrID)
	if err != nil {
		return 0, fmt.Errorf("搜索会话 %q 失败: %w", nameOrID, err)
	}
	var resp struct {
		Data []struct {
			ID              uint64 `json:"id"`
			Name            string `json:"name"`
			OtherMemberName string `json:"other_member_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("解析搜索结果失败: %w", err)
	}
	// 精确匹配
	for _, c := range resp.Data {
		if c.Name == nameOrID || c.OtherMemberName == nameOrID {
			return c.ID, nil
		}
	}
	if len(resp.Data) > 0 {
		return resp.Data[0].ID, nil
	}
	return 0, fmt.Errorf("未找到会话 %q", nameOrID)
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

// userPost 以用户 JWT 身份 POST /api/v1/*（任务/日历等）。
// 与 bot token 不同：任务/日历归用户所有，用用户 token 创建语义正确。
func userPost(path string, body map[string]any) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败（先 qim config set）: %w", err)
	}
	if err := ensureUserToken(&cfg); err != nil {
		return nil, err
	}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, cfg.ServerURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+cfg.UserToken)
	return do(req)
}

// userGet 以用户 JWT 身份 GET /api/v1/*。
func userGet(path string) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := ensureUserToken(&cfg); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, cfg.ServerURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.UserToken)
	return do(req)
}

// userPut 以用户 JWT 身份 PUT /api/v1/*。
func userPut(path string, body map[string]any) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := ensureUserToken(&cfg); err != nil {
		return nil, err
	}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPut, cfg.ServerURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+cfg.UserToken)
	return do(req)
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

// extractOutputFlag 从 args 中提取 --output json 并返回剩余 args。
func extractOutputFlag(args []string) ([]string, string) {
	out := ""
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--output" && i+1 < len(args) {
			out = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--output=") {
			out = strings.TrimPrefix(args[i], "--output=")
			continue
		}
		rest = append(rest, args[i])
	}
	return rest, out
}

// out 在 --output json 时输出 JSON，否则输出人类可读文本。
// jsonVal 应为可被 json.Marshal 的值（如 map[string]any）。
func out(jsonVal any, humanFmt string, humanArgs ...any) {
	if outputFmt == "json" {
		b, _ := json.Marshal(jsonVal)
		fmt.Println(string(b))
		return
	}
	fmt.Printf(humanFmt, humanArgs...)
}

// outRaw 在 --output json 时输出原始 JSON body，否则输出人类可读文本。
func outRaw(rawJSON []byte, humanFmt string, humanArgs ...any) {
	if outputFmt == "json" {
		fmt.Println(string(rawJSON))
		return
	}
	fmt.Printf(humanFmt, humanArgs...)
}

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
