package upload

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"testing"
)

// fakeStorage 是最小可用的 StorageBackend 假实现，仅用于验证校验逻辑，
// 不关注真实存储语义。
type fakeStorage struct {
	putCount int
}

func (f *fakeStorage) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	f.putCount++
	return nil
}
func (f *fakeStorage) Delete(context.Context, string) error { return nil }
func (f *fakeStorage) Kind() string                         { return "local" }

// mustMultipartHeader 构造一个可被 header.Open() 读取的文件头（与真实上传一致）。
// 通过把完整 multipart body 交给 multipart.Reader.ReadForm 解析得到。
func mustMultipartHeader(t *testing.T, field, filename, body string) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := fw.Write([]byte(body)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	form, err := multipart.NewReader(&buf, mw.Boundary()).ReadForm(4 << 20)
	if err != nil {
		t.Fatalf("ReadForm failed: %v", err)
	}
	t.Cleanup(func() { form.RemoveAll() })
	headers := form.File[field]
	if len(headers) == 0 {
		t.Fatalf("no uploaded file part for field %q", field)
	}
	return headers[0]
}

// TestSaveMultipartFile_SkipTypeCheck 验证 SkipTypeCheck 只对受信任来源放行可执行文件，
// 普通上传（默认 false）仍被黑名单拦截，防止回归引入"任意可执行文件可上传"。
func TestSaveMultipartFile_SkipTypeCheck(t *testing.T) {
	policy := NewPolicy(DefaultMaxSize, nil, false)
	body := "\x4d\x5a\x90" // 假 PE/MZ 头，MIME 不会被识别为危险类型

	cases := []struct {
		name          string
		filename      string
		skipTypeCheck bool
		wantErr       bool
	}{
		{"普通上传 .exe 仍被拦截（无 SkipTypeCheck）", "installer.exe", false, true},
		{"受信任来源 .exe 放行", "qim-mcp-windows-amd64.exe", true, false},
		{"受信任来源无扩展名二进制放行", "qim-cli-darwin-arm64", true, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			header := mustMultipartHeader(t, "file", c.filename, body)
			st := &fakeStorage{}
			_, err := SaveMultipartFile(header, SaveConfig{
				Policy:        policy,
				Storage:       st,
				KeyPrefix:     "uploads/2026/01/",
				SkipTypeCheck: c.skipTypeCheck,
			})
			if c.wantErr && err == nil {
				t.Fatalf("期望返回错误，但成功保存（filename=%q skipTypeCheck=%v）", c.filename, c.skipTypeCheck)
			}
			if !c.wantErr {
				if err != nil {
					t.Fatalf("期望保存成功，但返回错误: %v", err)
				}
				if st.putCount != 1 {
					t.Fatalf("期望调用一次 Put，实际 %d", st.putCount)
				}
			}
		})
	}
}
