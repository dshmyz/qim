package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// ensureUserToken 的本地预检逻辑不依赖网络：qusr_ 长期令牌必须放行，
// 普通 JWT 走过期预判 + refresh_token 续期路径。
func TestEnsureUserToken_QusrTokenPassesThrough(t *testing.T) {
	cfg := config{ServerURL: "http://localhost:8080", BotToken: "qbot_x", UserToken: "qusr_abc123"}
	if err := ensureUserToken(&cfg); err != nil {
		t.Fatalf("qusr_ 长期令牌不应触发本地 JWT 预检: %v", err)
	}
	if cfg.UserToken != "qusr_abc123" {
		t.Fatalf("qusr_ 令牌不应被续期逻辑改写")
	}
}

func TestEnsureUserToken_RequiresLoginWhenEmpty(t *testing.T) {
	cfg := config{ServerURL: "http://localhost:8080", BotToken: "qbot_x"}
	if err := ensureUserToken(&cfg); err == nil {
		t.Fatal("UserToken 为空时应报未登录")
	}
}

func TestEnsureUserToken_ExpiredJWTWithoutRefresh(t *testing.T) {
	cfg := config{ServerURL: "http://localhost:8080", BotToken: "qbot_x", UserToken: expiredJWT(t)}
	if err := ensureUserToken(&cfg); err == nil {
		t.Fatal("过期 JWT 且无 refresh_token 应报需要重新登录")
	}
}

func TestEnsureUserToken_ValidJWTPasses(t *testing.T) {
	cfg := config{ServerURL: "http://localhost:8080", BotToken: "qbot_x", UserToken: testJWT(t)}
	if err := ensureUserToken(&cfg); err != nil {
		t.Fatalf("未过期 JWT 不应报错: %v", err)
	}
}

func expiredJWT(t *testing.T) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, err := json.Marshal(map[string]int64{"exp": time.Now().Add(-time.Hour).Unix()})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}
