package ai

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
)

func TestClassifyCompletionError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil 不报错", nil, ""},
		{"连接拒绝(穿透系统错误)", &net.OpError{Err: syscall.ECONNREFUSED}, "AI 服务未启动或无法连接，请检查 AI 网关是否运行"},
		{"连接拒绝(字符串形态)", errors.New(`Post "http://127.0.0.1:18080/v1/chat/completions": dial tcp 127.0.0.1:18080: connect: connection refused`), "AI 服务未启动或无法连接，请检查 AI 网关是否运行"},
		{"主机不存在", errors.New("dial tcp: lookup ai.local: no such host"), "AI 服务未启动或无法连接，请检查 AI 网关是否运行"},
		{"超时", context.DeadlineExceeded, "AI 服务响应超时，请稍后重试"},
		{"请求超时字符串", errors.New("context deadline exceeded"), "AI 服务响应超时，请稍后重试"},
		{"401 鉴权", errors.New("OpenAI API returned status 401: unauthorized"), "AI 服务鉴权失败，请检查 API Key"},
		{"403 鉴权", errors.New("OpenAI API returned status 403: forbidden"), "AI 服务鉴权失败，请检查 API Key"},
		{"429 限流", errors.New("OpenAI API returned status 429: rate limit"), "AI 服务请求过于频繁，请稍后重试"},
		{"400 模型拒绝", errors.New("OpenAI API returned status 400: bad request"), "AI 模型拒绝了请求，请检查模型配置"},
		{"500 服务内部", errors.New("OpenAI API returned status 500: internal"), "AI 服务内部错误，请稍后重试"},
		{"未知错误兜底", errors.New("some unknown failure"), "AI 服务暂时不可用，请稍后重试"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyCompletionError(tc.err); got != tc.want {
				t.Errorf("ClassifyCompletionError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWrapCompletionError_UserMessage(t *testing.T) {
	raw := &net.OpError{Err: syscall.ECONNREFUSED}
	wrapped := WrapCompletionError(raw)
	// 用户文案取分类结果
	if got := UserMessage(wrapped); got != "AI 服务未启动或无法连接，请检查 AI 网关是否运行" {
		t.Errorf("UserMessage(wrapped) = %q", got)
	}
	// 原始原因保留：errors.Is 仍可穿透，日志仍能看到底层错误
	if !errors.Is(wrapped, syscall.ECONNREFUSED) {
		t.Error("WrapCompletionError 应保留 errors.Is 穿透")
	}
	// 未包装的裸错误走通用兜底
	if got := UserMessage(raw); got != "AI 服务暂时不可用，请稍后重试" {
		t.Errorf("UserMessage(raw) = %q", got)
	}
	// nil 边界
	if got := UserMessage(nil); got != "" {
		t.Errorf("UserMessage(nil) = %q", got)
	}
	if WrapCompletionError(nil) != nil {
		t.Error("WrapCompletionError(nil) 应为 nil")
	}
}
