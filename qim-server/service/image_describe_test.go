package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// parseDescribeResult：模型回复 → 描述文本。重点覆盖模型把 JSON 包在 ```json
// 代码围栏里或带前言（直接 Unmarshal 会失败、整段 JSON 泄漏进弹窗）的场景。
func TestParseDescribeResult(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   string
	}{
		{
			name:   "纯 JSON",
			result: `{"description": "这是一张风景照"}`,
			want:   "这是一张风景照",
		},
		{
			name:   "json 代码围栏包裹（原弹窗显示一坨 JSON 的根因）",
			result: "```json\n{\"description\": \"这是一张风景照\"}\n```",
			want:   "这是一张风景照",
		},
		{
			name:   "带前言后接 JSON",
			result: "好的，我来描述这张图片：\n{\"description\": \"一只戴帽子的猫\"}",
			want:   "一只戴帽子的猫",
		},
		{
			name:   "未按 JSON 返回的自由文本直接取全文",
			result: "图片里是一只正在睡觉的橘猫。",
			want:   "图片里是一只正在睡觉的橘猫。",
		},
		{
			name:   "自由文本里误含花括号不被误抽为 JSON",
			result: "图上写着{a、b}两个选项",
			want:   "图上写着{a、b}两个选项",
		},
		{
			name:   "description 为空的 JSON 返回原文",
			result: `{"description": ""}`,
			want:   `{"description": ""}`,
		},
		{
			name:   "空回复",
			result: "   ",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseDescribeResult(tt.result))
		})
	}
}
