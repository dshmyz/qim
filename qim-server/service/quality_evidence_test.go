package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
)

// TestSmartReplyContextQualityEvidenceBlocks 验证质量门 evidence 单源拼装：
// 非空块收录、空块过滤、引用内容纳入。字段清单与 SmartReplyContext 的动态上下文一一对应。
func TestSmartReplyContextQualityEvidenceBlocks(t *testing.T) {
	c := &SmartReplyContext{
		KnowledgeCtx: "知识库内容",
		MemoryCtx:    "", // 空块应被过滤
		ChatHistory:  "[user:甲]: 你好",
		PendingTasks: "待办A",
		Quoted:       &QuotedContext{Kind: QuotedText, Name: "对方", Text: "引用原文"},
	}
	blocks := c.QualityEvidenceBlocks()
	joined := strings.Join(blocks, "\n\n")
	for _, want := range []string{"知识库内容", "[user:甲]: 你好", "待办A", "引用原文"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("evidence 缺少 %q, got=%q", want, joined)
		}
	}
	if len(blocks) != 4 {
		t.Fatalf("空 MemoryCtx 应被过滤，期望 4 块, got %d: %q", len(blocks), blocks)
	}

	// 全空时返回空切片（供 runQualityGate 快速通道判定）。
	empty := (&SmartReplyContext{}).QualityEvidenceBlocks()
	if len(empty) != 0 {
		t.Fatalf("全空上下文应返回空 evidence, got=%q", empty)
	}
}

// TestAvatarReplyContextQualityEvidenceBlocks 分身侧 evidence 单源同构验证。
func TestAvatarReplyContextQualityEvidenceBlocks(t *testing.T) {
	c := &AvatarReplyContext{
		NoteContext:    "笔记内容",
		GroupKnowledge: "",
		MemoryContext:  "记忆内容",
		History:        "历史内容",
		TaskContext:    "任务内容",
	}
	blocks := c.QualityEvidenceBlocks()
	if len(blocks) != 4 {
		t.Fatalf("空 GroupKnowledge 应被过滤，期望 4 块, got %d: %q", len(blocks), blocks)
	}
	joined := strings.Join(blocks, "\n\n")
	for _, want := range []string{"笔记内容", "记忆内容", "历史内容", "任务内容"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("evidence 缺少 %q, got=%q", want, joined)
		}
	}
}

// TestPersistAIMessageExtraQualityNote 质量门旁路意见（quality_note）随 Extra 落库；
// 无意见时不写该键，全空时维持 no-op 语义。
func TestPersistAIMessageExtraQualityNote(t *testing.T) {
	msg := &model.Message{}
	PersistAIMessageExtra(func() *model.Message { return msg }, nil, nil, "回复可能存在无依据内容")
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(msg.Extra), &extra); err != nil {
		t.Fatalf("Extra 序列化失败: %v", err)
	}
	if extra["quality_note"] != "回复可能存在无依据内容" {
		t.Fatalf("quality_note 未落库: %v", extra)
	}

	// 无意见：不写 quality_note 键（仅知识来源照常）。
	msg2 := &model.Message{}
	PersistAIMessageExtra(func() *model.Message { return msg2 }, nil, nil)
	if msg2.Extra != "" {
		t.Fatalf("无任何内容时应为 no-op, got=%q", msg2.Extra)
	}
}
