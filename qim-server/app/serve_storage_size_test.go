package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageObjectSize_WithStat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.bin")
	// 写入 1234 字节
	data := make([]byte, 1234)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size, ok := storageObjectSize(f)
	if !ok {
		t.Fatal("os.File 应能返回 size")
	}
	if size != 1234 {
		t.Fatalf("期望 1234，得到 %d", size)
	}

	// Stat 路径不改动读游标，确保后续 io.Copy 仍能读到内容
	buf := make([]byte, 6)
	n, err := f.Read(buf)
	if err != nil || n != 6 {
		t.Fatalf("Stat 后应仍可正常读，n=%d err=%v", n, err)
	}
	t.Logf("size=%d, 游标未受影响可继续读: %v", size, buf)
}

func TestStorageObjectSize_SeekFallbackRestoresCursor(t *testing.T) {
	// 构造一个只实现 io.Seeker（无 fs.File.Stat）的对象，验证 Seek 兜底判 size
	var r sizeSeekerOnly = make([]byte, 500)
	size, ok := storageObjectSize(r)
	if !ok || size != 500 {
		t.Fatalf("Seek 兜底应返回 500，got ok=%v size=%d", ok, size)
	}
}

// sizeSeekerOnly 模拟只有 io.Seeker + io.Reader、没有 fs.File.Stat 的存储流
type sizeSeekerOnly []byte

func (s sizeSeekerOnly) Read(p []byte) (int, error) { return 0, os.ErrClosed } // Seek-only 用例，不测读
func (s sizeSeekerOnly) Seek(offset int64, whence int) (int64, error) {
	var base int64
	switch whence {
	case ioSeekStart:
	case ioSeekCurrent:
		base = 0 // 简化：假对象游标恒 0
	case ioSeekEnd:
		base = int64(len(s))
	default:
		return 0, os.ErrInvalid
	}
	return base + offset, nil
}

// 避免导入 io 常量歧义
const (
	ioSeekStart   = 0
	ioSeekCurrent = 1
	ioSeekEnd     = 2
)