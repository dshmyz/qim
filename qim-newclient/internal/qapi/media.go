package qapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 富媒体消息的协议契约（照 qim-server/handler/message_handler.go 与
// qim-client/src/components/message/{ImageMessage,FileMessage}.vue 对齐）：
//
//   - type = "image" | "file" | "video" | "audio"
//   - content 通常是 JSON 字符串 {"url":"/static/uploads/...","id":123,
//     "name":"a.png","size":1234,"mimeType":"image/png"}；
//     历史消息里也存在「content 直接就是 URL」的老格式，两种都要认。
//   - 上传 POST /api/v1/upload（multipart 字段名 file）返回
//     data.{id,url,name,size}；发送时把这份结构原样塞回 content，
//     后端 SendMessage 会解析出 id 把文件来源标记为 "chat"。
//   - 下载优先直接 GET url（相对路径拼服务端地址），失败再回退
//     GET /api/v1/files/{id}/download（该接口强制鉴权、支持 inline）。

// maxMediaBytes 是单次上传/下载允许放进内存的字节数（64 MiB）。
// 后端单次上传上限是 500MB 且提供分片接口，但原生第一版先走简单路径，
// 超大文件给出明确提示，而不是把整个文件读进内存把进程撑爆。
const maxMediaBytes = 64 << 20

// Attachment 是图片/文件消息 content 归一化后的结果。
type Attachment struct {
	URL       string `json:"url,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Size      int64  `json:"size,omitempty"`
	MimeType  string `json:"mimeType,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// IsZero 表示这条消息其实没有可用附件（例如异常 content）。
func (a Attachment) IsZero() bool { return a.URL == "" && a.ID == "" }

// DisplayName 给卡片用：优先 name，其次 URL 里的文件名，最后兜底。
func (a Attachment) DisplayName() string {
	if a.Name != "" {
		return a.Name
	}
	if u, err := url.Parse(a.URL); err == nil && u.Path != "" {
		if base := filepath.Base(u.Path); base != "." && base != "/" {
			if decoded, derr := url.PathUnescape(base); derr == nil {
				return decoded
			}
			return base
		}
	}
	if a.ID != "" {
		return "附件 " + a.ID
	}
	return "未知附件"
}

// KindLabel 给卡片右上角的类型标签。
func (a Attachment) KindLabel() string {
	ext := strings.ToLower(filepath.Ext(a.DisplayName()))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".heic", ".svg":
		return "图片"
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		return "视频"
	case ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg":
		return "音频"
	case ".pdf":
		return "PDF"
	case ".zip", ".rar", ".7z", ".gz", ".tar":
		return "压缩包"
	case ".doc", ".docx":
		return "文档"
	case ".xls", ".xlsx", ".csv":
		return "表格"
	case ".ppt", ".pptx":
		return "演示"
	case ".go", ".js", ".ts", ".py", ".java", ".json", ".md", ".txt":
		return "代码 / 文本"
	}
	if ext != "" {
		return strings.TrimPrefix(ext, ".")
	}
	return "文件"
}

// IsImage 判断附件该按图片渲染还是按文件卡片渲染。
// mime 优先，扩展名兜底（历史消息常常只有文件名）。
func (a Attachment) IsImage() bool {
	if strings.HasPrefix(strings.ToLower(a.MimeType), "image/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(a.DisplayName())) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return true
	}
	return false
}

// Content 生成要发给后端的 content 字符串。
// id 必须是数字：后端 SendMessage 用 `json.Unmarshal` 到 struct{ID uint}，
// 写成字符串会导致文件来源不更新（不影响消息本身，但会留下孤儿文件）。
func (a Attachment) Content() string {
	m := map[string]any{"url": a.URL}
	if n, err := strconv.ParseUint(a.ID, 10, 64); err == nil && n > 0 {
		m["id"] = n
	}
	if a.Name != "" {
		m["name"] = a.Name
	}
	if a.Size > 0 {
		m["size"] = a.Size
	}
	if a.MimeType != "" {
		m["mimeType"] = a.MimeType
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// ParseAttachment 把消息 content 解析成附件。
// msg 可以为 nil；非 nil 时用消息上的 file_name / file_size / file_url 补齐
// 老格式没写进 content 的字段。
func ParseAttachment(content string, msg *Message) Attachment {
	content = strings.TrimSpace(content)
	var a Attachment
	var wire struct {
		URL       string `json:"url"`
		FileURL   string `json:"file_url"`
		ID        ID     `json:"id"`
		FileID    ID     `json:"file_id"`
		Name      string `json:"name"`
		FileName  string `json:"file_name"`
		Filename  string `json:"filename"`
		Size      int64  `json:"size"`
		FileSize  int64  `json:"file_size"`
		MimeType  string `json:"mimeType"`
		MimeSnake string `json:"mime_type"`
		Thumbnail string `json:"thumbnail"`
		ThumbURL  string `json:"thumbnail_url"`
	}
	if strings.HasPrefix(content, "{") && json.Unmarshal([]byte(content), &wire) == nil {
		a.URL = firstNonEmpty(wire.URL, wire.FileURL)
		a.ID = firstNonEmpty(wire.ID.String(), wire.FileID.String())
		a.Name = firstNonEmpty(wire.Name, wire.FileName, wire.Filename)
		a.Size = firstPositive(wire.Size, wire.FileSize)
		a.MimeType = firstNonEmpty(wire.MimeType, wire.MimeSnake)
		a.Thumbnail = firstNonEmpty(wire.Thumbnail, wire.ThumbURL)
	} else if strings.HasPrefix(content, "http://") ||
		strings.HasPrefix(content, "https://") ||
		strings.HasPrefix(content, "/") {
		// 老格式：content 本身就是 URL 或 /static/uploads/... 路径
		a.URL = content
	}
	if msg != nil {
		if a.URL == "" {
			a.URL = msg.FileURL
		}
		if a.Name == "" {
			a.Name = msg.FileName
		}
		if a.Size == 0 {
			a.Size = msg.FileSize
		}
		if a.Thumbnail == "" {
			a.Thumbnail = msg.ThumbnailURL
		}
	}
	if a.MimeType == "" && a.Name != "" {
		a.MimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(a.Name)))
	}
	return a
}

// IsMediaType 判断消息类型是否走富媒体渲染。
func IsMediaType(t string) bool {
	switch strings.ToLower(t) {
	case "image", "file", "video", "audio":
		return true
	}
	return false
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func firstPositive(v ...int64) int64 {
	for _, n := range v {
		if n > 0 {
			return n
		}
	}
	return 0
}

// MediaRequest 构造一条富媒体发送请求。
func MediaRequest(typ string, a Attachment, quotedID string) SendRequest {
	// file_name / file_size 后端 SendMessage 不解，但带上能让老版本网关、
	// 以及未来的消息检索走同一份元数据，不会改变消息内容。
	req := SendRequest{
		Type: typ, Content: a.Content(),
		FileName: a.Name, FileSize: a.Size,
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(quotedID), 10, 64); err == nil && n > 0 {
		req.QuotedMessageID = n
	}
	return req
}

// UploadedFile 是 /api/v1/upload 的返回。
type UploadedFile struct {
	ID   ID     `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (u *UploadedFile) Attachment(localName string) Attachment {
	name := u.Name
	if localName != "" {
		name = localName // 后端把 original_name 存成随机文件名，展示用本地原名
	}
	size := u.Size
	a := Attachment{URL: u.URL, ID: u.ID.String(), Name: name, Size: size}
	if a.MimeType == "" {
		a.MimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	}
	return a
}

// ResolveURL 把后端给的相对路径拼成可请求的绝对 URL。
// 绝对路径 /static/... 与 /uploads/... 都保留前导斜杠拼在 base 后面。
func (c *Client) ResolveURL(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return c.base + "/" + strings.TrimPrefix(raw, "/")
}

// Upload 上传本地文件，返回可写进消息 content 的附件信息。
// 大文件在读入内存前就拦下，避免一次上传把进程内存吃掉。
func (c *Client) Upload(ctx context.Context, path string) (Attachment, error) {
	f, err := os.Open(path)
	if err != nil {
		return Attachment{}, fmt.Errorf("打开文件: %w", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Attachment{}, fmt.Errorf("读取文件信息: %w", err)
	}
	if st.Size() > maxMediaBytes {
		return Attachment{}, fmt.Errorf("文件 %.1f MB 超过单次上传上限 64 MB，请先用网页端分片上传", float64(st.Size())/(1<<20))
	}
	data, err := io.ReadAll(io.LimitReader(f, maxMediaBytes+1))
	if err != nil {
		return Attachment{}, fmt.Errorf("读取文件: %w", err)
	}
	name := filepath.Base(path)

	bodyFn := func() (io.Reader, string) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, _ := w.CreateFormFile("file", name)
		_, _ = part.Write(data)
		_ = w.Close()
		return bytes.NewReader(buf.Bytes()), w.FormDataContentType()
	}

	var out UploadedFile
	if err := c.doMedia(ctx, http.MethodPost, c.base+"/api/v1/upload", bodyFn, &out); err != nil {
		return Attachment{}, err
	}
	if out.URL == "" {
		return Attachment{}, fmt.Errorf("上传响应缺少 url: %+v", out)
	}
	return out.Attachment(name), nil
}

// Download 下载附件内容。先走 content 里的 url（相对路径自动补 base），
// 失败且带 id 时回退 /api/v1/files/{id}/download。
func (c *Client) Download(ctx context.Context, a Attachment) ([]byte, error) {
	if raw := c.ResolveURL(a.URL); raw != "" {
		b, err := c.getBinary(ctx, raw)
		if err == nil {
			return b, nil
		}
		if a.ID == "" {
			return nil, err
		}
		first := err
		fallback := fmt.Sprintf("%s/api/v1/files/%s/download", c.base, url.PathEscape(a.ID))
		b, err = c.getBinary(ctx, fallback)
		if err != nil {
			return nil, fmt.Errorf("下载失败（直连: %v；文件接口: %v）", first, err)
		}
		return b, nil
	}
	if a.ID != "" {
		raw := fmt.Sprintf("%s/api/v1/files/%s/download", c.base, url.PathEscape(a.ID))
		return c.getBinary(ctx, raw)
	}
	return nil, fmt.Errorf("附件没有下载地址")
}

// getBinary 发一个带鉴权的 GET，返回原始字节；401 会先刷新 token 再重放。
func (c *Client) getBinary(ctx context.Context, rawURL string) ([]byte, error) {
	bodyFn := func() (io.Reader, string) { return nil, "" }
	var data []byte
	err := c.doMedia(ctx, http.MethodGet, rawURL, bodyFn, &data)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("下载内容为空")
	}
	return data, nil
}

// doMedia 是富媒体专用的请求通道：body 可能是 multipart，out 可能是
// JSON 对象也可能是 []byte（二进制）。沿用 do 的 401 → refresh → 重放语义，
// 每次重放都通过 bodyFn 重新构造 reader，避免 body 已被读空。
func (c *Client) doMedia(ctx context.Context, method, rawURL string, bodyFn func() (io.Reader, string), out any) error {
	once := func() error {
		body, contentType := bodyFn()
		req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
		if err != nil {
			return fmt.Errorf("构造请求: %w", err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("X-App-Version", c.Version)
		req.Header.Set("X-App-Platform", c.Platform)
		if t, _ := c.Session(); t != "" {
			req.Header.Set("Authorization", "Bearer "+t)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("请求 %s: %w", rawURL, err)
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaBytes+1))
		if err != nil {
			return fmt.Errorf("读取响应: %w", err)
		}
		if len(raw) > maxMediaBytes {
			return fmt.Errorf("响应超过 64 MB")
		}

		// JSON 响应（上传成功/业务错误）与二进制响应（下载）分开处理。
		ct := resp.Header.Get("Content-Type")
		isJSON := strings.Contains(ct, "application/json") ||
			(len(raw) > 0 && raw[0] == '{')
		if isJSON {
			var env struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Detail  json.RawMessage `json:"detail"`
				Data    json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(raw, &env); err == nil {
				if env.Code != 0 {
					return &APIError{Code: env.Code, Message: env.Message, Detail: env.Detail}
				}
				if resp.StatusCode == http.StatusUnauthorized {
					return &APIError{Code: 401, Message: "未授权"}
				}
				if out == nil {
					return nil
				}
				if _, ok := out.(*[]byte); ok {
					// 下载接口返回了 JSON（多半是错误页），当失败处理
					return fmt.Errorf("下载接口返回 JSON: %s", truncate(string(raw), 160))
				}
				if len(env.Data) == 0 || string(env.Data) == "null" {
					return nil
				}
				return json.Unmarshal(env.Data, out)
			}
		}
		if resp.StatusCode >= http.StatusBadRequest {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 160))
		}
		if out == nil {
			return nil
		}
		if b, ok := out.(*[]byte); ok {
			*b = raw
			return nil
		}
		return fmt.Errorf("响应不是 JSON，无法解析为对象")
	}

	err := once()
	if err == nil || !isUnauthorized(err) {
		return err
	}
	if rerr := c.refresh(ctx); rerr != nil {
		return err
	}
	return once()
}

// FormatSize 给 UI 显示人类可读的文件大小。
func FormatSize(n int64) string {
	switch {
	case n <= 0:
		return ""
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n < 1<<30:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	}
}
