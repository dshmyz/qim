package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextAssembler_Notes_InjectsSystemMessageAndSource
// SourceNotes 命中时产出系统上下文消息（文案与历史 bot 注入一致）+ KnowledgeSources
// （仅标题/分数，不暴露笔记正文），供 Extra 透出。
func TestContextAssembler_Notes_InjectsSystemMessageAndSource(t *testing.T) {
	fake := &fakeNoteSearcher{
		results: []SearchResult{
			{Content: "Q3 规划正文", Score: 0.92, Metadata: map[string]string{"title": "Q3 规划"}},
			{Content: "MCP 接入计划", Score: 0.85},
		},
	}
	asm := NewContextAssembler(nil)
	asm.SetNoteSearcher(fake)

	bundle := asm.Assemble(context.Background(), "Q3 规划", []ContextSource{
		{Type: SourceNotes, Key: 7, TopK: 3},
	})

	require.Len(t, bundle.Messages, 1, "命中笔记应产出 1 条 system 消息")
	assert.Equal(t, "system", bundle.Messages[0].Role)
	content := bundle.Messages[0].Content
	assert.Contains(t, content, "以下是创建者的相关笔记")
	assert.Contains(t, content, "[笔记: Q3 规划]\nQ3 规划正文")
	assert.Contains(t, content, "[笔记: 未命名]\nMCP 接入计划", "无标题时回退「未命名」")

	// KnowledgeSources 仅标题/分数，且 scope 走创建者 Key
	require.Len(t, bundle.KnowledgeSources, 2)
	assert.Equal(t, "Q3 规划", bundle.KnowledgeSources[0].Title)
	assert.InDelta(t, 0.92, bundle.KnowledgeSources[0].Score, 1e-9)
	assert.Equal(t, "未命名", bundle.KnowledgeSources[1].Title)
	assert.True(t, fake.called)
	assert.Equal(t, uint(7), fake.calledUserID, "scope 必须是声明的 Key（创建者 ID）")
	assert.Equal(t, 3, fake.calledTopK)
}

// TestContextAssembler_Notes_PreservesExactTitle 标题按原样保留，仅空标题回退「未命名」。
func TestContextAssembler_Notes_PreservesExactTitle(t *testing.T) {
	asm := NewContextAssembler(nil)
	asm.SetNoteSearcher(&fakeNoteSearcher{
		results: []SearchResult{{Content: "c", Score: 0.5, Metadata: map[string]string{"title": "真标题"}}},
	})
	bundle := asm.Assemble(context.Background(), "q", []ContextSource{{Type: SourceNotes, Key: 1, TopK: 3}})
	require.Len(t, bundle.KnowledgeSources, 1)
	assert.Equal(t, "真标题", bundle.KnowledgeSources[0].Title)

	asm2 := NewContextAssembler(nil)
	asm2.SetNoteSearcher(&fakeNoteSearcher{
		results: []SearchResult{{Content: "c", Score: 0.5}},
	})
	bundle2 := asm2.Assemble(context.Background(), "q", []ContextSource{{Type: SourceNotes, Key: 1, TopK: 3}})
	assert.Equal(t, "未命名", bundle2.KnowledgeSources[0].Title, "空标题回退为未命名")
}

// TestContextAssembler_Notes_NilSearcherSkips 无搜索器时静默跳过：不产出消息、不 panic。
func TestContextAssembler_Notes_NilSearcherSkips(t *testing.T) {
	asm := NewContextAssembler(nil) // 不 SetNoteSearcher，保留 nil
	bundle := asm.Assemble(context.Background(), "q", []ContextSource{{Type: SourceNotes, Key: 1, TopK: 3}})
	assert.Empty(t, bundle.Messages)
	assert.Empty(t, bundle.KnowledgeSources)
}

// TestContextAssembler_Notes_ErrorDegrades 检索出错时降级为不注入，不 panic。
func TestContextAssembler_Notes_ErrorDegrades(t *testing.T) {
	asm := NewContextAssembler(nil)
	asm.SetNoteSearcher(&fakeNoteSearcher{err: assert.AnError})
	bundle := asm.Assemble(context.Background(), "q", []ContextSource{{Type: SourceNotes, Key: 1, TopK: 3}})
	assert.Empty(t, bundle.Messages)
	assert.Empty(t, bundle.KnowledgeSources)
}

// TestContextAssembler_Notes_NoHitSkips 搜索器存在但无命中时也不注入。
func TestContextAssembler_Notes_NoHitSkips(t *testing.T) {
	asm := NewContextAssembler(nil)
	asm.SetNoteSearcher(&fakeNoteSearcher{})
	bundle := asm.Assemble(context.Background(), "q", []ContextSource{{Type: SourceNotes, Key: 1, TopK: 3}})
	assert.Empty(t, bundle.Messages)
	assert.Empty(t, bundle.KnowledgeSources)
}

// TestContextAssembler_History_BuildsCompactBlock
// SourceHistory 加载最近 TopK 条消息，按时间正序产出「name: content」紧凑块 +
// assistant 确认块，语义与侧边栏 current 模式既有注入一致。
func TestContextAssembler_History_BuildsCompactBlock(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	alice := &model.User{Username: "alice", Nickname: "Alice", Type: "user"}
	dan := &model.User{Username: "dan", Nickname: "", Type: "user"} // 无昵称，回退 username
	db.Create(alice)
	db.Create(dan)
	conv := &model.Conversation{}
	db.Create(conv)
	db.Create(&model.Message{ConversationID: conv.ID, SenderID: alice.ID, Type: "text", Content: "早", Origin: "user"})
	db.Create(&model.Message{ConversationID: conv.ID, SenderID: dan.ID, Type: "text", Content: "早啊", Origin: "user"})

	asm := NewContextAssembler(db)
	bundle := asm.Assemble(context.Background(), "q", []ContextSource{
		{Type: SourceHistory, Key: conv.ID, TopK: 20},
	})

	require.Len(t, bundle.Messages, 2)
	assert.Equal(t, "user", bundle.Messages[0].Role)
	assert.Contains(t, bundle.Messages[0].Content, "【最近对话记录】")
	assert.Contains(t, bundle.Messages[0].Content, "Alice: 早")
	assert.Contains(t, bundle.Messages[0].Content, "dan: 早啊", "无昵称回退 username")
	// 时间正序：早 在 早啊 之前
	assert.Less(t, strings.Index(bundle.Messages[0].Content, "Alice: 早"),
		strings.Index(bundle.Messages[0].Content, "dan: 早啊"))
	assert.Equal(t, "assistant", bundle.Messages[1].Role)
	assert.Contains(t, bundle.Messages[1].Content, "已了解当前会话上下文")
}

// TestContextAssembler_History_EmptyNoMessages 空历史不产出上下文块。
func TestContextAssembler_History_EmptyNoMessages(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	conv := &model.Conversation{}
	db.Create(conv)

	asm := NewContextAssembler(db)
	bundle := asm.Assemble(context.Background(), "q", []ContextSource{
		{Type: SourceHistory, Key: conv.ID, TopK: 20},
	})
	assert.Empty(t, bundle.Messages)
}

// TestContextAssembler_History_NilDBOrZeroKeyDegrades db==nil 或 Key==0 时静默降级，不 panic。
func TestContextAssembler_History_NilDBOrZeroKeyDegrades(t *testing.T) {
	asm := NewContextAssembler(nil)
	bundle := asm.Assemble(context.Background(), "q", []ContextSource{{Type: SourceHistory, Key: 0, TopK: 20}})
	assert.Empty(t, bundle.Messages)

	asm2 := NewContextAssembler(setupBotMessagingTestDB(t))
	bundle2 := asm2.Assemble(context.Background(), "q", []ContextSource{{Type: SourceHistory, Key: 0, TopK: 20}})
	assert.Empty(t, bundle2.Messages)
}

// TestContextAssembler_MultiSource_OrderedAndDegradedIndependently
// 声明多源时按声明顺序产出消息；单源失败不影响其他源（降级语义）。
func TestContextAssembler_MultiSource_OrderedAndDegradedIndependently(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	alice := &model.User{Username: "alice", Nickname: "Alice", Type: "user"}
	db.Create(alice)
	conv := &model.Conversation{}
	db.Create(conv)
	db.Create(&model.Message{ConversationID: conv.ID, SenderID: alice.ID, Type: "text", Content: "hi", Origin: "user"})

	asm := NewContextAssembler(db)
	asm.SetNoteSearcher(&fakeNoteSearcher{err: assert.AnError}) // 笔记检索失败，应被跳过

	bundle := asm.Assemble(context.Background(), "hi", []ContextSource{
		{Type: SourceNotes, Key: 7, TopK: 3},
		{Type: SourceHistory, Key: conv.ID, TopK: 20},
	})

	// notes 失败被跳过，history 仍正常产出；消息顺序与声明一致
	require.Len(t, bundle.Messages, 2)
	assert.Equal(t, "user", bundle.Messages[0].Role)
	assert.Empty(t, bundle.KnowledgeSources)
}

// TestMessageService_ContextAsmReceivesNoteSearcher
// MessageService.SetNoteSearcher 须同源透传到内部 contextAsm，保证 bot 走统一装配路径
// 时仍命中同一搜索器（否则 bot 笔记注入会因搜索器缺失而静默跳过）。
func TestMessageService_ContextAsmReceivesNoteSearcher(t *testing.T) {
	fake := &fakeNoteSearcher{
		results: []SearchResult{{Content: "c", Score: 0.5, Metadata: map[string]string{"title": "t"}}},
	}
	svc := NewMessageService(nil, nil, nil)
	svc.SetNoteSearcher(fake)
	require.NotNil(t, svc.contextAsm, "NewMessageService 应预置 contextAsm")
	require.NotNil(t, svc.contextAsm.noteSearcher, "SetNoteSearcher 应透传到 contextAsm")

	bundle := svc.contextAsm.Assemble(context.Background(), "q", []ContextSource{
		{Type: SourceNotes, Key: 7, TopK: 3},
	})
	assert.Len(t, bundle.Messages, 1)
}
