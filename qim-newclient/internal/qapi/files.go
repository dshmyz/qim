package qapi

// 文件网盘（对齐 qim-client api/file.ts + 后端 handler/file_handler.go +
// handler/chunk_handler.go）：个人文件箱的文件/文件夹 CRUD、星标、批量操作，
// 以及 init/chunk/complete/cancel 四段分片上传。下载走流式写盘（大文件不进内存）。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// FileItem 文件记录（后端 model.File 序列化）。
type FileItem struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	OriginalName string        `json:"original_name"`
	Size         int64         `json:"size"`
	MimeType     string        `json:"mime_type"`
	StoragePath  string        `json:"storage_path"`
	FolderID     *int64        `json:"folder_id"`
	Source       string        `json:"source"`
	IsStarred    bool          `json:"is_starred"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	Uploader     *FileUploader `json:"uploader,omitempty"`
}

// FileUploader 群文件列表附带的上传者摘要。
type FileUploader struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// FolderItem 文件夹（/folders/tree 懒加载返回子层）。
type FolderItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ParentID    *int64 `json:"parent_id"`
	HasChildren bool   `json:"has_children"`
	FileCount   int    `json:"file_count"`
	CreatedAt   string `json:"created_at"`
}

// FileListResp GET /files 的分页载荷。
type FileListResp struct {
	Files    []FileItem `json:"files"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// FileListParams 列表过滤参数（folder_id 0 = 根目录，-1 = 不限）。
type FileListParams struct {
	FolderID  int64
	Starred   bool
	Search    string
	SortBy    string // created_at / name（空 = created_at）
	SortOrder string // asc / desc（空 = desc）
	Page      int
	PageSize  int
}

// ChunkInitResp POST /files/upload/init 载荷。
type ChunkInitResp struct {
	UploadID       string `json:"upload_id"`
	ChunkSize      int64  `json:"chunk_size"`
	TotalChunks    int    `json:"total_chunks"`
	UploadedChunks []int  `json:"uploaded_chunks"`
}

// ListFiles 文件列表（GET /files）。
func (c *Client) ListFiles(ctx context.Context, p FileListParams) (FileListResp, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(maxInt(p.Page, 1)))
	q.Set("page_size", strconv.Itoa(clampInt(p.PageSize, 1, 100, 20)))
	if p.FolderID >= 0 && !p.Starred { // 星标视图跨文件夹，后端按全局处理
		q.Set("folder_id", strconv.FormatInt(p.FolderID, 10))
	}
	if p.Starred {
		q.Set("starred", "true")
	}
	if p.Search != "" {
		q.Set("search", p.Search)
	}
	if p.SortBy != "" {
		q.Set("sort_by", p.SortBy)
	}
	if p.SortOrder != "" {
		q.Set("sort_order", p.SortOrder)
	}
	var out FileListResp
	if err := c.do(ctx, http.MethodGet, "/api/v1/files?"+q.Encode(), nil, &out); err != nil {
		return FileListResp{}, err
	}
	return out, nil
}

// StorageUsage 存储用量与配额（字节）（GET /files/storage-usage）。
func (c *Client) StorageUsage(ctx context.Context) (used, quota int64, err error) {
	var out struct {
		Used  int64 `json:"used"`
		Quota int64 `json:"quota"`
	}
	err = c.do(ctx, http.MethodGet, "/api/v1/files/storage-usage", nil, &out)
	return out.Used, out.Quota, err
}

// FolderTree 文件夹树。lazy 语义由后端 service 决定：parentID 传 -1 时拿根层。
func (c *Client) FolderTree(ctx context.Context, parentID int64) ([]FolderItem, error) {
	q := url.Values{}
	q.Set("lazy", "true")
	if parentID >= 0 {
		q.Set("parent_id", strconv.FormatInt(parentID, 10))
	}
	var out []FolderItem
	if err := c.do(ctx, http.MethodGet, "/api/v1/folders/tree?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateFolder 新建文件夹（POST /folders）。parentID 0 = 根目录。
func (c *Client) CreateFolder(ctx context.Context, name string, parentID int64) (FolderItem, error) {
	body := map[string]any{"name": name, "parent_id": parentID}
	if parentID <= 0 {
		body["parent_id"] = nil // 后端把 0 归一化为根，显式 null 更稳
	}
	var out FolderItem
	if err := c.do(ctx, http.MethodPost, "/api/v1/folders", body, &out); err != nil {
		return FolderItem{}, err
	}
	return out, nil
}

// DeleteFolder 删除文件夹。recursive=true 级联删子文件夹与其中文件。
func (c *Client) DeleteFolder(ctx context.Context, id int64, recursive bool) error {
	path := fmt.Sprintf("/api/v1/folders/%d", id)
	if recursive {
		path += "?recursive=true"
	}
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// UpdateFile 更新文件元信息（重命名/移动）。
func (c *Client) UpdateFile(ctx context.Context, id int64, name string, folderID *int64) error {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if folderID != nil {
		body["folder_id"] = *folderID
	}
	path := fmt.Sprintf("/api/v1/files/%d", id)
	return c.do(ctx, http.MethodPut, path, body, nil)
}

// DeleteFile 删除文件（软删除）。
func (c *Client) DeleteFile(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/v1/files/%d", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ToggleStar 星标/取消星标（服务端 toggle 语义），返回新状态。
func (c *Client) ToggleStar(ctx context.Context, id int64) (bool, error) {
	path := fmt.Sprintf("/api/v1/files/%d/star", id)
	var out FileItem
	if err := c.do(ctx, http.MethodPut, path, map[string]any{"starred": true}, &out); err != nil {
		return false, err
	}
	return out.IsStarred, nil
}

// BatchOperation 批量操作（POST /files/batch）：delete/move/star/unstar。
// move 必须带 targetFolderID（>0）。
func (c *Client) BatchOperation(ctx context.Context, operation string, fileIDs []int64, targetFolderID int64) error {
	if len(fileIDs) == 0 {
		return fmt.Errorf("文件列表为空")
	}
	ids := make([]int64, len(fileIDs))
	copy(ids, fileIDs)
	body := map[string]any{"file_ids": ids, "operation": operation}
	if operation == "move" {
		if targetFolderID <= 0 {
			return fmt.Errorf("移动操作需要目标文件夹")
		}
		body["target_folder_id"] = targetFolderID
	}
	return c.do(ctx, http.MethodPost, "/api/v1/files/batch", body, nil)
}

// InitChunkUpload 初始化分片上传（POST /files/upload/init）。
// 返回 chunk_size / total_chunks / 已传分片（断点续传）。
func (c *Client) InitChunkUpload(ctx context.Context, filename string, fileSize int64, folderID int64, resumeID string) (ChunkInitResp, error) {
	body := map[string]any{"filename": filename, "file_size": fileSize}
	if folderID > 0 {
		body["folder_id"] = folderID
	}
	if resumeID != "" {
		body["upload_id"] = resumeID
	}
	var out ChunkInitResp
	if err := c.do(ctx, http.MethodPost, "/api/v1/files/upload/init", body, &out); err != nil {
		return ChunkInitResp{}, err
	}
	if out.UploadID == "" {
		return out, fmt.Errorf("init 响应缺少 upload_id")
	}
	if out.ChunkSize <= 0 {
		out.ChunkSize = 5 << 20 // 后端缺省兜底：5MB
	}
	return out, nil
}

// UploadChunkFile 上传一个分片（POST /files/upload/chunk，multipart）。
// chunkData 由调用方切片，chunkHash 为该分片的 MD5 十六进制串。
func (c *Client) UploadChunkFile(ctx context.Context, uploadID string, chunkIndex int, chunkHash string, chunkData []byte) error {
	bodyFn := func() (io.Reader, string) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("upload_id", uploadID)
		_ = w.WriteField("chunk_index", strconv.Itoa(chunkIndex))
		_ = w.WriteField("chunk_hash", chunkHash)
		part, _ := w.CreateFormFile("chunk", "chunk")
		_, _ = part.Write(chunkData)
		_ = w.Close()
		return &buf, w.FormDataContentType()
	}
	var out map[string]any
	return c.doMedia(ctx, http.MethodPost, c.base+"/api/v1/files/upload/chunk", bodyFn, &out)
}

// CompleteChunkUpload 完成分片上传（POST /files/upload/complete），
// 服务端合并分片并建文件记录。
func (c *Client) CompleteChunkUpload(ctx context.Context, uploadID string) (FileItem, error) {
	var out FileItem
	err := c.do(ctx, http.MethodPost, "/api/v1/files/upload/complete", map[string]any{"upload_id": uploadID}, &out)
	if err != nil {
		return FileItem{}, err
	}
	return out, nil
}

// CancelChunkUpload 取消分片上传并清理已传分片。
func (c *Client) CancelChunkUpload(ctx context.Context, uploadID string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/files/upload/cancel", map[string]any{"upload_id": uploadID}, nil)
}

// DownloadFileTo 流式下载文件到本地路径（GET /files/:id/download）。
// 大文件逐块写盘，不整段进内存（分片上传可传远超 64MB 的文件，
// getBinary 的 64MB 上限在此不适用）。
func (c *Client) DownloadFileTo(ctx context.Context, id int64, destPath string) (written int64, err error) {
	raw := fmt.Sprintf("%s/api/v1/files/%d/download", c.base, id)
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		w, e := c.downloadOnce(ctx, raw, destPath)
		if e == nil {
			return w, nil
		}
		lastErr = e
		if !isUnauthorized(e) {
			return 0, e
		}
		if rerr := c.refresh(ctx); rerr != nil {
			return 0, lastErr
		}
	}
	return 0, lastErr
}

func (c *Client) downloadOnce(ctx context.Context, rawURL, destPath string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-App-Version", c.Version)
	req.Header.Set("X-App-Platform", c.Platform)
	if t, _ := c.Session(); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if resp.StatusCode >= http.StatusBadRequest ||
		strings.Contains(ct, "application/json") {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var env struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Code != 0 {
			return 0, &APIError{Code: env.Code, Message: env.Message}
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return 0, &APIError{Code: 401, Message: "未授权"}
		}
		return 0, fmt.Errorf("下载失败 HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("创建本地文件: %w", err)
	}
	n, cpErr := io.Copy(f, resp.Body)
	clErr := f.Close()
	if cpErr != nil {
		return n, fmt.Errorf("写盘: %w", cpErr)
	}
	if clErr != nil {
		return n, fmt.Errorf("收尾文件: %w", clErr)
	}
	return n, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, lo, hi, def int) int {
	if v < lo || v > hi {
		return def
	}
	return v
}
