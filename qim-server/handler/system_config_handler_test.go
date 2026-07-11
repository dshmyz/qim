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
