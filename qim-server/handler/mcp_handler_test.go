package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/service"
	"github.com/stretchr/testify/assert"
)

func TestValidateExternalURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		allowPrivate bool
		wantErr      bool
		errMsg       string
	}{
		// — 正常 URL —
		{"正常 HTTPS", "https://mcp.example.com/sse", false, false, ""},
		{"正常 HTTP", "http://mcp.example.com:8080/tools", false, false, ""},

		// — 协议拒绝（allowPrivate 不放开协议校验）—
		{"file 协议", "file:///etc/passwd", false, true, "仅支持 http/https"},
		{"ftp 协议", "ftp://example.com/file", true, true, "仅支持 http/https"},

		// — localhost（精确匹配）—
		{"localhost", "http://localhost:8080/mcp", false, true, "不允许访问本机"},
		{"127.0.0.1", "http://127.0.0.1/mcp", false, true, "不允许访问本机"},
		{"::1", "http://[::1]:8080/mcp", false, true, "不允许访问本机"},
		{"0.0.0.0", "http://0.0.0.0/mcp", false, true, "不允许访问本机"},

		// — IP 私有/链路本地（字面 IP）—
		{"私有 10.x", "http://10.0.0.1/mcp", false, true, "不允许访问内网"},
		{"私有 172.16.x", "http://172.16.0.1/mcp", false, true, "不允许访问内网"},
		{"私有 192.168.x", "http://192.168.1.1/mcp", false, true, "不允许访问内网"},
		{"链路本地 169.254.x", "http://169.254.169.254/latest/meta-data/", false, true, "不允许访问内网"},

		// — 边界 IP 地址（0/0.0/0.0.0 作为主机名形式传入）—
		{"单零 0", "http://0:8080/mcp", false, true, "不允许访问本机"},
		{"双零 0.0", "http://0.0:8080/mcp", false, true, "不允许访问本机"},
		{"三零 0.0.0", "http://0.0.0:8080/mcp", false, true, "不允许访问本机"},

		// — 已知内部元数据/内网域名 —
		{"AWS 元数据", "http://169.254.169.254/latest/meta-data/", false, true, "不允许访问内网"},

		// — 输入校验 —
		{"空 URL", "", false, true, "仅支持 http/https"},
		{"无主机", "http://", false, true, "URL 缺少主机名"},

		// — allowPrivate=true：内网部署放开地址限制，但仍只允许 http/https 且要求主机名 —
		{"allowPrivate localhost", "http://localhost:9100/mcp", true, false, ""},
		{"allowPrivate 127.0.0.1", "http://127.0.0.1/mcp", true, false, ""},
		{"allowPrivate 私网 192.168.x", "http://192.168.1.5:9100/mcp", true, false, ""},
		{"allowPrivate 私网 10.x", "http://10.0.0.1/mcp", true, false, ""},
		{"allowPrivate 链路本地元数据", "http://169.254.169.254/latest/meta-data/", true, false, ""},
		{"allowPrivate 仍拒绝 file 协议", "file:///etc/passwd", true, true, "仅支持 http/https"},
		{"allowPrivate 仍要求主机名", "http://", true, true, "URL 缺少主机名"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateExternalURL(tt.url, tt.allowPrivate)
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
