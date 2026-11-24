package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// jwtExpired 解析 JWT payload 检查是否过期（预留 60s 余量）。
// 注意：此函数仅做本地过期预判，不验证签名（CLI 场景可接受，
// 真正的鉴权由服务端完成）。
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

// reloginHint 统一的重新获取用户令牌提示：登录续期或改用开发者长期令牌，二选一。
// 所有"令牌过期/失效"类报错共用，改文案只动这里。
const reloginHint = "%s login，或在客户端「设置 → 开发者设置」签发 qusr_ 长期令牌后 %s config set --user-token qusr_..."

// ensureUserToken 确保 user_token 有效：过期则自动用 refresh_token 续期。
// 失败返回 error（需要重新 qim login）。
func ensureUserToken(cfg *config) error {
	// 开发者设置签发的长期令牌（qusr_…）：有效期/撤销由服务端管理，
	// 本地不做 JWT 过期预判（qusr_ 不是 JWT，jwtExpired 会误判为过期）。
	if strings.HasPrefix(cfg.UserToken, "qusr_") {
		return nil
	}
	if cfg.UserToken == "" {
		return fmt.Errorf("未配置用户令牌，请二选一:\n  1. %s login（交互式登录，token 自动续期）\n  2. 在客户端「设置 → 开发者设置」签发 qusr_ 长期令牌，然后 %s config set --user-token qusr_...", CliName, CliName)
	}
	if !jwtExpired(cfg.UserToken) {
		return nil // 还没过期
	}
	if cfg.RefreshToken == "" {
		return fmt.Errorf("token 已过期且无 refresh_token，请重新获取: %s", fmt.Sprintf(reloginHint, CliName, CliName))
	}
	if jwtExpired(cfg.RefreshToken) {
		return fmt.Errorf("refresh_token 也已过期，请重新获取: %s", fmt.Sprintf(reloginHint, CliName, CliName))
	}

	// 自动续期
	body, err := json.Marshal(map[string]string{"refresh_token": cfg.RefreshToken})
	if err != nil {
		return fmt.Errorf("序列化续期请求失败: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, cfg.ServerURL+"/api/v1/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建续期请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+cfg.RefreshToken)

	respBody, err := doReq(req)
	if err != nil {
		return fmt.Errorf("刷新 token 失败: %w（请重新获取: %s）", err, fmt.Sprintf(reloginHint, CliName, CliName))
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil || resp.Code != 0 {
		return fmt.Errorf("刷新 token 失败，请重新获取: %s", fmt.Sprintf(reloginHint, CliName, CliName))
	}

	cfg.UserToken = resp.Data.Token
	cfg.RefreshToken = resp.Data.RefreshToken
	if err := saveConfig(*cfg); err != nil {
		return fmt.Errorf("保存新 token 失败: %w", err)
	}
	fmt.Fprintln(os.Stderr, "🔄 token 已自动续期")
	return nil
}

// out 在 --output json 时输出 JSON，--output id 时只输出裸 ID（第一个 %d 参数），
// 否则输出人类可读文本。
func out(jsonVal any, humanFmt string, humanArgs ...any) {
	if outputFmt == "json" {
		b, _ := json.Marshal(jsonVal)
		fmt.Println(string(b))
		return
	}
	if outputFmt == "id" && len(humanArgs) > 0 {
		fmt.Println(humanArgs[0])
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

// botConversation 是 GET /api/v1/bot/conversations 的单项。
type botConversation struct {
	ConversationID uint64 `json:"conversation_id"`
	ConvType       string `json:"conv_type"`
	Name           string `json:"name"`
}

// lookupBotConversation 尝试把会话 ID 或名称解析为 bot 会话 ID。
// 名称在 /api/v1/bot/conversations 里精确匹配（群名或 1:1 对端昵称）。
// 未匹配到返回 (0, false)，由调用方决定是否回退到服务端 thread_name（用户名/昵称解析）。
func lookupBotConversation(nameOrID string) (uint64, bool) {
	if isNumeric(nameOrID) {
		id, err := strconv.ParseUint(nameOrID, 10, 64)
		if err != nil {
			die("无效的会话 ID: %s", nameOrID)
		}
		return id, true
	}

	cfg := mustConfig()
	body, err := httpGet(cfg, cfg.ServerURL+"/api/v1/bot/conversations")
	if err != nil {
		die("查询会话失败: %v", err)
	}
	var resp struct {
		Data []botConversation `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		die("解析会话列表失败: %v", err)
	}

	var matched []botConversation
	for _, c := range resp.Data {
		if c.Name == nameOrID {
			matched = append(matched, c)
		}
	}
	switch len(matched) {
	case 0:
		return 0, false
	case 1:
		return matched[0].ConversationID, true
	default:
		die("会话名 %q 不唯一（匹配 %d 个），请改用会话 ID", nameOrID, len(matched))
	}
	return 0, false
}

// resolveConversationID 严格模式：解析不到会话直接报错。
func resolveConversationID(nameOrID string) uint64 {
	id, found := lookupBotConversation(nameOrID)
	if !found {
		die("未找到会话 %q（用 `%s groups` 查看已入群会话，或改用会话 ID）", nameOrID, CliName)
	}
	return id
}

// warnThreadDeprecated 统一提示 --thread 已弃用。
func warnThreadDeprecated() {
	fmt.Fprintln(os.Stderr, "⚠️  --thread 已弃用，请改用 --conversation（两者都支持会话 ID 或名称）")
}

// resolveThreadArg 返回 fetchMessages 用的 thread 参数。
// --conversation 严格解析（不存在直接报错）；--thread 解析不到时保留原值，
// 交给服务端 thread_name 按用户名/昵称回退，兼容旧用法。
func resolveThreadArg(conversation, thread string) string {
	if conversation != "" {
		return fmt.Sprint(resolveConversationID(conversation))
	}
	if thread == "" {
		die("--conversation 必填（会话 ID 或名称）")
	}
	if id, found := lookupBotConversation(thread); found {
		return fmt.Sprint(id)
	}
	return thread
}
