package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// httpClient 共享带超时的 HTTP 客户端，避免 poll 等长场景永久阻塞。
var httpClient = &http.Client{Timeout: 30 * time.Second}

// httpGetRaw 发送 GET 请求返回原始 body（不带 auth header）。
func httpGetRaw(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("HTTP %d: 读取响应失败: %w", resp.StatusCode, readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func httpGet(cfg config, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setAuth(req, cfg)
	return doReq(req)
}

func httpPost(cfg config, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	setAuth(req, cfg)
	return doReq(req)
}

// httpPut 发送 PUT 请求返回原始 body（带 bot token 鉴权）。
func httpPut(cfg config, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	setAuth(req, cfg)
	return doReq(req)
}

func setAuth(req *http.Request, cfg config) {
	if cfg.BotToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.BotToken)
	}
}

// userPost 以用户 JWT 身份 POST /api/v1/*（任务/日历等）。
func userPost(path string, body map[string]any) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败（先 qim config set）: %w", err)
	}
	if err := ensureUserToken(&cfg); err != nil {
		return nil, err
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, cfg.ServerURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+cfg.UserToken)
	return doReq(req)
}

// userGet 以用户 JWT 身份 GET /api/v1/*。
func userGet(path string) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := ensureUserToken(&cfg); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, cfg.ServerURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.UserToken)
	return doReq(req)
}

// userUpload 以用户 JWT 身份 multipart 上传文件到 /api/v1/upload。
// 返回原始响应 body（含 file id/url/name）。
func userUpload(path string, filename string, reader io.Reader) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := ensureUserToken(&cfg); err != nil {
		return nil, err
	}
	// 构建 multipart 表单
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, reader); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.ServerURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+cfg.UserToken)
	return doReq(req)
}

// userPut 以用户 JWT 身份 PUT /api/v1/*。
func userPut(path string, body map[string]any) ([]byte, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := ensureUserToken(&cfg); err != nil {
		return nil, err
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, cfg.ServerURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+cfg.UserToken)
	return doReq(req)
}

func doReq(req *http.Request) ([]byte, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("HTTP %d: 读取响应失败: %w", resp.StatusCode, readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}
