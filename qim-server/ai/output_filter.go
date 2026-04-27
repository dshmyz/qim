package ai

import (
	"fmt"
)

// OutputLengthConfig 输出长度配置
type OutputLengthConfig struct {
	DefaultLimit int            `json:"default_limit"`
	TypeLimits   map[string]int `json:"type_limits"`
}

// DefaultOutputConfig 默认输出长度配置
var DefaultOutputConfig = OutputLengthConfig{
	DefaultLimit: 3000,
	TypeLimits: map[string]int{
		"ai_reply":     3000,
		"ai_summary":   5000,
		"ai_translate": 2000,
		"ai_rewrite":   2000,
		"ai_polish":    2000,
		"ai_daily":     8000,
	},
}

// FilterOutput 根据消息类型过滤输出长度
func (s *AIService) FilterOutput(content string, msgType string) string {
	config := DefaultOutputConfig
	limit, ok := config.TypeLimits[msgType]
	if !ok {
		limit = config.DefaultLimit
	}

	if len(content) > limit {
		return content[:limit] + "\n\n---\n*内容过长已截断,完整内容可导出查看*"
	}
	return content
}

// GetOutputLimit 获取指定类型的输出限制
func GetOutputLimit(msgType string) int {
	config := DefaultOutputConfig
	limit, ok := config.TypeLimits[msgType]
	if !ok {
		limit = config.DefaultLimit
	}
	return limit
}

// FormatContentPreview 格式化内容预览(用于截断提示)
func FormatContentPreview(fullLength int, limit int) string {
	return fmt.Sprintf("*内容已截断,显示前 %d 字符,完整内容共 %d 字符*", limit, fullLength)
}
