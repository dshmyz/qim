package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMapConfigToFrontend_MaxFileSizeStringFallback 验证当 file_upload:max_size 因历史
// 类型识别 bug 被存成 type=string（值为字节数字符串）时，读路径仍能解析出正确的 MB 数，
// 而不是硬回退到默认 50。配合 service 层 BatchUpdate 的 int/int64 识别一起关闭回归。
func TestMapConfigToFrontend_MaxFileSizeStringFallback(t *testing.T) {
	out := mapConfigToFrontend(map[string]interface{}{
		"file_upload:max_size": "104857600", // 100MB，字符串形态
	})
	assert.Equal(t, 100, out["maxFileSize"], "字符串形态的字节数应解析为对应 MB，而非回退 50")
}

// TestMapConfigToFrontend_MaxFileSizeInt 正常 int 路径不应受影响。
func TestMapConfigToFrontend_MaxFileSizeInt(t *testing.T) {
	out := mapConfigToFrontend(map[string]interface{}{
		"file_upload:max_size": 104857600,
	})
	assert.Equal(t, 100, out["maxFileSize"])
}

// TestMapConfigToFrontend_MaxFileSizeMissing 缺失时仍给默认值，保证前端有数。
func TestMapConfigToFrontend_MaxFileSizeMissing(t *testing.T) {
	out := mapConfigToFrontend(map[string]interface{}{})
	assert.Equal(t, 50, out["maxFileSize"])
}

// TestValidateClientUpdateBaseURL 校验客户端更新服务器地址：空值合法、http(s):// 合法、其余非法。
func TestValidateClientUpdateBaseURL(t *testing.T) {
	cases := []struct {
		name  string
		value interface{}
		valid bool
	}{
		{"空字符串合法", "", true},
		{"空白合法", "   ", true},
		{"非字符串视为未配置", 123, true},
		{"https 合法", "https://updates.example.com", true},
		{"http 合法", "http://192.168.1.10:8080", true},
		{"尾部斜杠合法", "https://updates.example.com/", true},
		{"缺协议非法", "updates.example.com", false},
		{"协议拼错非法", "ftp://updates.example.com", false},
		{"乱串非法", "not-a-url", false},
		{"裸 https scheme 非法", "https://", false},
		{"裸 http scheme 非法", "http://", false},
		{"scheme 后只有斜杠非法", "https:///path", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateClientUpdateBaseURL(tc.value)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestMapConfig_ClientUpdateBaseURL 验证前端字段 clientUpdateBaseUrl 与存储 key
// client:update_base_url 的双向映射一致（保存与回显闭环）。
func TestMapConfig_ClientUpdateBaseURL(t *testing.T) {
	mapped := mapConfigFromFrontend(map[string]interface{}{
		"clientUpdateBaseUrl": "https://updates.example.com",
	})
	assert.Equal(t, "https://updates.example.com", mapped["client:update_base_url"])

	displayed := mapConfigToFrontend(map[string]interface{}{
		"client:update_base_url": "https://updates.example.com",
	})
	assert.Equal(t, "https://updates.example.com", displayed["clientUpdateBaseUrl"])
}
