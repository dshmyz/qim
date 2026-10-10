package main

import (
	"testing"
)

// 用户报过的回归：qim config set --server/--token 重设 bot 令牌时，
// 不得清掉已配置的 user_token/refresh_token（合并语义）。
func TestConfigSet_PreservesUserToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// 初始配置：带 user_token + refresh_token
	init := newConfigSetCmd()
	init.SetArgs([]string{"--server", "http://a:1", "--token", "qbot_a", "--user-token", "qusr_keep"})
	if err := init.Execute(); err != nil {
		t.Fatalf("初始配置失败: %v", err)
	}

	// 只更新 server + bot token，不传 --user-token
	reset := newConfigSetCmd()
	reset.SetArgs([]string{"--server", "http://b:2", "--token", "qbot_b"})
	if err := reset.Execute(); err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	if cfg.ServerURL != "http://b:2" || cfg.BotToken != "qbot_b" {
		t.Fatalf("server/bot 令牌应更新: %+v", cfg)
	}
	if cfg.UserToken != "qusr_keep" {
		t.Fatalf("user_token 被覆盖: got %q", cfg.UserToken)
	}

	// 显式传 --user-token 时视为替换，refresh_token 一并作废
	replace := newConfigSetCmd()
	replace.SetArgs([]string{"--server", "http://b:2", "--token", "qbot_b", "--user-token", "qusr_new"})
	if err := replace.Execute(); err != nil {
		t.Fatalf("替换 user_token 失败: %v", err)
	}
	cfg, _ = loadConfig()
	if cfg.UserToken != "qusr_new" {
		t.Fatalf("显式 --user-token 应替换: got %q", cfg.UserToken)
	}

	// 追加模式：只传 --user-token，server/bot 保留
	appendCmd := newConfigSetCmd()
	appendCmd.SetArgs([]string{"--user-token", "qusr_appended"})
	if err := appendCmd.Execute(); err != nil {
		t.Fatalf("追加 user_token 失败: %v", err)
	}
	cfg, _ = loadConfig()
	if cfg.BotToken != "qbot_b" || cfg.UserToken != "qusr_appended" {
		t.Fatalf("追加模式应保留其余字段: %+v", cfg)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2.0.45", "2.0.45", 0},
		{"2.0.45", "2.0.10", 1},
		{"2.0.9", "2.0.10", -1},
		{"2.1", "2.0.99", 1},
		{"2.1", "2.1.0", 0},
		{"3.0.0", "2.9.9", 1},
		{"dev", "2.0.1", -1}, // 非数字段按 0 处理
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestBizMessage(t *testing.T) {
	if got := bizMessage([]byte(`{"code":1001,"message":"未找到会话"}`)); got != "未找到会话" {
		t.Fatalf("应提取 message: got %q", got)
	}
	if got := bizMessage([]byte(`not json`)); got != "not json" {
		t.Fatalf("解析失败应回退原文: got %q", got)
	}
}
