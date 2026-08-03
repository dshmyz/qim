package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelServiceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.ChannelSubscriber{},
	))
	return db
}

// active 频道可用
func TestEnsureChannelUsable_Active(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	svc := NewChannelService(db)

	ch := &model.Channel{Name: "已激活频道", Status: model.ChannelStatusActive}
	require.NoError(t, db.Create(ch).Error)

	err := svc.EnsureChannelUsable(ch)
	assert.NoError(t, err)
}

// pending 频道不可用：审批中不能发消息
func TestEnsureChannelUsable_Pending(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	svc := NewChannelService(db)

	ch := &model.Channel{Name: "审批中频道", Status: model.ChannelStatusPending}
	require.NoError(t, db.Create(ch).Error)

	err := svc.EnsureChannelUsable(ch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "审批中")
}

// rejected 频道不可用：被拒绝的频道不能继续发消息
func TestEnsureChannelUsable_Rejected(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	svc := NewChannelService(db)

	ch := &model.Channel{Name: "已拒绝频道", Status: model.ChannelStatusRejected}
	require.NoError(t, db.Create(ch).Error)

	err := svc.EnsureChannelUsable(ch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "拒绝")
}

// inactive 频道不可用：管理员停用的频道不能发消息
func TestEnsureChannelUsable_Inactive(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	svc := NewChannelService(db)

	ch := &model.Channel{Name: "停用频道", Status: "inactive"}
	require.NoError(t, db.Create(ch).Error)

	err := svc.EnsureChannelUsable(ch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "停用")
}

// 未知状态不可用
func TestEnsureChannelUsable_UnknownStatus(t *testing.T) {
	db := setupChannelServiceTestDB(t)
	svc := NewChannelService(db)

	ch := &model.Channel{Name: "未知状态频道", Status: "weird"}
	require.NoError(t, db.Create(ch).Error)

	err := svc.EnsureChannelUsable(ch)
	assert.Error(t, err)
}
