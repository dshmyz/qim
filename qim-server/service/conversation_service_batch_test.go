package service

import (
	"context"
	"testing"
	"sync/atomic"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupConvServiceDB 构造内存 sqlite DB，并迁移所需表。
func setupConvServiceDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// newConvServiceWithInsertCounter 构造 ConversationService，并返回 INSERT 次数计数器。
// 通过注册 before:create 回调统计对 conversation_sessions 表的 INSERT 调用次数，
// 用以验证缺失 session 是否走批量创建而非 N 次单条 INSERT。
//
// 注意：回调必须在所有"预置数据"完成之后注册，避免预置 INSERT 被误计入。
func newConvServiceWithInsertCounter(t *testing.T, db *gorm.DB) (*ConversationService, *int32) {
	var insertCount int32
	if err := db.Callback().Create().Before("gorm:create").Register("count_session_inserts", func(tx *gorm.DB) {
		if tx.Statement.Table == "conversation_sessions" && tx.Statement.Schema != nil {
			atomic.AddInt32(&insertCount, 1)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	svc := NewConversationService(db)
	svc.convRepo = repository.NewConversationRepository(db)
	return svc, &insertCount
}

// countSessionInserts 注册 INSERT 计数回调并返回计数器指针。
// 供需要"先预置数据再注册回调"的测试使用，避免预置 INSERT 被误计入。
func countSessionInserts(t *testing.T, db *gorm.DB) *int32 {
	var insertCount int32
	if err := db.Callback().Create().Before("gorm:create").Register("count_session_inserts_"+t.Name(), func(tx *gorm.DB) {
		if tx.Statement.Table == "conversation_sessions" && tx.Statement.Schema != nil {
			atomic.AddInt32(&insertCount, 1)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	return &insertCount
}

// TestGetConversations_BatchCreatesMissingSessions 验证：
// 当用户有多个会话但全部缺失 ConversationSession 时，GetConversations
// 必须以"一次批量 INSERT"创建所有缺失 session，而不是 N 次单条 INSERT。
//
// 这是 P0 级性能修复的回归保护：循环内 s.db.Create(&session) 会随会话数
// 线性放大 DB 写入次数，尾延迟劣化。批量创建后只产生 1 次 INSERT。
func TestGetConversations_BatchCreatesMissingSessions(t *testing.T) {
	db := setupConvServiceDB(t)
	svc, insertCounter := newConvServiceWithInsertCounter(t, db)

	user := &model.User{Username: "perfuser", PasswordHash: "h"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 构造 5 个会话，全部缺失 session（模拟新用户首次拉取）
	const convCount = 5
	for i := 0; i < convCount; i++ {
		conv := &model.Conversation{Type: "single"}
		if err := db.Create(conv).Error; err != nil {
			t.Fatalf("create conv %d: %v", i, err)
		}
		if err := db.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         user.ID,
			Role:           "member",
		}).Error; err != nil {
			t.Fatalf("create member %d: %v", i, err)
		}
	}

	// 预置条件：没有任何 session
	var preCount int64
	db.Model(&model.ConversationSession{}).Count(&preCount)
	assert.Equal(t, int64(0), preCount, "测试前置：应无 session")

	result, err := svc.GetConversations(user.ID)
	assert.NoError(t, err)
	assert.Len(t, result, convCount)

	// 核心断言：一次批量 INSERT（允许 =1），而非 N 次（5 次）
	// batch create 会在 GORM 内部走单次 CREATE，回调计数为 1。
	got := atomic.LoadInt32(insertCounter)
	assert.Equal(t, int32(1), got, "缺失 session 必须批量创建，实际 INSERT 回调次数应为 1，得到 %d", got)

	// 结果正确性：所有会话都有 session 记录
	var sessionCount int64
	db.Model(&model.ConversationSession{}).Where("user_id = ?", user.ID).Count(&sessionCount)
	assert.Equal(t, int64(convCount), sessionCount, "所有缺失 session 都应被创建")
}

// TestGetConversations_DoesNotCreateExistingSessions 验证：
// 已存在的 session 不应被重复创建。
func TestGetConversations_DoesNotCreateExistingSessions(t *testing.T) {
	db := setupConvServiceDB(t)

	user := &model.User{Username: "perfuser2", PasswordHash: "h"}
	db.Create(user)

	conv := &model.Conversation{Type: "single"}
	db.Create(conv)
	db.Create(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         user.ID,
		Role:           "member",
	})
	// 预创建 session（在注册计数回调之前，避免被误计入）
	db.Create(&model.ConversationSession{
		UserID:         user.ID,
		ConversationID: conv.ID,
	})

	// 预置数据完成后再注册计数回调
	insertCounter := countSessionInserts(t, db)
	svc := NewConversationService(db)
	svc.convRepo = repository.NewConversationRepository(db)

	result, err := svc.GetConversations(user.ID)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	// 已存在 session 不应触发任何 INSERT
	got := atomic.LoadInt32(insertCounter)
	assert.Equal(t, int32(0), got, "已存在 session 不应再 INSERT")
}

// TestGetConversations_BatchPreservesSessionFields 验证：
// 批量创建的 session 字段（UserID、ConversationID、LastVisitedAt）正确填充。
func TestGetConversations_BatchPreservesSessionFields(t *testing.T) {
	db := setupConvServiceDB(t)
	svc, _ := newConvServiceWithInsertCounter(t, db)

	user := &model.User{Username: "perfuser3", PasswordHash: "h"}
	db.Create(user)

	convIDs := make([]uint, 0, 3)
	for i := 0; i < 3; i++ {
		conv := &model.Conversation{Type: "single"}
		db.Create(conv)
		db.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         user.ID,
			Role:           "member",
		})
		convIDs = append(convIDs, conv.ID)
	}

	_, err := svc.GetConversations(user.ID)
	assert.NoError(t, err)

	for _, cid := range convIDs {
		var sess model.ConversationSession
		err := db.Where("user_id = ? AND conversation_id = ?", user.ID, cid).First(&sess).Error
		assert.NoError(t, err, "会话 %d 的 session 应被创建", cid)
		assert.Equal(t, user.ID, sess.UserID)
		assert.Equal(t, cid, sess.ConversationID)
		assert.False(t, sess.IsPinned, "新创建的 session 不应被置顶")
		assert.False(t, sess.IsHidden, "新创建的 session 不应被隐藏")
		assert.False(t, sess.LastVisitedAt.IsZero(), "LastVisitedAt 应被填充")
	}
}

// 兼容旧 import 用到的 context（repository.FindByUserID 需要 ctx）
var _ = context.Background
