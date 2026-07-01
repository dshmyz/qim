package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupApprovalTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Approval{},
		&model.AvatarConfig{},
		&model.Bot{},
		&model.Channel{},
		&model.Group{},
		&model.Conversation{},
		&model.ChannelSubscriber{},
	))

	if database.D == nil {
		database.D = database.NewSQLiteDialect()
	}
	return db
}

// TestCreateApproval_WritesSnapshot 验证 CreateApproval 创建审批时写入快照字段
func TestCreateApproval_WritesSnapshot(t *testing.T) {
	db := setupApprovalTestDB(t)
	svc := NewApprovalService(db)

	// 准备申请人和 Bot
	applicant := &model.User{Username: "applicant", Nickname: "张三", PasswordHash: "x", Avatar: "/a.png"}
	require.NoError(t, db.Create(applicant).Error)

	bot := &model.Bot{Name: "工作助手", Description: "日常问答机器人", Type: model.BotTypeCustom}
	require.NoError(t, db.Create(bot).Error)

	// 创建审批
	err := svc.CreateApproval(model.ApprovalTypeBot, bot.ID, applicant.ID)
	require.NoError(t, err)

	// 查出审批记录，验证快照字段
	var approval model.Approval
	require.NoError(t, db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeBot, bot.ID).
		First(&approval).Error)

	assert.Equal(t, "工作助手", approval.TargetName)
	assert.Equal(t, "日常问答机器人", approval.TargetDescription)
	assert.Equal(t, "张三", approval.CreatorName)
	assert.Equal(t, "/a.png", approval.CreatorAvatar)
	assert.NotEmpty(t, approval.ExtraJSON, "ExtraJSON 应包含 bot_type")
	assert.Contains(t, approval.ExtraJSON, "bot_type")
}

// TestCreateApproval_AvatarSnapshot 验证 Avatar 类型审批的快照
func TestCreateApproval_AvatarSnapshot(t *testing.T) {
	db := setupApprovalTestDB(t)
	svc := NewApprovalService(db)

	applicant := &model.User{Username: "u1", Nickname: "李四", PasswordHash: "x"}
	require.NoError(t, db.Create(applicant).Error)

	config := &model.AvatarConfig{UserID: applicant.ID, Name: "我的分身"}
	require.NoError(t, db.Create(config).Error)

	err := svc.CreateApproval(model.ApprovalTypeAvatar, config.ID, applicant.ID)
	require.NoError(t, err)

	var approval model.Approval
	require.NoError(t, db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeAvatar, config.ID).
		First(&approval).Error)

	assert.Equal(t, "我的分身", approval.TargetName)
	assert.Equal(t, "用户分身功能", approval.TargetDescription)
	assert.Equal(t, "李四", approval.CreatorName)
}

// TestCreateApproval_ChannelSnapshot 验证 Channel 类型审批的快照
func TestCreateApproval_ChannelSnapshot(t *testing.T) {
	db := setupApprovalTestDB(t)
	svc := NewApprovalService(db)

	applicant := &model.User{Username: "u2", Nickname: "王五", PasswordHash: "x"}
	require.NoError(t, db.Create(applicant).Error)

	channel := &model.Channel{Name: "技术频道", Description: "技术分享", PublishPermission: "creator_only"}
	require.NoError(t, db.Create(channel).Error)

	err := svc.CreateApproval(model.ApprovalTypeChannel, channel.ID, applicant.ID)
	require.NoError(t, err)

	var approval model.Approval
	require.NoError(t, db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeChannel, channel.ID).
		First(&approval).Error)

	assert.Equal(t, "技术频道", approval.TargetName)
	assert.Equal(t, "技术分享", approval.TargetDescription)
	assert.Contains(t, approval.ExtraJSON, "channel_id")
	assert.Contains(t, approval.ExtraJSON, "publish_permission")
}

// TestCreateApproval_GroupAISnapshot 验证 GroupAI 类型审批的快照
func TestCreateApproval_GroupAISnapshot(t *testing.T) {
	db := setupApprovalTestDB(t)
	svc := NewApprovalService(db)

	applicant := &model.User{Username: "u3", Nickname: "赵六", PasswordHash: "x"}
	require.NoError(t, db.Create(applicant).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)

	group := &model.Group{
		ConversationID: conv.ID,
		GroupType:      "group",
		Name:           "项目群",
		CreatorID:      applicant.ID,
		AIConfigJSON:   `{"enabled":true,"assistant_name":"小助手"}`,
	}
	require.NoError(t, db.Create(group).Error)

	err := svc.CreateApproval(model.ApprovalTypeGroupAI, group.ID, applicant.ID)
	require.NoError(t, err)

	var approval model.Approval
	require.NoError(t, db.Where("target_type = ? AND target_id = ?", model.ApprovalTypeGroupAI, group.ID).
		First(&approval).Error)

	assert.Equal(t, "项目群", approval.TargetName)
	assert.Equal(t, "群聊AI助手", approval.TargetDescription)
	assert.Contains(t, approval.ExtraJSON, "group_id")
	assert.Contains(t, approval.ExtraJSON, "assistant_name")
}

// TestListAllApprovals_SingleTablePagination 验证 listAllApprovals 使用单表分页，
// 不回查源表，返回的 item 包含快照字段
func TestListAllApprovals_SingleTablePagination(t *testing.T) {
	db := setupApprovalTestDB(t)
	svc := NewApprovalService(db)

	applicant := &model.User{Username: "u", Nickname: "申请人", PasswordHash: "x", Avatar: "/av.png"}
	require.NoError(t, db.Create(applicant).Error)

	// 创建 3 条 Bot 审批 + 2 条 Avatar 审批，共 5 条
	for i := 0; i < 3; i++ {
		bot := &model.Bot{Name: "Bot", Description: "desc", Type: model.BotTypeCustom}
		require.NoError(t, db.Create(bot).Error)
		require.NoError(t, svc.CreateApproval(model.ApprovalTypeBot, bot.ID, applicant.ID))
	}
	for i := 0; i < 2; i++ {
		extraUser := &model.User{Username: "au" + string(rune('A'+i)), Nickname: "申请人", PasswordHash: "x", Avatar: "/av.png"}
		require.NoError(t, db.Create(extraUser).Error)
		config := &model.AvatarConfig{UserID: extraUser.ID, Name: "分身"}
		require.NoError(t, db.Create(config).Error)
		require.NoError(t, svc.CreateApproval(model.ApprovalTypeAvatar, config.ID, applicant.ID))
	}

	// 第 1 页，pageSize=3，应返回 3 条，total=5
	items, total, err := svc.ListApprovals("all", "", 1, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, items, 3)

	// 验证返回的 item 包含快照字段（不需要回查源表）
	for _, item := range items {
		assert.NotEmpty(t, item.Name, "Name 应来自快照")
		assert.NotEmpty(t, item.CreatorName, "CreatorName 应来自快照")
		assert.Equal(t, "申请人", item.CreatorName)
		assert.Equal(t, "/av.png", item.CreatorAvatar)
	}

	// 第 2 页，应返回 2 条
	items2, _, err := svc.ListApprovals("all", "", 2, 3)
	require.NoError(t, err)
	assert.Len(t, items2, 2)
}

// TestListAllApprovals_StatusFilter 验证状态过滤
func TestListAllApprovals_StatusFilter(t *testing.T) {
	db := setupApprovalTestDB(t)
	svc := NewApprovalService(db)

	applicant := &model.User{Username: "u", Nickname: "申请人", PasswordHash: "x"}
	require.NoError(t, db.Create(applicant).Error)

	// 创建 2 条 pending + 1 条 approved
	for i := 0; i < 2; i++ {
		bot := &model.Bot{Name: "Bot", Type: model.BotTypeCustom}
		require.NoError(t, db.Create(bot).Error)
		require.NoError(t, svc.CreateApproval(model.ApprovalTypeBot, bot.ID, applicant.ID))
	}
	bot3 := &model.Bot{Name: "Bot3", Type: model.BotTypeCustom}
	require.NoError(t, db.Create(bot3).Error)
	require.NoError(t, svc.CreateApproval(model.ApprovalTypeBot, bot3.ID, applicant.ID))
	// 把第三条改成 approved
	require.NoError(t, db.Model(&model.Approval{}).
		Where("target_id = ?", bot3.ID).
		Updates(map[string]interface{}{"status": model.ApprovalStatusApproved}).Error)

	// 过滤 pending，应返回 2 条
	items, total, err := svc.ListApprovals("all", "pending", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
	for _, item := range items {
		assert.Equal(t, "pending", item.ApprovalStatus)
	}
}
