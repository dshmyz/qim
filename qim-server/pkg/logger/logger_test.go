package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDiagModuleRoutesToDiagLog 验证 diag 模块日志进入独立 diag.log（供检索/依据诊断排查），
// 且非 diag 模块的日志不会混入。
func TestDiagModuleRoutesToDiagLog(t *testing.T) {
	dir := t.TempDir()
	lg := newLoggerWith(dir, "debug", DefaultRotateConfig())

	lg.With("module", "diag").Info("diag-test-marker")
	lg.With("module", "ai").Info("ai-test-marker")
	lg.Info("plain-test-marker")

	// lumberjack 异步写盘，轮询等待
	diagPath := filepath.Join(dir, "diag.log")
	aiPath := filepath.Join(dir, "ai.log")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		diag, errDiag := os.ReadFile(diagPath)
		ai, errAI := os.ReadFile(aiPath)
		if errDiag == nil && errAI == nil && len(diag) > 0 && len(ai) > 0 {
			diagStr, aiStr := string(diag), string(ai)
			if !strings.Contains(diagStr, "diag-test-marker") {
				t.Fatalf("diag.log 应包含 diag 标记日志，got %q", diagStr)
			}
			if strings.Contains(diagStr, "ai-test-marker") || strings.Contains(diagStr, "plain-test-marker") {
				t.Fatalf("diag.log 不应混入非 diag 模块日志，got %q", diagStr)
			}
			if !strings.Contains(aiStr, "ai-test-marker") {
				t.Fatalf("ai.log 应包含 ai 模块日志（既有模块路由不能退化），got %q", aiStr)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("diag.log / ai.log 未在超时内生成")
}
