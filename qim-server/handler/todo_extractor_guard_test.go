package handler

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── 内容截断护栏 ─────────────────────────────────────────────────────────────

// TestExtractAndCreateTodos_TruncatesOversizedContent 问题⑤取证：
// 超长消息（粘贴长文）此前整段进 prompt，一条消息就能把提取调用的 token 打爆。
// 送 AI 前应按 rune 截断并附加截断标记。
func TestExtractAndCreateTodos_TruncatesOversizedContent(t *testing.T) {
	db := setupTodoTestDB(t)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "u1", PasswordHash: "x", Nickname: "张三"}).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: 1, Role: "member"}).Error)

	aiSvc, capProv := newExtractTestAI(t, "[]")
	e := NewTodoExtractor(aiSvc)

	long := strings.Repeat("项目背景资料", 500) // 2500 rune，超过上限
	e.ExtractAndCreateTodos(long, 1, conv.ID)

	require.NotNil(t, capProv.lastMessages, "Provider 应收到调用")
	userMsg := ""
	for _, m := range capProv.lastMessages {
		if m.Role == "user" {
			userMsg = m.Content
		}
	}
	// user 消息 = "张三: " 前缀 + 截断后的正文 + 截断标记
	body := strings.TrimPrefix(userMsg, "张三: ")
	assert.LessOrEqual(t, utf8.RuneCountInString(body), maxExtractContentRunes+20,
		"超长正文应被截断到上限内")
	assert.Contains(t, userMsg, "已截断", "应附加截断标记让模型知道内容不完整")
}

// TestRunePrefix 截断日志前缀的 rune 安全性：多字节字符不再被切出乱码。
func TestRunePrefix(t *testing.T) {
	s := strings.Repeat("项目", 30)
	assert.Len(t, []rune(runePrefix(s, 10)), 10, "应按 rune 数截取")
	assert.Equal(t, "项", runePrefix("项目", 1))
	assert.Equal(t, "abc", runePrefix("abc", 10), "不超长时原样返回")
}

// ── 有界队列护栏（入队 + 过载丢弃）────────────────────────────────────────

// fakeExtractSubmitter 记录入队 (key, handle) 并按 err 决定是否执行。
type fakeExtractSubmitter struct {
	mu        sync.Mutex
	key       uint
	handle    func()
	submitted bool
	err       error
}

func (f *fakeExtractSubmitter) Submit(key uint, handle func()) error {
	f.mu.Lock()
	f.key = key
	f.handle = handle
	f.submitted = true
	err := f.err
	f.mu.Unlock()
	if err != nil {
		return err
	}
	handle()
	return nil
}

func (f *fakeExtractSubmitter) snapshot() (submitted bool, key uint) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.submitted, f.key
}

func setupBusyGroup(t *testing.T) *model.Conversation {
	t.Helper()
	db := setupTodoTestDB(t)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	group := &model.Group{
		ConversationID: conv.ID,
		GroupType:      "group",
		Name:           "测试群",
		CreatorID:      1,
		AIConfigJSON:   `{"enabled":false,"extract_todos":true}`,
	}
	require.NoError(t, db.Create(group).Error)
	return conv
}

// TestTryExtractTodos_EnqueuesToBoundedWorker 提取不再裸 go，而是入队到有界
// worker 池（key=会话 ID，按会话串行），约束 AI 并发上限。
func TestTryExtractTodos_EnqueuesToBoundedWorker(t *testing.T) {
	conv := setupBusyGroup(t)

	fake := &fakeExtractSubmitter{}
	origSubmitter := extractSubmitter
	extractSubmitter = fake
	t.Cleanup(func() { extractSubmitter = origSubmitter })

	mock := &mockTodoExtractor{}
	origExtractor := todoExtractor
	todoExtractor = mock
	t.Cleanup(func() { todoExtractor = origExtractor })

	TryExtractTodos(1, conv.ID, "明天需要完成项目报告")

	assert.Eventually(t, func() bool {
		submitted, key := fake.snapshot()
		return submitted && key == conv.ID
	}, testTimeout, testInterval, "应入队且 key 为会话 ID")

	var called bool
	assert.Eventually(t, func() bool {
		called, _, _, _ = mock.snapshot()
		return called
	}, testTimeout, testInterval, "入队的 handle 应执行提取")
}

// TestTryExtractTodos_DropsWhenSubmitterBusy 队列过载（Submit 失败）时丢弃本条
// 提取并记日志——待办是派生工件，宁可丢一条提取也不能把 goroutine/AI 调用重新打满。
func TestTryExtractTodos_DropsWhenSubmitterBusy(t *testing.T) {
	conv := setupBusyGroup(t)

	fake := &fakeExtractSubmitter{err: errors.New("会话回复入队超时，已跳过")}
	origSubmitter := extractSubmitter
	extractSubmitter = fake
	t.Cleanup(func() { extractSubmitter = origSubmitter })

	mock := &mockTodoExtractor{}
	origExtractor := todoExtractor
	todoExtractor = mock
	t.Cleanup(func() { todoExtractor = origExtractor })

	TryExtractTodos(1, conv.ID, "明天需要完成项目报告")

	submitted, _ := fake.snapshot()
	require.True(t, submitted, "应先尝试入队")

	assert.Never(t, func() bool {
		called, _, _, _ := mock.snapshot()
		return called
	}, 200*time.Millisecond, 10*time.Millisecond, "入队失败时不应执行提取")
}
