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

type config struct {
	ServerURL    string `json:"server_url"`
	BotToken     string `json:"bot_token"`
	UserToken    string `json:"user_token"`     // 用户 JWT，用于以用户身份调 /api/v1/*（任务/日历等）
	RefreshToken string `json:"refresh_token"`  // 用于自动续期 user_token
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "config":
		cmdConfig(os.Args[2:])
	case "login":
		cmdLogin(os.Args[2:])
	case "messages":
		cmdMessages(os.Args[2:])
	case "send":
		cmdSend(os.Args[2:])
	case "stream":
		cmdStream(os.Args[2:])
	case "stream-stdin":
		cmdStreamStdin(os.Args[2:])
	case "task":
		cmdTask(os.Args[2:])
	case "event":
		cmdEvent(os.Args[2:])
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
  login [-u username]                  交互式登录获取 user_token（自动续期，无需手动管理 token）
  messages list --thread ID [--after-id N] [--limit 50]   拉会话消息（JSON lines）
  messages poll --thread ID [--interval 2s]               轮询新消息（JSON lines）
  send --to USER --thread ID --type text|markdown|card --content ...   发消息，输出 message_id
  stream --message-id ID --delta "..." [--finish]         追加流式分段
  stream-stdin --to USER --thread ID                       stdin 逐行喂 delta，EOF finish
  task create --title "..." [--due 2026-08-01] [--priority low|medium|high] [--desc "..."]   以用户身份建待办
  event create --title "..." --start "2026-08-01 14:00" --end "2026-08-01 15:00" [--reminder 15] [--desc "..."]   以用户身份建日历事件
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

// ---------- task ----------

func cmdTask(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim task create")
		os.Exit(2)
	}
	switch args[0] {
	case "create":
		createTaskCmd(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: task "+args[0])
		os.Exit(2)
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
	fmt.Printf("✅ 待办已创建 (ID: %d)\n", resp.Data.ID)
}

// ---------- event ----------

func cmdEvent(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: qim event create")
		os.Exit(2)
	}
	switch args[0] {
	case "create":
		createEventCmd(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "未知子命令: event "+args[0])
		os.Exit(2)
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
	fmt.Printf("✅ 事件已创建 (ID: %d)\n", resp.Data.ID)
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
