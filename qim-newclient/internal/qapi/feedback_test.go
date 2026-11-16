package qapi

// 反馈与版本检查契约测试（POST/GET /feedbacks、GET /client/versions）。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLatestVersion(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/client/versions") {
			t.Errorf("路径不对: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"list":[
			{"version":"2.1.0","releaseDate":"2026-11-01","updateNotes":"修复若干问题","downloadUrl":"https://example.com/qim.dmg"}
		],"total":1}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	v, err := c.LatestVersion(context.Background(), "macos")
	if err != nil {
		t.Fatalf("拉版本失败: %v", err)
	}
	if !strings.Contains(gotQuery, "platform=macos") || !strings.Contains(gotQuery, "pageSize=1") {
		t.Fatalf("查询参数不对: %s", gotQuery)
	}
	if v == nil || v.Version != "2.1.0" || v.DownloadURL != "https://example.com/qim.dmg" {
		t.Fatalf("版本解析错: %+v", v)
	}
}

func TestLatestVersionEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"list":[],"total":0}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	v, err := c.LatestVersion(context.Background(), "linux")
	if err != nil || v != nil {
		t.Fatalf("空列表应返回 (nil,nil): %v %+v", err, v)
	}
}

func TestSubmitFeedback(t *testing.T) {
	var gotType, gotContent, gotShot string
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/feedbacks") {
			t.Errorf("请求不对: %s %s", r.Method, r.URL.Path)
		}
		gotCT = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Errorf("multipart 解析失败: %v", err)
		}
		gotType = r.FormValue("type")
		gotContent = r.FormValue("content")
		if f, hdr, err := r.FormFile("screenshot"); err == nil {
			gotShot = hdr.Filename
			_ = f.Close()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"message":"提交成功","data":{"id":7,"type":"bug","status":"pending"}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	// 带截图
	shot := filepath.Join(t.TempDir(), "shot.png")
	if err := os.WriteFile(shot, []byte("png-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitFeedback(context.Background(), "bug", "输入框换行会丢字", shot); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	if gotType != "bug" || gotContent != "输入框换行会丢字" || gotShot != "shot.png" {
		t.Fatalf("表单字段错: type=%q content=%q shot=%q", gotType, gotContent, gotShot)
	}
	if !strings.HasPrefix(gotCT, "multipart/form-data") {
		t.Fatalf("Content-Type 应为 multipart: %s", gotCT)
	}
	// 不带截图（screenshot 字段缺席但请求仍应成功）
	gotShot = ""
	if err := c.SubmitFeedback(context.Background(), "other", "纯文字反馈", ""); err != nil {
		t.Fatalf("无截图提交失败: %v", err)
	}
	if gotType != "other" || gotShot != "" {
		t.Fatalf("无截图字段错: type=%q shot=%q", gotType, gotShot)
	}
}

func TestMyFeedbacks(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"list":[
			{"id":7,"type":"bug","content":"输入框换行会丢字","status":"resolved","reply":"已修复","handlerName":"管理员","createdAt":"2026-11-15T10:00:00+08:00"},
			{"id":6,"type":"feature","content":"希望加夜间模式","status":"pending","createdAt":"2026-11-14T09:00:00Z"}
		],"total":2}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	list, err := c.MyFeedbacks(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("拉列表失败: %v", err)
	}
	if !strings.Contains(gotQuery, "page=1") || !strings.Contains(gotQuery, "pageSize=20") {
		t.Fatalf("分页参数不对: %s", gotQuery)
	}
	if len(list) != 2 {
		t.Fatalf("条数错: %d", len(list))
	}
	if list[0].Reply != "已修复" || list[0].HandlerName != "管理员" || list[0].Status != "resolved" {
		t.Fatalf("第一条字段错: %+v", list[0])
	}
	raw, _ := json.Marshal(list[1])
	if !strings.Contains(string(raw), `"status":"pending"`) {
		t.Fatalf("第二条序列化异常: %s", raw)
	}
}
