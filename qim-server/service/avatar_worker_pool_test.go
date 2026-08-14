package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompactSources 覆盖分身回复「依据来源」下发前的压缩：nil 安全、按类型+标题去重、
// snippet 截断到 80 个字节-rune（保证 WS 载荷有界）。
func TestCompactSources(t *testing.T) {
	p := &AvatarWorkerPool{}

	// 空 / nil → 返回 nil，避免下发空 sources 字段
	assert.Nil(t, p.compactSources(nil))
	assert.Nil(t, p.compactSources([]KnowledgeSource{}))

	// 去重：同 source+title 仅保留一条；不同 source 同 title 不视为重复
	in := []KnowledgeSource{
		{Source: "notes", Title: "会议", Snippet: "a"},
		{Source: "notes", Title: "会议", Snippet: "b"},     // 重复（同 source+title）
		{Source: "knowledge", Title: "会议", Snippet: "c"}, // 不同 source，保留
	}
	out := p.compactSources(in)
	assert.Len(t, out, 2, "同 source+title 应去重")
	assert.Equal(t, "a", out[0].Snippet, "保留首次命中的 snippet")
	assert.Equal(t, "knowledge", out[1].Source)

	// snippet 截断到 80 个字符（按 rune，兼容中文）
	long := strings.Repeat("依", 200)
	trunc := p.compactSources([]KnowledgeSource{{Source: "memory", Snippet: long}})
	assert.Len(t, []rune(trunc[0].Snippet), 80, "snippet 应截断到 80 字符")
}

// helper：构造一个可直接驱动 enqueueToBucket 的裸工作池（仅承载合并桶，不触发 flush 落库）。
func newCoalesceTestPool() *AvatarWorkerPool {
	return &AvatarWorkerPool{buckets: make(map[avatarBatchKey]*avatarBatch)}
}

// TestAvatarWorkerPoolCoalesce_AccumulatesSameKey 同一 (UserID,ConversationID) 3 秒窗口内
// 连发的多条消息应聚合同一个桶，批内按序累积；未达批上限不触发 flush。
func TestAvatarWorkerPoolCoalesce_AccumulatesSameKey(t *testing.T) {
	p := newCoalesceTestPool()
	key := avatarBatchKey{UserID: 1, ConversationID: 10}
	for i := 0; i < 3; i++ {
		err := p.enqueueToBucket(
			AvatarTask{UserID: 1, ConversationID: 10, TriggerMessage: fmt.Sprintf("m%d", i), TriggerMsgType: "text"},
			AvatarBatchItem{Msg: fmt.Sprintf("m%d", i), MsgType: "text", TriggerUserID: 5},
		)
		require.NoError(t, err)
	}

	p.coalesceMu.Lock()
	b, ok := p.buckets[key]
	if b != nil && b.timer != nil {
		b.timer.Stop()
	}
	p.coalesceMu.Unlock()

	require.True(t, ok, "同一会话连发应建立合并桶")
	require.NotNil(t, b)
	require.Len(t, b.items, 3, "3 条连发应聚合成一批 3 项")
	assert.Equal(t, "m0", b.items[0].Msg, "批内按序累积")
	assert.Equal(t, "m2", b.items[2].Msg)
}

// TestAvatarWorkerPoolCoalesce_SeparateKeysIsolated 不同会话/不同分身用户的消息不应互相合并，
// 各自独立成桶。
func TestAvatarWorkerPoolCoalesce_SeparateKeysIsolated(t *testing.T) {
	p := newCoalesceTestPool()
	for _, task := range []AvatarTask{
		{UserID: 1, ConversationID: 10, TriggerMessage: "a", TriggerMsgType: "text"},
		{UserID: 1, ConversationID: 20, TriggerMessage: "b", TriggerMsgType: "text"},
		{UserID: 2, ConversationID: 10, TriggerMessage: "c", TriggerMsgType: "text"},
	} {
		require.NoError(t, p.enqueueToBucket(task, AvatarBatchItem{Msg: task.TriggerMessage, MsgType: "text"}))
	}

	p.coalesceMu.Lock()
	for _, b := range p.buckets {
		if b.timer != nil {
			b.timer.Stop()
		}
	}
	assert.Len(t, p.buckets, 3, "三个不同 key 各自成桶，互不合并")
	p.coalesceMu.Unlock()
}

// TestAvatarWorkerPoolCoalesce_ImageItemKeepsFileID 图片消息入桶时解析并保留文件 id，
// 供批量多模态读图使用（MsgType=image + FileID 完整）。
func TestAvatarWorkerPoolCoalesce_ImageItemKeepsFileID(t *testing.T) {
	p := newCoalesceTestPool()
	key := avatarBatchKey{UserID: 1, ConversationID: 10}
	task := AvatarTask{
		UserID: 1, ConversationID: 10,
		TriggerMessage: `{"id":42,"url":"/files/a.png","name":"cat.png"}`, TriggerMsgType: "image",
	}
	require.NoError(t, p.enqueueToBucket(task, AvatarBatchItem{Msg: task.TriggerMessage, MsgType: "image", FileID: 42, Name: "cat.png"}))

	p.coalesceMu.Lock()
	b, ok := p.buckets[key]
	if b != nil && b.timer != nil {
		b.timer.Stop()
	}
	p.coalesceMu.Unlock()

	require.True(t, ok)
	require.Len(t, b.items, 1)
	assert.Equal(t, "image", b.items[0].MsgType)
	assert.Equal(t, uint(42), b.items[0].FileID, "图片文件 id 应随批项保留供批量读图")
}
