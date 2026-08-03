package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTaskServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	return db
}

// 自己的私人任务（conversation_id=0）能查到
func TestGetTaskForConversation_OwnPrivateTask(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	task := &model.Task{Title: "私人任务", UserID: 100, ConversationID: 0}
	require.NoError(t, db.Create(task).Error)

	got, err := svc.GetTaskForConversation(100, 200, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "私人任务", got.Title)
}

// 别人的私人任务（conversation_id=0）在会话里能查到（A 主动分享到会话，会话成员可见）
// 枚举风险可控：仅在知道 task_id 时能查到，/task 自动补全不列出别人的私人任务
func TestGetTaskForConversation_OthersPrivateTaskVisibleWhenReferenced(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	// A 的私人任务
	task := &model.Task{Title: "A的私人任务", UserID: 100, ConversationID: 0}
	require.NoError(t, db.Create(task).Error)

	// B（user_id=101）在会话 200 里查 A 的私人任务 —— 应能查到
	got, err := svc.GetTaskForConversation(101, 200, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "A的私人任务", got.Title)
}

// 别人的会话任务在同会话能查到（会话任务对会话成员共见）
func TestGetTaskForConversation_OthersConversationTaskVisible(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	// A 在会话 200 建的任务
	task := &model.Task{Title: "A建的群任务", UserID: 100, ConversationID: 200}
	require.NoError(t, db.Create(task).Error)

	// B（user_id=101）在会话 200 能查到 —— 不再受 user_id 过滤
	got, err := svc.GetTaskForConversation(101, 200, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "A建的群任务", got.Title)
	assert.Equal(t, uint(200), got.ConversationID)
}

// 不存在的任务 ID 返回错误
func TestGetTaskForConversation_NotFound(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	_, err := svc.GetTaskForConversation(100, 200, 99999)
	assert.Error(t, err)
}

// conversationID=0 被拒绝（防越权）
func TestGetTaskForConversation_ZeroConversationIDRejected(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	task := &model.Task{Title: "私人任务", UserID: 100, ConversationID: 0}
	require.NoError(t, db.Create(task).Error)

	_, err := svc.GetTaskForConversation(100, 0, task.ID)
	assert.Error(t, err, "conversationID=0 应被拒绝，防止越权")
}

// ListByConversation 列出该会话可引用的全部任务
// 含：该会话的任务（不限创建者）+ 自己的私人任务；不含别人的私人任务
func TestListByConversation_ReturnsReferableTasks(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	require.NoError(t, db.Create(&model.Task{Title: "我的私人任务", UserID: 100, ConversationID: 0}).Error)
	require.NoError(t, db.Create(&model.Task{Title: "我的群任务", UserID: 100, ConversationID: 200}).Error)
	require.NoError(t, db.Create(&model.Task{Title: "别人的群任务", UserID: 101, ConversationID: 200}).Error)
	require.NoError(t, db.Create(&model.Task{Title: "别人的私人任务", UserID: 101, ConversationID: 0}).Error)

	got, err := svc.ListByConversation(100, 200)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	titles := make([]string, len(got))
	for i, t := range got {
		titles[i] = t.Title
	}
	assert.Contains(t, titles, "我的私人任务")
	assert.Contains(t, titles, "我的群任务")
	assert.Contains(t, titles, "别人的群任务")
	assert.NotContains(t, titles, "别人的私人任务")
}

// ConversationID=0 被拒绝（防越权枚举私人任务）
func TestListByConversation_ZeroConversationIDRejected(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	require.NoError(t, db.Create(&model.Task{Title: "私人任务", UserID: 100, ConversationID: 0}).Error)

	_, err := svc.ListByConversation(100, 0)
	assert.Error(t, err, "conversationID=0 应被拒绝")
}

// 空会话返回空切片（而非 nil），方便 handler 序列化
func TestListByConversation_EmptyReturnsEmptySlice(t *testing.T) {
	db := setupTaskServiceTestDB(t)
	svc := NewTaskService(db)

	got, err := svc.ListByConversation(100, 999)
	require.NoError(t, err)
	assert.Empty(t, got)
}
