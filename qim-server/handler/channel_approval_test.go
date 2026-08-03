package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelApprovalTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.ChannelSubscriber{},
		&model.Approval{},
	))
	return db
}

// pending 频道仅创建者可见，active 频道所有人可见
func TestGetChannels_PendingVisibleOnlyToCreator(t *testing.T) {
	db := setupChannelApprovalTestDB(t)

	creator := model.User{Username: "creator", Type: "user"}
	require.NoError(t, db.Create(&creator).Error)
	other := model.User{Username: "other", Type: "user"}
	require.NoError(t, db.Create(&other).Error)

	pendingCh := model.Channel{Name: "审批中频道", CreatorID: creator.ID, Status: model.ChannelStatusPending}
	require.NoError(t, db.Create(&pendingCh).Error)
	activeCh := model.Channel{Name: "已激活频道", CreatorID: creator.ID, Status: model.ChannelStatusActive}
	require.NoError(t, db.Create(&activeCh).Error)
	rejectedCh := model.Channel{Name: "已拒绝频道", CreatorID: creator.ID, Status: model.ChannelStatusRejected}
	require.NoError(t, db.Create(&rejectedCh).Error)

	// 创建者视角：能看到自己的 pending + active + rejected
	var creatorChannels []model.Channel
	db.Where("status = ? OR creator_id = ?", model.ChannelStatusActive, creator.ID).Find(&creatorChannels)
	assert.Equal(t, 3, len(creatorChannels), "创建者应看到自己的全部频道")

	// 其他用户视角：只能看到 active
	var otherChannels []model.Channel
	db.Where("status = ? OR creator_id = ?", model.ChannelStatusActive, other.ID).Find(&otherChannels)
	assert.Equal(t, 1, len(otherChannels), "其他用户只能看到 active 频道")
	assert.Equal(t, "已激活频道", otherChannels[0].Name)
}

// 频道状态常量值正确
func TestChannelStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", model.ChannelStatusPending)
	assert.Equal(t, "active", model.ChannelStatusActive)
	assert.Equal(t, "rejected", model.ChannelStatusRejected)
}

// approveChannel 应将 pending 频道状态改为 active（模拟事务逻辑）
func TestApproveChannel_StatusTransition(t *testing.T) {
	db := setupChannelApprovalTestDB(t)

	user := model.User{Username: "creator", Type: "user"}
	require.NoError(t, db.Create(&user).Error)

	channel := model.Channel{Name: "测试频道", CreatorID: user.ID, Status: model.ChannelStatusPending}
	require.NoError(t, db.Create(&channel).Error)

	approval := model.Approval{
		TargetType: model.ApprovalTypeChannel,
		TargetID:   channel.ID,
		Status:     model.ApprovalStatusPending,
		AppliedBy:  user.ID,
	}
	require.NoError(t, db.Create(&approval).Error)

	// 模拟 approveChannel 的核心事务逻辑
	tx := db.Begin()
	require.NoError(t, tx.Model(&approval).Updates(map[string]interface{}{
		"status":      model.ApprovalStatusApproved,
		"approved_by": user.ID,
	}).Error)
	require.NoError(t, tx.Model(&model.Channel{}).Where("id = ?", channel.ID).
		Update("status", model.ChannelStatusActive).Error)
	tx.Commit()

	// 验证频道状态已变为 active
	var updated model.Channel
	db.First(&updated, channel.ID)
	assert.Equal(t, model.ChannelStatusActive, updated.Status, "审批通过后频道状态应为 active")
}

// rejectChannel 应将 pending 频道状态改为 rejected（模拟事务逻辑）
func TestRejectChannel_StatusTransition(t *testing.T) {
	db := setupChannelApprovalTestDB(t)

	user := model.User{Username: "creator", Type: "user"}
	require.NoError(t, db.Create(&user).Error)

	channel := model.Channel{Name: "测试频道", CreatorID: user.ID, Status: model.ChannelStatusPending}
	require.NoError(t, db.Create(&channel).Error)

	approval := model.Approval{
		TargetType: model.ApprovalTypeChannel,
		TargetID:   channel.ID,
		Status:     model.ApprovalStatusPending,
		AppliedBy:  user.ID,
	}
	require.NoError(t, db.Create(&approval).Error)

	// 模拟 rejectChannel 的核心事务逻辑
	tx := db.Begin()
	require.NoError(t, tx.Model(&approval).Updates(map[string]interface{}{
		"status":        model.ApprovalStatusRejected,
		"reject_reason": "名称不合规",
		"approved_by":   user.ID,
	}).Error)
	require.NoError(t, tx.Model(&model.Channel{}).Where("id = ?", channel.ID).
		Update("status", model.ChannelStatusRejected).Error)
	tx.Commit()

	// 验证频道状态已变为 rejected
	var updated model.Channel
	db.First(&updated, channel.ID)
	assert.Equal(t, model.ChannelStatusRejected, updated.Status, "审批拒绝后频道状态应为 rejected")
}

// rejected 频道修改后重新提交审批：状态改回 pending
func TestReapplyChannel_StatusTransition(t *testing.T) {
	db := setupChannelApprovalTestDB(t)

	user := model.User{Username: "creator", Type: "user"}
	require.NoError(t, db.Create(&user).Error)

	channel := model.Channel{Name: "旧名称", CreatorID: user.ID, Status: model.ChannelStatusRejected}
	require.NoError(t, db.Create(&channel).Error)

	approval := model.Approval{
		TargetType: model.ApprovalTypeChannel,
		TargetID:   channel.ID,
		Status:     model.ApprovalStatusRejected,
		AppliedBy:  user.ID,
	}
	require.NoError(t, db.Create(&approval).Error)

	// 模拟 UpdateMyChannel 的重新申请逻辑
	require.NoError(t, db.Model(&channel).Updates(map[string]interface{}{
		"name":   "新名称",
		"status": model.ChannelStatusPending,
	}).Error)
	require.NoError(t, db.Model(&approval).Updates(map[string]interface{}{
		"status":        model.ApprovalStatusPending,
		"target_name":   "新名称",
		"reject_reason": "",
		"approved_at":   nil,
		"approved_by":   nil,
	}).Error)

	// 验证频道状态已改回 pending
	var updatedCh model.Channel
	db.First(&updatedCh, channel.ID)
	assert.Equal(t, model.ChannelStatusPending, updatedCh.Status)
	assert.Equal(t, "新名称", updatedCh.Name)

	// 验证审批记录已重置
	var updatedApp model.Approval
	db.First(&updatedApp, approval.ID)
	assert.Equal(t, model.ApprovalStatusPending, updatedApp.Status)
	assert.Equal(t, "新名称", updatedApp.TargetName)
}

// active 频道不应被编辑（UpdateMyChannel 仅允许 pending/rejected）
func TestUpdateMyChannel_RejectsActiveChannel(t *testing.T) {
	db := setupChannelApprovalTestDB(t)

	user := model.User{Username: "creator", Type: "user"}
	require.NoError(t, db.Create(&user).Error)

	channel := model.Channel{Name: "已激活", CreatorID: user.ID, Status: model.ChannelStatusActive}
	require.NoError(t, db.Create(&channel).Error)

	// 模拟 UpdateMyChannel 的状态检查
	var loaded model.Channel
	db.Where("id = ? AND creator_id = ?", channel.ID, user.ID).First(&loaded)
	canEdit := loaded.Status == model.ChannelStatusPending || loaded.Status == model.ChannelStatusRejected
	assert.False(t, canEdit, "active 频道不应允许编辑")
}
