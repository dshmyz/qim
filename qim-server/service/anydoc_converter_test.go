package service

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

// writeFakeBinary 写一个可执行的假 anydoc 脚本到临时目录，返回其路径。
// script 为脚本主体（在 stdout 输出 Markdown，通过 exit 码模拟 anydoc 行为）。
// 用于在不依赖真实 anydoc 二进制的情况下测试 CLI 封装逻辑。
func writeFakeBinary(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "anydoc")
	if runtime.GOOS == "windows" {
		path += ".bat"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return path
}

// fakeAnydocScript 生成一个输出固定 Markdown、按指定退出码退出的 shell 脚本。
func fakeAnydocScript(markdown string, exitCode int) string {
	return "#!/bin/sh\nprintf '%s' '" + markdown + "'\nexit " + strconv.Itoa(exitCode) + "\n"
}

func TestAnydocConverter(t *testing.T) {
	t.Run("convert success returns stdout markdown", func(t *testing.T) {
		bin := writeFakeBinary(t, fakeAnydocScript("# 报告\n\n正文内容", 0))
		c := NewAnydocConverter()
		c.bin = bin // 直接指定二进制，绕过 PATH 探测，保证确定性

		if !c.Available() {
			t.Fatal("Available() should be true with a valid binary")
		}
		got, err := c.Convert(filepath.Join(t.TempDir(), "x.doc"))
		if err != nil {
			t.Fatalf("Convert failed: %v", err)
		}
		if got != "# 报告\n\n正文内容" {
			t.Fatalf("got %q, want markdown output", got)
		}
	})

	t.Run("exit code 1 maps to unsupported error", func(t *testing.T) {
		bin := writeFakeBinary(t, fakeAnydocScript("", 1))
		c := NewAnydocConverter()
		c.bin = bin

		_, err := c.Convert(filepath.Join(t.TempDir(), "scan.pdf"))
		if err == nil {
			t.Fatal("expected error for exit code 1")
		}
		if !errors.Is(err, errAnydocUnsupported) {
			t.Fatalf("exit 1 should map to errAnydocUnsupported, got: %v", err)
		}
	})

	t.Run("exit code 2 returns generic error not unsupported", func(t *testing.T) {
		bin := writeFakeBinary(t, fakeAnydocScript("", 2))
		c := NewAnydocConverter()
		c.bin = bin

		_, err := c.Convert(filepath.Join(t.TempDir(), "x.doc"))
		if err == nil {
			t.Fatal("expected error for exit code 2")
		}
		if errors.Is(err, errAnydocUnsupported) {
			t.Fatalf("exit 2 should NOT map to unsupported, got errAnydocUnsupported")
		}
	})

	t.Run("no binary unavailable and convert errors", func(t *testing.T) {
		// 未找到任何 anydoc 时 bin 为空 → Available false，Convert 报错。
		// 依赖本机 PATH 里没有任何doc；为避免偶发装上，强制 bin="" 模拟"未部署"。
		c := NewAnydocConverter()
		c.bin = ""
		if c.Available() {
			t.Fatal("Available() should be false when no binary resolved")
		}
		if _, err := c.Convert(filepath.Join(t.TempDir(), "x.doc")); err == nil {
			t.Fatal("Convert should error when no binary resolved")
		}
	})

	t.Run("env var overrides PATH detection", func(t *testing.T) {
		bin := writeFakeBinary(t, fakeAnydocScript("env-ok", 0))
		t.Setenv(anydocBinaryEnv, bin)
		// 找一个确定不在 PATH 的可执行名，确保探测走 env 而非 LookPath
		c := NewAnydocConverter()
		if c.bin != bin {
			t.Fatalf("env override not applied: got %q want %q", c.bin, bin)
		}
		if _, err := c.Convert(filepath.Join(t.TempDir(), "x.doc")); err != nil {
			t.Fatalf("Convert with env binary failed: %v", err)
		}
	})
}
