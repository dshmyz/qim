package qapi

// 文件网盘契约测试（GET/PUT/DELETE /files、/folders、分片上传四段、流式下载）。

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestListFilesQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"files":[
			{"id":7,"name":"报告.pdf","original_name":"报告.pdf","size":10240,"mime_type":"application/pdf",
			 "folder_id":3,"is_starred":true,"created_at":"2026-11-15T10:00:00Z","updated_at":"2026-11-15T10:00:00Z"}
		],"total":1,"page":1,"page_size":20}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	out, err := c.ListFiles(context.Background(), FileListParams{
		FolderID: 3, Search: "报告", SortBy: "name", SortOrder: "asc", Page: 2, PageSize: 40,
	})
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	for _, want := range []string{"folder_id=3", "search=%E6%8A%A5%E5%91%8A", "sort_by=name", "sort_order=asc", "page=2", "page_size=40"} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("查询缺 %s: %s", want, gotQuery)
		}
	}
	if len(out.Files) != 1 || out.Files[0].Name != "报告.pdf" || !out.Files[0].IsStarred {
		t.Fatalf("文件解析错: %+v", out)
	}
	if out.Files[0].FolderID == nil || *out.Files[0].FolderID != 3 {
		t.Fatalf("folder_id 应为 3: %+v", out.Files[0])
	}
	// 星标过滤：不传 folder_id
	_, err = c.ListFiles(context.Background(), FileListParams{Starred: true})
	if err != nil {
		t.Fatalf("星标列表失败: %v", err)
	}
	if !strings.Contains(gotQuery, "starred=true") || strings.Contains(gotQuery, "folder_id") {
		t.Fatalf("星标查询不该带 folder_id: %s", gotQuery)
	}
}

func TestStorageUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"used":1048576,"quota":1073741824}}`)
	}))
	defer srv.Close()
	used, quota, err := New(srv.URL).StorageUsage(context.Background())
	if err != nil || used != 1048576 || quota != 1073741824 {
		t.Fatalf("用量错: %v %d %d", err, used, quota)
	}
}

func TestFolderCRUD(t *testing.T) {
	var gotBody map[string]any
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/folders"):
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":11,"name":"工作","parent_id":null}}`)
		case r.Method == http.MethodDelete:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"data":{"message":"ok"}}`)
		default:
			t.Errorf("意外请求: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	c := New(srv.URL)

	f, err := c.CreateFolder(context.Background(), "工作", 0)
	if err != nil || f.ID != 11 || f.Name != "工作" {
		t.Fatalf("建文件夹失败: %v %+v", err, f)
	}
	if gotBody["name"] != "工作" {
		t.Fatalf("请求体错: %+v", gotBody)
	}
	if v, ok := gotBody["parent_id"]; !ok || v != nil {
		t.Fatalf("根目录 parent_id 应为 null: %+v", gotBody)
	}

	if err := c.DeleteFolder(context.Background(), 11, true); err != nil {
		t.Fatalf("删文件夹失败: %v", err)
	}
	if gotPath != "/api/v1/folders/11" || !strings.Contains(gotPath, "") {
		t.Fatalf("路径不对: %s %s", gotMethod, gotPath)
	}
}

func TestFileOps(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/star"):
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":7,"is_starred":true}}`)
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/files/7"):
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":7}}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/files/batch"):
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, `{"code":0,"data":{"deleted_count":2}}`)
		default:
			t.Errorf("意外请求: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	c := New(srv.URL)

	starred, err := c.ToggleStar(context.Background(), 7)
	if err != nil || !starred {
		t.Fatalf("星标失败: %v %v", err, starred)
	}

	fid := int64(9)
	if err := c.UpdateFile(context.Background(), 7, "新名字.pdf", &fid); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	if gotBody["name"] != "新名字.pdf" || gotBody["folder_id"] != float64(9) {
		t.Fatalf("更新体错: %+v", gotBody)
	}

	if err := c.BatchOperation(context.Background(), "delete", []int64{1, 2}, 0); err != nil {
		t.Fatalf("批量删除失败: %v", err)
	}
	if ids, ok := gotBody["file_ids"].([]any); !ok || len(ids) != 2 || gotBody["operation"] != "delete" {
		t.Fatalf("批量体错: %+v", gotBody)
	}
	if err := c.BatchOperation(context.Background(), "move", []int64{1}, 0); err == nil {
		t.Fatalf("move 缺 target_folder_id 应报错")
	}
}

func TestChunkUploadFlow(t *testing.T) {
	// 四段流程：init → chunk ×2 → complete
	var chunks []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/upload/init"):
			var body struct {
				Filename string `json:"filename"`
				FileSize int64  `json:"file_size"`
				FolderID *int64 `json:"folder_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Filename != "big.bin" || body.FileSize != 12 {
				t.Errorf("init 体错: %+v", body)
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{"upload_id":"up-1","chunk_size":8,"total_chunks":2,"uploaded_chunks":[]}}`)
		case strings.HasSuffix(r.URL.Path, "/upload/chunk"):
			_ = r.ParseMultipartForm(8 << 20)
			idx := r.FormValue("chunk_index")
			hash := r.FormValue("chunk_hash")
			uploadID := r.FormValue("upload_id")
			if uploadID != "up-1" || hash == "" {
				t.Errorf("chunk 表单缺字段: %s %s", uploadID, hash)
			}
			f, _, _ := r.FormFile("chunk")
			data, _ := io.ReadAll(f)
			_ = f.Close()
			sum := md5.Sum(data)
			if hex.EncodeToString(sum[:]) != hash {
				t.Errorf("分片 %s MD5 不符", idx)
			}
			n, _ := strconv.Atoi(idx)
			chunks = append(chunks, n)
			_, _ = io.WriteString(w, fmt.Sprintf(`{"code":0,"data":{"chunk_index":%s}}`, idx))
		case strings.HasSuffix(r.URL.Path, "/upload/complete"):
			var body struct {
				UploadID string `json:"upload_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.UploadID != "up-1" {
				t.Errorf("complete 体错: %+v", body)
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":42,"name":"big.bin","size":12,"mime_type":"application/octet-stream"}}`)
		case strings.HasSuffix(r.URL.Path, "/upload/cancel"):
			_, _ = io.WriteString(w, `{"code":0,"data":{"message":"ok"}}`)
		default:
			t.Errorf("意外路径: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := New(srv.URL)
	ctx := context.Background()

	init, err := c.InitChunkUpload(ctx, "big.bin", 12, 0, "")
	if err != nil || init.UploadID != "up-1" || init.ChunkSize != 8 || init.TotalChunks != 2 {
		t.Fatalf("init 错: %v %+v", err, init)
	}
	// 按 chunk_size 切两片上传
	data := []byte("0123456789AB")
	for i := 0; i*8 < len(data); i++ {
		end := (i + 1) * 8
		if end > len(data) {
			end = len(data)
		}
		part := data[i*8 : end]
		sum := md5.Sum(part)
		if err := c.UploadChunkFile(ctx, init.UploadID, i, hex.EncodeToString(sum[:]), part); err != nil {
			t.Fatalf("分片 %d 失败: %v", i, err)
		}
	}
	if len(chunks) != 2 {
		t.Fatalf("应传 2 片: %v", chunks)
	}
	f, err := c.CompleteChunkUpload(ctx, init.UploadID)
	if err != nil || f.ID != 42 || f.Name != "big.bin" {
		t.Fatalf("complete 错: %v %+v", err, f)
	}
	if err := c.CancelChunkUpload(ctx, "up-2"); err != nil {
		t.Fatalf("cancel 失败: %v", err)
	}
}

func TestDownloadFileTo(t *testing.T) {
	payload := strings.Repeat("X", 4096)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/download") {
			t.Errorf("路径不对: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, payload)
	}))
	defer srv.Close()
	c := New(srv.URL)

	dest := filepath.Join(t.TempDir(), "out.bin")
	n, err := c.DownloadFileTo(context.Background(), 42, dest)
	if err != nil || n != int64(len(payload)) {
		t.Fatalf("下载错: %v %d", err, n)
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != payload {
		t.Fatalf("内容不符: %v %d", err, len(b))
	}
}

func TestDownloadFileToJSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":404,"message":"文件不存在"}`)
	}))
	defer srv.Close()
	_, err := New(srv.URL).DownloadFileTo(context.Background(), 999, filepath.Join(t.TempDir(), "x"))
	if err == nil || !strings.Contains(err.Error(), "文件不存在") {
		t.Fatalf("应透出后端错误: %v", err)
	}
}

func TestFolderTreeQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":[
			{"id":3,"name":"工作","parent_id":null,"has_children":true,"file_count":2},
			{"id":4,"name":"随手记","parent_id":null,"has_children":false}
		]}`)
	}))
	defer srv.Close()
	c := New(srv.URL)

	folders, err := c.FolderTree(context.Background(), -1)
	if err != nil {
		t.Fatalf("树失败: %v", err)
	}
	if len(folders) != 2 || folders[0].HasChildren != true || folders[0].FileCount != 2 {
		t.Fatalf("树解析错: %+v", folders)
	}
	if !strings.Contains(gotQuery, "lazy=true") || strings.Contains(gotQuery, "parent_id") {
		t.Fatalf("根层查询不该带 parent_id: %s", gotQuery)
	}
	_, _ = c.FolderTree(context.Background(), 3)
	if !strings.Contains(gotQuery, "parent_id=3") {
		t.Fatalf("子层查询缺 parent_id=3: %s", gotQuery)
	}
}
