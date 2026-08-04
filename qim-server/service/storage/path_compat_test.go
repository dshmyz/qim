package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/config"
)

// TestParsePathLegacyFormats 校验三种历史格式的解析结果，新旧格式最终落到同一 key。
func TestParsePathLegacyFormats(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/static/uploads/2026/01/x.png", "uploads/2026/01/x.png"},
		{"/uploads/2026/01/x.png", "uploads/2026/01/x.png"},
		{"/s3/uploads/2026/01/x.png", "uploads/2026/01/x.png"},
	}
	for _, c := range cases {
		_, key := ParsePath(c.in)
		if key != c.want {
			t.Errorf("ParsePath(%q) key = %q, want %q", c.in, key, c.want)
		}
	}
}

// TestUploadsLegacyRoute 验证 /uploads/<key> 旧格式 storagePath 能通过
// Manager.ByPath + LocalStorage 读取到磁盘文件（模拟新增的 /uploads/* 兼容路由）。
func TestUploadsLegacyRoute(t *testing.T) {
	base := filepath.Join(t.TempDir(), "uploads")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ls, err := NewLocalStorage(config.LocalStorageConfig{Path: base})
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	mgr := NewManager(ls)

	// 用新格式写入文件
	content := "hello legacy file"
	if err := ls.Put(context.Background(), "uploads/2026/01/photo.png",
		strings.NewReader(content), int64(len(content)), "image/png"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// 历史消息里存的是旧格式 /uploads/2026/01/photo.png
	legacyPath := "/uploads/2026/01/photo.png"
	st, key, ok := mgr.ByPath(legacyPath)
	if !ok || st == nil {
		t.Fatal("ByPath legacy not ok")
	}
	r, err := st.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get legacy key %q: %v", key, err)
	}
	b, _ := io.ReadAll(r)
	r.Close()
	if string(b) != content {
		t.Fatalf("content mismatch: got %q want %q", string(b), content)
	}
}
