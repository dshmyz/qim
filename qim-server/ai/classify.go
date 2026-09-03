package ai

import (
	"context"
	"errors"
	"strings"
	"syscall"
)

// ClassifyCompletionError 把 GetCompletion 返回的错误映射为用户可读的提示语。
// provider 错误多为裸 fmt.Errorf 字符串（无类型），分类靠 errors.Is 穿透到系统错误
// + 关键子串匹配。返回值直接面向用户，handler 层无需再猜「连接失败」还是「内容问题」。
func ClassifyCompletionError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	// 基础设施不可达：AI 网关未启动 / 网络不通（最常见的是 connection refused）
	case errors.Is(err, syscall.ECONNREFUSED), errors.Is(err, syscall.ECONNRESET),
		errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, syscall.ENETUNREACH),
		strings.Contains(msg, "connection refused"), strings.Contains(msg, "no such host"),
		strings.Contains(msg, "ENOTFOUND"):
		return "AI 服务未启动或无法连接，请检查 AI 网关是否运行"
	// 超时
	case errors.Is(err, context.DeadlineExceeded),
		strings.Contains(msg, "timeout"), strings.Contains(msg, "ETIMEDOUT"),
		strings.Contains(msg, "deadline exceeded"):
		return "AI 服务响应超时，请稍后重试"
	// provider 返回的 HTTP 状态（provider 错误格式 "status %d"）
	case strings.Contains(msg, "status 401"), strings.Contains(msg, "status 403"):
		return "AI 服务鉴权失败，请检查 API Key"
	case strings.Contains(msg, "status 429"):
		return "AI 服务请求过于频繁，请稍后重试"
	case strings.Contains(msg, "status 400"), strings.Contains(msg, "status 404"):
		return "AI 模型拒绝了请求，请检查模型配置"
	case strings.Contains(msg, "status 5"):
		return "AI 服务内部错误，请稍后重试"
	default:
		return "AI 服务暂时不可用，请稍后重试"
	}
}
