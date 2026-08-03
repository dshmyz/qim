package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
