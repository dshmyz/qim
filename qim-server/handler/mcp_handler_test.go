package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateExternalURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{"正常 HTTPS", "https://mcp.example.com/sse", false, ""},
		{"正常 HTTP", "http://mcp.example.com:8080/tools", false, ""},
		{"file 协议", "file:///etc/passwd", true, "仅支持 http/https"},
		{"ftp 协议", "ftp://example.com/file", true, "仅支持 http/https"},
		{"localhost", "http://localhost:8080/mcp", true, "不允许访问本机"},
		{"127.0.0.1", "http://127.0.0.1/mcp", true, "不允许访问本机"},
		{"::1", "http://[::1]:8080/mcp", true, "不允许访问本机"},
		{"0.0.0.0", "http://0.0.0.0/mcp", true, "不允许访问本机"},
		{"私有 10.x", "http://10.0.0.1/mcp", true, "不允许访问内网"},
		{"私有 172.16.x", "http://172.16.0.1/mcp", true, "不允许访问内网"},
		{"私有 192.168.x", "http://192.168.1.1/mcp", true, "不允许访问内网"},
		{"链路本地 169.254.x", "http://169.254.169.254/latest/meta-data/", true, "不允许访问内网"},
		{"空 URL", "", true, "URL 格式错误"},
		{"无主机", "http://", true, "URL 缺少主机名"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExternalURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
