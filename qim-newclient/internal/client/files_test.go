package client

// 文件箱（D 组文件网盘）逻辑测试：加载/目录导航/排序/搜索/建文件夹/
// 批量操作/分片上传（成功+取消）/展示辅助函数。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"qim-newclient/internal/qapi"
)

// filesFakeSrv 模拟后端文件网盘的最小集合（树/列表/用量/建目录/批量/分片）。
type filesFakeSrv struct {
	mu          sync.Mutex
	treeRoot    []qapi.FolderItem
	files       []qapi.FileItem
	used, quota int64
	created     []string      // CreateFolder 收到的名字
	batch       []string      // BatchOperation 收到的 operation
	cancelled   int           // cancel 次数
	blockChunk  chan struct{} // 非 nil 时阻塞分片上传（测取消）
}

func writeResp(w http.ResponseWriter, code int, data string) {
	w.Header().Set("Content-Type", "application/json")
	if data == "" {
		_, _ = io.WriteString(w, fmt.Sprintf(`{"code":%d}`, code))
		return
	}
	_, _ = io.WriteString(w, fmt.Sprintf(`{"code":%d,"data":%s}`, code, data))
}

func newFilesFake(t *testing.T) (*filesFakeSrv, *httptest.Server) {
	fs := &filesFakeSrv{
		treeRoot: []qapi.FolderItem{
			{ID: 3, Name: "工作文档", HasChildren: true},
			{ID: 4, Name: "随手记"},
		},
		files: []qapi.FileItem{
			{ID: 7, Name: "季度报告.pdf", Size: 10240, MimeType: "application/pdf", IsStarred: true, UpdatedAt: "2026-11-15T10:00:00Z"},
			{ID: 8, Name: "封面.png", Size: 4096, MimeType: "image/png", UpdatedAt: "2026-11-14T09:00:00Z"},
			{ID: 9, Name: "notes.txt", Size: 512, MimeType: "text/plain", UpdatedAt: "2026-11-13T08:00:00Z"},
		},
		used: 1048576, quota: 1073741824,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasSuffix(p, "/folders/tree"):
			fs.mu.Lock()
			out, _ := json.Marshal(fs.treeRoot)
			fs.mu.Unlock()
			writeResp(w, 0, string(out))
		case strings.HasSuffix(p, "/files/storage-usage"):
			fs.mu.Lock()
			used, quota := fs.used, fs.quota
			fs.mu.Unlock()
			writeResp(w, 0, fmt.Sprintf(`{"used":%d,"quota":%d}`, used, quota))
		case strings.HasSuffix(p, "/files") && r.Method == http.MethodGet:
			fs.mu.Lock()
			filesJSON, _ := json.Marshal(fs.files)
			total := len(fs.files)
			fs.mu.Unlock()
			writeResp(w, 0, fmt.Sprintf(`{"files":%s,"total":%d,"page":1,"page_size":100}`, filesJSON, total))
		case strings.HasSuffix(p, "/folders") && r.Method == http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			fs.mu.Lock()
			fs.created = append(fs.created, body.Name)
			fs.mu.Unlock()
			writeResp(w, 0, `{"id":50,"name":"x"}`)
		case strings.HasSuffix(p, "/files/batch"):
			var body struct {
				Operation string `json:"operation"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			fs.mu.Lock()
			fs.batch = append(fs.batch, body.Operation)
			fs.mu.Unlock()
			writeResp(w, 0, `{"deleted_count":1}`)
		case strings.HasSuffix(p, "/upload/init"):
			writeResp(w, 0, `{"upload_id":"up-1","chunk_size":8,"total_chunks":2,"uploaded_chunks":[]}`)
		case strings.HasSuffix(p, "/upload/chunk"):
			if ch := fs.blockChunkChannel(); ch != nil {
				<-ch // 卡住等取消
			}
			_ = r.ParseMultipartForm(8 << 20)
			io.WriteString(w, `{"code":0,"data":{"chunk_index":1}}`)
		case strings.HasSuffix(p, "/upload/complete"):
			writeResp(w, 0, `{"id":99,"name":"x"}`)
		case strings.HasSuffix(p, "/upload/cancel"):
			fs.mu.Lock()
			fs.cancelled++
			fs.mu.Unlock()
			writeResp(w, 0, "")
		default:
			t.Errorf("意外请求: %s %s", r.Method, p)
			writeResp(w, 0, "")
		}
	}))
	return fs, srv
}

func (fs *filesFakeSrv) blockChunkChannel() chan struct{} {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.blockChunk
}

// waitFilesBusy 等 filesBusy 复位（后台 goroutine 全部落定）。
func waitFilesBusy(t *testing.T, a *app) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		busy := a.filesBusy
		a.mu.Unlock()
		if !busy {
			time.Sleep(80 * time.Millisecond) // 给后续 reload goroutine 留时间
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("等待超时")
}

func newFilesApp(srvURL string) *app {
	return &app{
		loggedIn:      true,
		client:        qapi.New(srvURL),
		ctx:           context.Background(),
		treeKids:      make(map[int64][]qapi.FolderItem),
		treeExpanded:  make(map[int64]bool),
		filesSel:      make(map[int64]bool),
		filesFolderID: -1,
	}
}

// TestHumanSize / 类型标签：主干分支。
func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{{0, "0 B"}, {512, "512 B"}, {2048, "2.0 KB"}, {5 << 20, "5.0 MB"}, {3 << 30, "3.0 GB"}}
	for _, tc := range cases {
		if got := humanSize(tc.in); got != tc.want {
			t.Errorf("humanSize(%d) = %s, 期望 %s", tc.in, got, tc.want)
		}
	}
}

func TestFileKindLabel(t *testing.T) {
	cases := map[string]string{
		"":                         "文件",
		"image/png":                "图片",
		"video/mp4":                "视频",
		"audio/mpeg":               "音频",
		"application/pdf":          "PDF",
		"application/msword":       "文档",
		"application/vnd.ms-excel": "表格",
		"text/plain":               "文本",
		"application/zip":          "压缩包",
		"application/octet-stream": "文件",
	}
	for mime, want := range cases {
		if got := fileKindLabel(mime); got != want {
			t.Errorf("fileKindLabel(%q) = %s, 期望 %s", mime, got, want)
		}
	}
}

// TestFilesLoadAll 打开文件箱：树 + 列表 + 用量并发加载。
func TestFilesLoadAll(t *testing.T) {
	_, srv := newFilesFake(t)
	defer srv.Close()
	a := newFilesApp(srv.URL)

	a.openFiles()

	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.filesLoaded || a.filesLoading {
		t.Fatalf("应已加载完成: loaded=%v loading=%v", a.filesLoaded, a.filesLoading)
	}
	if len(a.treeRoot) != 2 || len(a.files) != 3 {
		t.Fatalf("树 %d 层 / 文件 %d 条", len(a.treeRoot), len(a.files))
	}
	if a.filesUsageUsed != 1048576 || a.filesUsageQuota != 1073741824 {
		t.Fatalf("用量错: %d/%d", a.filesUsageUsed, a.filesUsageQuota)
	}
}

// TestEnterFolderPath 面包屑反查：进入子文件夹时路径来自树缓存。
func TestEnterFolderPath(t *testing.T) {
	_, srv := newFilesFake(t)
	defer srv.Close()
	a := newFilesApp(srv.URL)

	a.openFiles()
	a.mutate(func() {
		a.treeKids[3] = []qapi.FolderItem{{ID: 30, Name: "子目录", ParentID: ptrInt64(3)}}
	})

	a.enterFolder(30, "子目录")
	a.mu.Lock()
	crumbLen := len(a.filesCrumb)
	folder := a.filesFolderID
	starred := a.filesStarred
	a.mu.Unlock()
	if crumbLen != 2 || folder != 30 || starred {
		t.Fatalf("面包屑 %d 段 / folder=%d / starred=%v", crumbLen, folder, starred)
	}
	// 回根
	a.enterFolder(-1, "")
	a.mu.Lock()
	crumbLen = len(a.filesCrumb)
	a.mu.Unlock()
	if crumbLen != 0 {
		t.Fatalf("根目录面包屑应为空: %d", crumbLen)
	}
}

// TestCycleSort 排序循环。
func TestCycleSort(t *testing.T) {
	_, srv := newFilesFake(t)
	defer srv.Close()
	a := newFilesApp(srv.URL)

	a.cycleSort()
	if a.filesSortIdx != 1 || filesSortPresets[1].label != "最早优先" {
		t.Fatalf("应切到第 2 档: %d %s", a.filesSortIdx, filesSortPresets[1].label)
	}
	for i := 0; i < 4; i++ {
		a.cycleSort()
	}
	if a.filesSortIdx != 1 {
		t.Fatalf("循环 4 次应回到第 2 档: %d", a.filesSortIdx)
	}
}

// TestCreateFolderFlow 新建文件夹 → 弹窗收起。
func TestCreateFolderFlow(t *testing.T) {
	fs, srv := newFilesFake(t)
	defer srv.Close()
	a := newFilesApp(srv.URL)

	a.openFiles()
	a.mutate(func() {
		a.filesModal = "create"
		a.newFolderName = "重要资料"
	})
	a.submitCreateFolder()
	waitFilesBusy(t, a)

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if len(fs.created) != 1 || fs.created[0] != "重要资料" {
		t.Fatalf("建目录请求: %v", fs.created)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.filesModal != "" || a.newFolderName != "" || a.filesBusy {
		t.Fatalf("弹窗应收起并复位: modal=%q busy=%v", a.filesModal, a.filesBusy)
	}
}

// TestBatchOps 勾选 → 批量星标 / 移动。
func TestBatchOps(t *testing.T) {
	fs, srv := newFilesFake(t)
	defer srv.Close()
	a := newFilesApp(srv.URL)

	a.openFiles()
	a.toggleFileSel(7)
	a.toggleFileSel(8)
	if got := len(a.filesSel); got != 2 {
		t.Fatalf("应选中 2 个: %d", got)
	}
	a.batchStarSelected(true)
	waitFilesBusy(t, a)
	a.toggleFileSel(7)
	a.beginMoveFiles(0)
	a.mutate(func() { a.moveSel = 3 })
	a.confirmMoveFiles()
	waitFilesBusy(t, a)

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if len(fs.batch) != 2 || fs.batch[0] != "star" || fs.batch[1] != "move" {
		t.Fatalf("批量请求: %v", fs.batch)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.filesSel) != 0 || a.filesModal != "" {
		t.Fatalf("批量后应清空选择与弹窗")
	}
}

// TestUploadWorkerChunked 分片上传全流程：2 片、完成态、不触发 cancel。
func TestUploadWorkerChunked(t *testing.T) {
	fs, srv := newFilesFake(t)
	defer srv.Close()
	a := newFilesApp(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "上传测试.bin")
	data := []byte("0123456789AB") // 12 字节，chunk_size=8 → 2 片
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	a.startFileUpload(path)
	deadline := time.Now().Add(5 * time.Second)
	for {
		a.mu.Lock()
		st := 0
		if len(a.uploadTasks) > 0 {
			st = a.uploadTasks[0].state
		}
		a.mu.Unlock()
		if st != 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("上传超时")
		}
		time.Sleep(30 * time.Millisecond)
	}

	a.mu.Lock()
	task := a.uploadTasks[0]
	state, done, size := task.state, task.done, task.size
	a.mu.Unlock()
	if state != 1 {
		t.Fatalf("应完成: state=%d", state)
	}
	if done != size || size != int64(len(data)) {
		t.Fatalf("进度 %d/%d（文件 %d 字节）", done, size, len(data))
	}
	fs.mu.Lock()
	cancelled := fs.cancelled
	fs.mu.Unlock()
	if cancelled != 0 {
		t.Fatalf("成功上传不应触发 cancel: %d", cancelled)
	}
}

// TestUploadCancel 取消：卡住分片 → cancel → 状态=已取消 + 服务端收到 cancel。
func TestUploadCancel(t *testing.T) {
	fs, srv := newFilesFake(t)
	defer srv.Close()
	ch := make(chan struct{})
	fs.mu.Lock()
	fs.blockChunk = ch
	fs.mu.Unlock()

	a := newFilesApp(srv.URL)
	dir := t.TempDir()
	path := filepath.Join(dir, "c.bin")
	if err := os.WriteFile(path, []byte("0123456789AB"), 0o600); err != nil {
		t.Fatal(err)
	}
	a.startFileUpload(path)
	// 等 init 完成
	deadline := time.Now().Add(3 * time.Second)
	for {
		a.mu.Lock()
		running := len(a.uploadTasks) > 0 && a.uploadTasks[0].uploadID != ""
		a.mu.Unlock()
		if running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("init 超时")
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.Lock()
	tid := a.uploadTasks[0].id
	a.mu.Unlock()
	a.cancelUploadTask(tid)
	// 放行被卡住的分片，让 worker 走到取消检查
	close(ch)

	deadline = time.Now().Add(5 * time.Second)
	for {
		a.mu.Lock()
		st := 0
		if len(a.uploadTasks) > 0 {
			st = a.uploadTasks[0].state
		}
		a.mu.Unlock()
		if st == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("取消未生效, state=%d", st)
		}
		time.Sleep(30 * time.Millisecond)
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.cancelled == 0 {
		t.Fatal("服务端应收到 cancel 请求")
	}
}

// TestPercent 进度百分比。
func TestPercent(t *testing.T) {
	tk := &fileUploadTask{size: 200, done: 100}
	if tk.percent() != 50 {
		t.Fatalf("percent = %d", tk.percent())
	}
	tk.done = 500
	if tk.percent() != 100 {
		t.Fatalf("应封顶 100: %d", tk.percent())
	}
	tk.size = 0
	if tk.percent() != 100 {
		t.Fatalf("零大小应视为 100: %d", tk.percent())
	}
}

// TestDismissUploadTask 移除已结束任务。
func TestDismissUploadTask(t *testing.T) {
	a := &app{}
	t1 := &fileUploadTask{id: 1, name: "a", state: 1}
	t2 := &fileUploadTask{id: 2, name: "b", state: 0}
	a.uploadTasks = []*fileUploadTask{t1, t2}
	a.dismissUploadTask(1)
	if len(a.uploadTasks) != 1 || a.uploadTasks[0].id != 2 {
		t.Fatalf("移除后剩: %v", a.uploadTasks)
	}
}

func ptrInt64(v int64) *int64 { return &v }
