package repository

import (
	"context"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTaskTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	return db
}

// 自己的私人任务（conversation_id=0）能查到
// repo 不再做 user_id 过滤；私人任务的越权防护由 service 层特判完成
func TestFindByConversationAndID_OwnPrivateTask(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	task := &model.Task{Title: "私人任务", UserID: 100, ConversationID: 0}
	require.NoError(t, db.Create(task).Error)

	got, err := repo.FindByConversationAndID(context.Background(), 200, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "私人任务", got.Title)
}

// 会话任务在该会话能查到（不限创建者：会话任务对会话成员共见）
func TestFindByConversationAndID_ConversationTaskVisibleRegardlessOfCreator(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	// A 创建的会话任务
	task := &model.Task{Title: "A建的群任务", UserID: 100, ConversationID: 200}
	require.NoError(t, db.Create(task).Error)

	// B（user_id=101）也能查到 —— repo 不再用 user_id 过滤会话任务
	got, err := repo.FindByConversationAndID(context.Background(), 200, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "A建的群任务", got.Title)
}

// 任务关联到别的会话，在当前会话查不到（会话隔离）
func TestFindByConversationAndID_OtherConversationTaskCannotView(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	task := &model.Task{Title: "会话300的任务", UserID: 100, ConversationID: 300}
	require.NoError(t, db.Create(task).Error)

	_, err := repo.FindByConversationAndID(context.Background(), 200, task.ID)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

// 不存在的任务 ID 返回 RecordNotFound
func TestFindByConversationAndID_NotFound(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	_, err := repo.FindByConversationAndID(context.Background(), 200, 99999)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

// FindByConversationID 列出该会话可引用的全部任务
// 含：该会话关联的任务（不限创建者，会话任务对会话成员共见）+ 自己的私人任务
// 不含：别人的私人任务（防越权）、其他会话的任务（会话隔离）
func TestFindByConversationID_ReturnsReferableTasks(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// 自己的私人任务（应被查出）
	require.NoError(t, db.Create(&model.Task{Title: "我的私人任务", UserID: 100, ConversationID: 0}).Error)
	// 自己在该会话的任务（应被查出）
	require.NoError(t, db.Create(&model.Task{Title: "我的群任务", UserID: 100, ConversationID: 200}).Error)
	// 别人在该会话的任务（应被查出 —— 会话任务对会话成员共见）
	require.NoError(t, db.Create(&model.Task{Title: "别人的群任务", UserID: 101, ConversationID: 200}).Error)
	// 别人的私人任务（不应被查出 —— 越权）
	require.NoError(t, db.Create(&model.Task{Title: "别人的私人任务", UserID: 101, ConversationID: 0}).Error)
	// 自己在其他会话的任务（不应被查出 —— 会话隔离）
	require.NoError(t, db.Create(&model.Task{Title: "我别的会话任务", UserID: 100, ConversationID: 300}).Error)

	got, err := repo.FindByConversationID(ctx, 100, 200)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	titles := []string{got[0].Title, got[1].Title, got[2].Title}
	assert.Contains(t, titles, "我的私人任务")
	assert.Contains(t, titles, "我的群任务")
	assert.Contains(t, titles, "别人的群任务")
	assert.NotContains(t, titles, "别人的私人任务")
	assert.NotContains(t, titles, "我别的会话任务")
}

// 空会话返回空列表（而非 nil）
func TestFindByConversationID_EmptyReturnsEmptySlice(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	got, err := repo.FindByConversationID(context.Background(), 100, 999)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// ConversationID=0 直接拒绝（防越权枚举所有未关联会话的私人任务）
func TestFindByConversationID_ZeroConversationIDRejected(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	require.NoError(t, db.Create(&model.Task{Title: "私人任务", UserID: 100, ConversationID: 0}).Error)

	got, err := repo.FindByConversationID(context.Background(), 100, 0)
	require.Error(t, err)
	assert.Nil(t, got)
}
