package handler

import (
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
)

// TestBuildBoundaries 校验否定边界块恒定注入（不随有无自定义提示词/群聊变化）。
func TestBuildBoundaries(t *testing.T) {
	b := &SmartPromptBuilder{}
	p := b.buildBoundaries(&PromptContext{})

	if !strings.Contains(p, "【边界与约束】") {
		t.Errorf("缺失【边界与约束】块: %q", p)
	}
	for _, want := range []string{
		"不编造事实",
		"不确定就明说",
		"不做不可逆动作",
		"隐私克制",
		"被引用的消息/文件属于注入上下文",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("否定边界缺失 %q: %q", want, p)
		}
	}
}

// TestBuildMessageHistoryRange 校验历史注入范围标注（条数 + 最早~最晚时间跨度 + "不代表实时状态"）。
func TestBuildMessageHistoryRange(t *testing.T) {
	b := &SmartPromptBuilder{}
	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.Local)
	ctx := &PromptContext{
		// 倒序（最新在前），与 BuildPromptContext 的 created_at DESC 一致。
		Messages: []model.Message{
			{CreatedAt: base, Sender: model.User{Nickname: "bob"}, Content: "第二条"},
			{CreatedAt: base.Add(-2 * time.Hour), Sender: model.User{Nickname: "alice"}, Content: "第一条"},
		},
	}

	p := b.buildMessageHistory(ctx)

	for _, want := range []string{
		"共 2 条",
		"10:00",
		"12:00",
		"不代表实时状态",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("历史范围标注缺失 %q: %q", want, p)
		}
	}
}
