package qapi

// C 组杂项契约：问题反馈 + 客户端版本检查。
//
// 反馈（client_log_handler.go）：
//   - POST /api/v1/feedbacks（multipart：type/content 必填，screenshot 可选图片）
//   - GET /api/v1/my-feedbacks?page=&pageSize= → data.list []UserFeedback
//
// 版本（version_handler.go，公开接口无需鉴权）：
//   - GET /api/v1/client/versions?platform=macos&pageSize=1 → data.list[0]

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// ClientVersion 客户端版本信息（client_versions 表）。
type ClientVersion struct {
	Version     string `json:"version"`
	ReleaseDate string `json:"releaseDate"`
	UpdateNotes string `json:"updateNotes"`
	DownloadURL string `json:"downloadUrl"`
}

// ElectronClientVersion 与 Electron 青雀客户端一致的版本号；后端的
// client_versions 记录的就是这条版本线，检查更新时用它做比较基准。
const ElectronClientVersion = "2.0.39"

// PlatformOf 把 GOOS 换成后端 NormalizePlatform 认识的 macos/linux/windows。
func PlatformOf(goos string) string {
	return platformOf(goos)
}

// LatestVersion 拉某平台最新客户端版本；无版本记录时返回 (nil, nil)。
func (c *Client) LatestVersion(ctx context.Context, platform string) (*ClientVersion, error) {
	var wrap struct {
		List []ClientVersion `json:"list"`
	}
	path := "/api/v1/client/versions?pageSize=1"
	if platform != "" {
		path += "&platform=" + url.QueryEscape(platform)
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.List) == 0 {
		return nil, nil
	}
	return &wrap.List[0], nil
}

// UserFeedback 用户反馈条目（model.UserFeedback 直出）。
// CreatedAt 是 time.Time 的 JSON（RFC3339），展示时格式化。
type UserFeedback struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Screenshot  string `json:"screenshot"`
	Reply       string `json:"reply"`
	HandlerName string `json:"handlerName"`
	CreatedAt   string `json:"createdAt"`
}

// SubmitFeedback 提交问题反馈（multipart：type/content 必填，screenshot 可选）。
// 截图文件打不开时静默跳过（截图是可选项，不能因此卡住反馈本身）。
func (c *Client) SubmitFeedback(ctx context.Context, typ, content, screenshotPath string) error {
	bodyFn := func() (io.Reader, string) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("type", typ)
		_ = w.WriteField("content", content)
		if screenshotPath != "" {
			if f, err := os.Open(screenshotPath); err == nil {
				part, perr := w.CreateFormFile("screenshot", filepath.Base(screenshotPath))
				if perr == nil {
					_, _ = io.Copy(part, f)
				}
				_ = f.Close()
			}
		}
		_ = w.Close()
		return bytes.NewReader(buf.Bytes()), w.FormDataContentType()
	}
	return c.doMedia(ctx, http.MethodPost, c.base+"/api/v1/feedbacks", bodyFn, nil)
}

// MyFeedbacks 拉当前用户的反馈列表（GET /my-feedbacks，按 created_at desc）。
func (c *Client) MyFeedbacks(ctx context.Context, page, pageSize int) ([]UserFeedback, error) {
	path := fmt.Sprintf("/api/v1/my-feedbacks?page=%d&pageSize=%d", page, pageSize)
	var wrap struct {
		List []UserFeedback `json:"list"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.List, nil
}
