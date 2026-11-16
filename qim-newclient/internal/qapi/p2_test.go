package qapi

// P2 富媒体协议：content 的两种历史格式、上传 multipart 字段、
// 二进制下载与 401 重放、直连失败后的文件接口回退。

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestParseAttachmentFormats(t *testing.T) {
	// 新格式：content 是 JSON
	a := ParseAttachment(`{"url":"/static/uploads/2026/11/x.png","id":109,"name":"原图.png","size":2048,"mimeType":"image/png"}`, nil)
	if a.URL != "/static/uploads/2026/11/x.png" || a.ID != "109" || a.Name != "原图.png" || a.Size != 2048 {
		t.Fatalf("JSON 解析不对: %+v", a)
	}
	if !a.IsImage() || a.DisplayName() != "原图.png" {
		t.Fatalf("图片判定/显示名不对: %+v", a)
	}
	// 后端历史里也有 snake_case 别名
	b := ParseAttachment(`{"file_url":"/static/a.pdf","file_id":"77","file_name":"报告.pdf","file_size":1024,"mime_type":"application/pdf"}`, nil)
	if b.URL != "/static/a.pdf" || b.ID != "77" || b.Name != "报告.pdf" || b.Size != 1024 || b.KindLabel() != "PDF" {
		t.Fatalf("snake_case 别名没解析: %+v", b)
	}
	// 老格式：content 直接是 URL，名字从路径里取
	c := ParseAttachment("https://cdn.example.com/a/b/cat.jpg?x=1", nil)
	if c.URL == "" || c.DisplayName() != "cat.jpg" || !c.IsImage() {
		t.Fatalf("纯 URL 格式没解析: %+v", c)
	}
	// 消息级字段兜底
	m := &Message{FileURL: "/static/f.zip", FileName: "打包.zip", FileSize: 4096}
	d := ParseAttachment("", m)
	if d.URL != "/static/f.zip" || d.Name != "打包.zip" || d.Size != 4096 {
		t.Fatalf("消息字段没兜底: %+v", d)
	}
	// 异常 content 不能变成可用附件
	if !ParseAttachment("这不是附件", nil).IsZero() {
		t.Error("普通文本不该被当成附件")
	}
}

func TestAttachmentContentRoundTrip(t *testing.T) {
	a := Attachment{
		URL: "/static/uploads/x.png", ID: "109", Name: "x.png",
		Size: 105, MimeType: "image/png",
	}
	content := a.Content()
	var raw map[string]any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		t.Fatalf("content 不是合法 JSON: %s", content)
	}
	// 后端 SendMessage 把 id 解成 uint，写成字符串会让文件来源更新失败。
	if _, ok := raw["id"].(float64); !ok {
		t.Fatalf("id 必须是数字: %v", raw["id"])
	}
	if raw["url"] != a.URL || raw["name"] != a.Name {
		t.Fatalf("字段丢失: %+v", raw)
	}
	got := ParseAttachment(content, nil)
	if got.URL != a.URL || got.ID != a.ID || got.Name != a.Name || got.Size != a.Size || got.MimeType != a.MimeType {
		t.Fatalf("往返不一致: %+v", got)
	}
	req := MediaRequest("image", a, "896")
	if req.Type != "image" || req.QuotedMessageID != 896 || req.FileName != "x.png" || !strings.HasPrefix(req.Content, "{") {
		t.Fatalf("发送请求不对: %+v", req)
	}
	// 非法引用 id 应该被忽略而不是让整条消息发不出去
	if MediaRequest("file", a, "local-3").QuotedMessageID != 0 {
		t.Error("本地临时 id 不该当引用 id")
	}
	if !IsMediaType("image") || !IsMediaType("FILE") || IsMediaType("text") {
		t.Error("IsMediaType 判定不对")
	}
}

func TestUploadDownloadAndReplay(t *testing.T) {
	fileData := []byte("\x89PNG\r\n\x1a\nP2-TEST-PAYLOAD")
	var mu sync.Mutex
	uploads, refreshed := 0, 0
	var lastFile, lastAuth string
	firstUnauthorized := true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/auth/refresh":
			mu.Lock()
			refreshed++
			mu.Unlock()
			p1Reply(w, map[string]any{"token": "new-token", "refresh_token": "new-refresh"})
		case r.URL.Path == "/api/v1/upload" && r.Method == http.MethodPost:
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("multipart 解析失败: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			f, header, err := r.FormFile("file")
			if err != nil {
				t.Errorf("后端拿不到 file 字段: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			got, _ := io.ReadAll(f)
			_ = f.Close()
			mu.Lock()
			uploads++
			lastFile, lastAuth = string(got), r.Header.Get("Authorization")
			need401 := firstUnauthorized
			firstUnauthorized = false
			mu.Unlock()
			if string(got) != string(fileData) {
				t.Errorf("上传内容不一致: %q", got)
			}
			if header.Filename != "pic.png" {
				t.Errorf("文件名字段不对: %q", header.Filename)
			}
			if need401 {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":1002,"message":"token 过期"}`))
				return
			}
			p1Reply(w, map[string]any{
				"id": 109, "url": "/static/uploads/pic.png",
				"name": "pic.png", "size": len(fileData),
			})
		case r.URL.Path == "/static/uploads/pic.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(fileData)
		case r.URL.Path == "/api/v1/files/109/download":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(fileData)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(tmp, fileData, 0o600); err != nil {
		t.Fatal(err)
	}
	c := New(srv.URL)
	c.SetSession("old-token", "refresh-1")

	att, err := c.Upload(context.Background(), tmp)
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	mu.Lock()
	nUploads, nRefresh, gotFile, gotAuth := uploads, refreshed, lastFile, lastAuth
	mu.Unlock()
	if nUploads != 2 || nRefresh != 1 {
		t.Fatalf("401 没有触发 refresh + 重放: uploads=%d refresh=%d", nUploads, nRefresh)
	}
	if gotFile != string(fileData) {
		t.Fatal("重放时 multipart body 被读空了")
	}
	if gotAuth != "Bearer new-token" {
		t.Fatalf("重放没用新 token: %q", gotAuth)
	}
	if att.ID != "109" || att.Name != "pic.png" || !att.IsImage() {
		t.Fatalf("上传返回解析不对: %+v", att)
	}

	got, err := c.Download(context.Background(), att)
	if err != nil {
		t.Fatalf("直连下载失败: %v", err)
	}
	if !bytes.Equal(got, fileData) {
		t.Fatalf("下载字节不一致: %q", got)
	}
	// 只有 id 时走文件接口回退
	got2, err := c.Download(context.Background(), Attachment{ID: "109", Name: "pic.png"})
	if err != nil || !bytes.Equal(got2, fileData) {
		t.Fatalf("文件接口下载失败: %v %q", err, got2)
	}
	if c.ResolveURL("/static/a.png") != srv.URL+"/static/a.png" ||
		c.ResolveURL("https://cdn.x/a.png") != "https://cdn.x/a.png" {
		t.Error("URL 拼接不对")
	}
}

func TestDownloadFallsBackToFileEndpoint(t *testing.T) {
	want := []byte("fallback-bytes")
	var mu sync.Mutex
	direct, viaID := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		switch r.URL.Path {
		case "/static/broken.bin":
			direct++
		case "/api/v1/files/55/download":
			viaID++
		}
		mu.Unlock()
		if r.URL.Path == "/static/broken.bin" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(want)
	}))
	defer srv.Close()

	c := New(srv.URL)
	got, err := c.Download(context.Background(), Attachment{URL: "/static/broken.bin", ID: "55", Name: "b.bin"})
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("回退失败: %v %q", err, got)
	}
	mu.Lock()
	defer mu.Unlock()
	if direct != 1 || viaID != 1 {
		t.Fatalf("通道顺序不对 direct=%d viaID=%d", direct, viaID)
	}
}

func TestUploadRejectsHugeFile(t *testing.T) {
	// 不真的造 64MB 文件：用稀疏文件（truncate）验证前置大小拦截。
	f := filepath.Join(t.TempDir(), "huge.bin")
	fh, err := os.Create(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := fh.Truncate(maxMediaBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = fh.Close()
	c := New("http://127.0.0.1:1")
	if _, err := c.Upload(context.Background(), f); err == nil || !strings.Contains(err.Error(), "64 MB") {
		t.Fatalf("超大文件没被拦下: %v", err)
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, ""}, {512, "512 B"}, {2048, "2.0 KB"}, {3 << 20, "3.0 MB"}, {2 << 30, "2.0 GB"},
	}
	for _, c := range cases {
		if got := FormatSize(c.n); got != c.want {
			t.Errorf("FormatSize(%d)=%q want %q", c.n, got, c.want)
		}
	}
}
