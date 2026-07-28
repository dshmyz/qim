package handler

import (
	"fmt"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupGroupManagementDB 建内存 DB + 种子：群「测试群」，含群主/管理员/普通成员/被踢者。
func setupGroupManagementDB(t *testing.T) (db *gorm.DB, owner, admin, member, victim *model.User, convID uint) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Conversation{}, &model.ConversationMember{}, &model.Group{}))
	database.DB = db

	users := []*model.User{
		{Username: "owner", Nickname: "群主"},
		{Username: "admin", Nickname: "管理员"},
		{Username: "member", Nickname: "普通成员"},
		{Username: "victim", Nickname: "被踢者"},
	}
	for _, u := range users {
		require.NoError(t, db.Create(u).Error)
	}
	owner, admin, member, victim = users[0], users[1], users[2], users[3]

	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.Group{
		ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: owner.ID,
	}).Error)

	for _, m := range []struct {
		uid  uint
		role string
	}{
		{owner.ID, "owner"}, {admin.ID, "admin"}, {member.ID, "member"}, {victim.ID, "member"},
	} {
		require.NoError(t, db.Create(&model.ConversationMember{
			ConversationID: conv.ID, UserID: m.uid, Role: m.role, JoinedAt: time.Now(),
		}).Error)
	}
	return db, owner, admin, member, victim, conv.ID
}

func memberExists(db *gorm.DB, convID, userID uint) bool {
	var count int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ? AND user_id = ?", convID, userID).Count(&count)
	return count > 0
}

func newTestMCPServer() *ai.MCPServer {
	srv := ai.NewMCPServer(false, nil)
	srv.RegisterTool(&GroupManagementTool{})
	return srv
}

// TestGroupManagementTool_RemoveMemberByOwner 群主经 AI 工具踢人，被踢者真实从 DB 移除。
func TestGroupManagementTool_RemoveMemberByOwner(t *testing.T) {
	db, owner, _, _, victim, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	result, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action":           "remove_member",
		"group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier":  victim.Username,
	}, &ai.CallerContext{UserID: owner.ID})

	require.NoError(t, err)
	assert.Equal(t, "success", result.(map[string]interface{})["result"])
	assert.False(t, memberExists(db, convID, victim.ID), "被踢者应已从群成员移除")
}

// TestGroupManagementTool_RemoveMemberByAdmin 管理员也能踢。
func TestGroupManagementTool_RemoveMemberByAdmin(t *testing.T) {
	db, _, admin, _, victim, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action":           "remove_member",
		"group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier":  victim.Username,
	}, &ai.CallerContext{UserID: admin.ID})

	require.NoError(t, err)
	assert.False(t, memberExists(db, convID, victim.ID))
}

// TestGroupManagementTool_RejectByPlainMember 普通成员让 AI 踢人被拒，被踢者仍在群。
func TestGroupManagementTool_RejectByPlainMember(t *testing.T) {
	db, _, _, member, victim, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action":           "remove_member",
		"group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier":  victim.Username,
	}, &ai.CallerContext{UserID: member.ID})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "权限不足")
	assert.True(t, memberExists(db, convID, victim.ID), "权限不足时被踢者不应被移除")
}

// TestGroupManagementTool_UserIDZeroBypassesPermission UserID=0 跳过权限校验直接执行移除，
// 演示修复前 smart_reply_graph.go:122 写死 userID=0 的权限绕过隐患。
// 修复后 @AI 走 ExecuteWithTools 用真实 input.UserID，不会出现 UserID=0。
func TestGroupManagementTool_UserIDZeroBypassesPermission(t *testing.T) {
	db, _, _, _, victim, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action":           "remove_member",
		"group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier":  victim.Username,
	}, &ai.CallerContext{UserID: 0})

	require.NoError(t, err)
	assert.False(t, memberExists(db, convID, victim.ID),
		"UserID=0 绕过权限执行了移除——这正是修复要堵的隐患；修复后 @AI 链路用真实 userID，此路径不再可达")
}

// TestGroupManagementTool_AddMemberByOwner 群主经 AI 工具加人，新成员真实入群。
func TestGroupManagementTool_AddMemberByOwner(t *testing.T) {
	db, owner, _, _, _, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	newUser := model.User{Username: "newbie", Nickname: "新人"}
	require.NoError(t, db.Create(&newUser).Error)

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action":           "add_member",
		"group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier":  newUser.Username,
	}, &ai.CallerContext{UserID: owner.ID})

	require.NoError(t, err)
	assert.True(t, memberExists(db, convID, newUser.ID), "新人应已入群")
}

// TestGroupManagementTool_MuteByOwner 群主禁言成员，MutedUntil 字段写入；解除后清除。
func TestGroupManagementTool_MuteByOwner(t *testing.T) {
	db, owner, _, _, victim, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action":           "mute",
		"group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier":  victim.Username,
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)

	var m model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", convID, victim.ID).First(&m).Error)
	require.NotNil(t, m.MutedUntil, "被禁言者应有 MutedUntil")
	assert.True(t, m.MutedUntil.After(time.Now()), "MutedUntil 应在未来（24h）")

	// unmute 清除
	_, err = srv.ExecuteTool("group_management", map[string]interface{}{
		"action":           "unmute",
		"group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier":  victim.Username,
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)
	// 用新变量查询：gorm First 复用已有变量时不会把 *time.Time 的 NULL 覆盖回 nil
	var mAfter model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", convID, victim.ID).First(&mAfter).Error)
	assert.Nil(t, mAfter.MutedUntil, "解除禁言后 MutedUntil 应为 nil")
}

// TestGroupManagementTool_SetRoleByOwner 群主设置/取消管理员。
func TestGroupManagementTool_SetRoleByOwner(t *testing.T) {
	db, owner, _, _, victim, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action": "set_role", "group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier": victim.Username, "role": "admin",
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)
	var m model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", convID, victim.ID).First(&m).Error)
	assert.Equal(t, "admin", m.Role)

	// 取消管理员
	_, err = srv.ExecuteTool("group_management", map[string]interface{}{
		"action": "set_role", "group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier": victim.Username, "role": "member",
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)
	var m2 model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", convID, victim.ID).First(&m2).Error)
	assert.Equal(t, "member", m2.Role)
}

// TestGroupManagementTool_TransferOwner 群主转让群主。
func TestGroupManagementTool_TransferOwner(t *testing.T) {
	db, owner, admin, _, _, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action": "transfer_owner", "group_identifier": fmt.Sprintf("%d", convID),
		"user_identifier": admin.Username,
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)

	var newOwner model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", convID, admin.ID).First(&newOwner).Error)
	assert.Equal(t, "owner", newOwner.Role)
	var oldOwner model.ConversationMember
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", convID, owner.ID).First(&oldOwner).Error)
	assert.Equal(t, "member", oldOwner.Role)
	var g model.Group
	require.NoError(t, db.Where("conversation_id = ?", convID).First(&g).Error)
	assert.Equal(t, admin.ID, g.CreatorID)
}

// TestGroupManagementTool_UpdateAnnouncement 群主修改群公告。
func TestGroupManagementTool_UpdateAnnouncement(t *testing.T) {
	db, owner, _, _, _, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	_, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action": "update_announcement", "group_identifier": fmt.Sprintf("%d", convID),
		"announcement": "新公告内容",
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)
	var g model.Group
	require.NoError(t, db.Where("conversation_id = ?", convID).First(&g).Error)
	assert.Equal(t, "新公告内容", g.Announcement)
}

// TestGroupManagementTool_ListMembers 查询群成员列表（4 人）。
func TestGroupManagementTool_ListMembers(t *testing.T) {
	_, _, _, _, _, convID := setupGroupManagementDB(t)
	srv := newTestMCPServer()

	result, err := srv.ExecuteTool("group_management", map[string]interface{}{
		"action": "list_members", "group_identifier": fmt.Sprintf("%d", convID),
	}, &ai.CallerContext{UserID: 0})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, 4, m["count"])
}

// ===== C: 新增群聊工具测试 =====

func setupToolsTestDB(t *testing.T) (*gorm.DB, *model.User, uint) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Conversation{}, &model.ConversationMember{}, &model.Group{},
		&model.Task{}, &model.Message{},
	))
	database.DB = db
	owner := model.User{Username: "owner", Nickname: "群主"}
	require.NoError(t, db.Create(&owner).Error)
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: owner.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "owner", JoinedAt: time.Now()}).Error)
	return db, &owner, conv.ID
}

func TestCreateTaskTool(t *testing.T) {
	db, owner, convID := setupToolsTestDB(t)
	srv := ai.NewMCPServer(false, nil)
	srv.RegisterTool(&CreateTaskTool{})

	result, err := srv.ExecuteTool("create_task", map[string]interface{}{
		"group_identifier":    fmt.Sprintf("%d", convID),
		"title":               "明天开会",
		"assignee_identifier": owner.Username,
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)
	assert.Equal(t, "success", result.(map[string]interface{})["result"])

	var task model.Task
	require.NoError(t, db.First(&task).Error)
	assert.Equal(t, "明天开会", task.Title)
	assert.Equal(t, owner.ID, task.UserID)
	assert.Equal(t, "todo", task.Status)
}

func TestSearchMessagesTool(t *testing.T) {
	db, owner, convID := setupToolsTestDB(t)
	require.NoError(t, db.Create(&model.Message{ConversationID: convID, SenderID: owner.ID, Content: "今天天气不错"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: convID, SenderID: owner.ID, Content: "明天开会讨论"}).Error)

	srv := ai.NewMCPServer(false, nil)
	srv.RegisterTool(&SearchMessagesTool{})
	result, err := srv.ExecuteTool("search_messages", map[string]interface{}{
		"group_identifier": fmt.Sprintf("%d", convID),
		"keyword":          "开会",
	}, &ai.CallerContext{UserID: owner.ID})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, 1, m["count"], "应只搜到1条含\"开会\"的消息")
}

// TestGroupSummaryTool_MissingGroupIdentifier 缺少 group_identifier 返回错误（不依赖 di 全局状态，避免与其他测试的 di 初始化竞争）。
func TestGroupSummaryTool_MissingGroupIdentifier(t *testing.T) {
	srv := ai.NewMCPServer(false, nil)
	srv.RegisterTool(&GroupSummaryTool{})

	_, err := srv.ExecuteTool("group_summary", map[string]interface{}{}, &ai.CallerContext{UserID: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "群组不存在")
}
