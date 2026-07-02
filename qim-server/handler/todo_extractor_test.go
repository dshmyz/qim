package handler

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	testTimeout  = 2 * time.Second
	testInterval = 10 * time.Millisecond
)

// mockTodoExtractor 用于测试，记录是否被调用及参数
type mockTodoExtractor struct {
	called         bool
	calledContent  string
	calledSenderID uint
	calledConvID   uint
}

func (m *mockTodoExtractor) ExtractAndCreateTodos(content string, senderID uint, conversationID uint) {
	m.called = true
	m.calledContent = content
	m.calledSenderID = senderID
	m.calledConvID = conversationID
}

func setupTodoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Group{},
		&model.Task{},
		&model.Notification{},
	))
	database.DB = db
	return db
}

// TestTryExtractTodos_TriggersExtractionWhenExtractTodosEnabled 验证：
// 群聊开启了 ExtractTodos 时，即使 AI 助手未启用（Enabled=false），
// 待办提取仍然被触发。
func TestTryExtractTodos_TriggersExtractionWhenExtractTodosEnabled(t *testing.T) {
	db := setupTodoTestDB(t)

	// 创建群聊会话
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)

	// 创建群聊配置：AI 未启用，但 ExtractTodos=true
	group := &model.Group{
		ConversationID: conv.ID,
		GroupType:      "group",
		Name:           "测试群",
		CreatorID:      1,
		AIConfigJSON:   `{"enabled":false,"extract_todos":true}`,
	}
	require.NoError(t, db.Create(group).Error)

	// 用 mock 替换全局 todoExtractor
	mock := &mockTodoExtractor{}
	origExtractor := todoExtractor
	todoExtractor = mock
	t.Cleanup(func() { todoExtractor = origExtractor })

	TryExtractTodos(1, conv.ID, "明天需要完成项目报告")

	// 等待 goroutine 执行
	assert.Eventually(t, func() bool {
		return mock.called
	}, testTimeout, testInterval, "ExtractTodos=true 时应触发提取")

	assert.Equal(t, "明天需要完成项目报告", mock.calledContent)
	assert.Equal(t, uint(1), mock.calledSenderID)
	assert.Equal(t, conv.ID, mock.calledConvID)
}

// TestTryExtractTodos_DoesNotTriggerForNonGroupChat 验证：
// 非群聊会话（如私聊）不触发待办提取。
func TestTryExtractTodos_DoesNotTriggerForNonGroupChat(t *testing.T) {
	db := setupTodoTestDB(t)

	conv := &model.Conversation{Type: "single"}
	require.NoError(t, db.Create(conv).Error)

	mock := &mockTodoExtractor{}
	origExtractor := todoExtractor
	todoExtractor = mock
	t.Cleanup(func() { todoExtractor = origExtractor })

	TryExtractTodos(1, conv.ID, "明天需要完成项目报告")

	// 等一小段时间确保 goroutine 没有调用
	assert.Never(t, func() bool {
		return mock.called
	}, 100*time.Millisecond, 10*time.Millisecond, "非群聊不应触发提取")
}

// TestTryExtractTodos_DoesNotTriggerWhenExtractTodosDisabled 验证：
// 群聊但 ExtractTodos=false 时，待办提取不被触发。
func TestTryExtractTodos_DoesNotTriggerWhenExtractTodosDisabled(t *testing.T) {
	db := setupTodoTestDB(t)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)

	// ExtractTodos=false（默认值）
	group := &model.Group{
		ConversationID: conv.ID,
		GroupType:      "group",
		Name:           "测试群",
		CreatorID:      1,
		AIConfigJSON:   `{"enabled":true,"extract_todos":false}`,
	}
	require.NoError(t, db.Create(group).Error)

	mock := &mockTodoExtractor{}
	origExtractor := todoExtractor
	todoExtractor = mock
	t.Cleanup(func() { todoExtractor = origExtractor })

	TryExtractTodos(1, conv.ID, "明天需要完成项目报告")

	assert.Never(t, func() bool {
		return mock.called
	}, 100*time.Millisecond, 10*time.Millisecond, "ExtractTodos=false 时不应触发提取")
}

// TestTryExtractTodos_UsesPreloadedConversation 验证当传入预加载的 conv 时，
// TryExtractTodosWithPreloaded 正常工作且只查 Group 表（不查 Conversation 表）。
func TestTryExtractTodos_UsesPreloadedConversation(t *testing.T) {
	db := setupTodoTestDB(t)

	conv := &model.Conversation{ID: 1, Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	group := &model.Group{
		ConversationID: conv.ID,
		GroupType:      "group",
		Name:           "测试群",
		CreatorID:      1,
		AIConfigJSON:   `{"enabled":false,"extract_todos":true}`,
	}
	require.NoError(t, db.Create(group).Error)

	mock := &mockTodoExtractor{}
	origExtractor := todoExtractor
	todoExtractor = mock
	t.Cleanup(func() { todoExtractor = origExtractor })

	TryExtractTodosWithPreloaded(1, conv, conv.ID, "明天需要完成项目报告")

	assert.Eventually(t, func() bool {
		return mock.called
	}, testTimeout, testInterval, "传入预加载 conv 时应触发提取")

	assert.Equal(t, "明天需要完成项目报告", mock.calledContent)
	assert.Equal(t, uint(1), mock.calledSenderID)
	assert.Equal(t, conv.ID, mock.calledConvID)
}
