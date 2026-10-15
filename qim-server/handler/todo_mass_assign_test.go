package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 大范围指派（assignee=all / 指派人数超阈值）的二次确认流：
// AI 一次误判的爆炸半径是全群（几百人 = 几百条任务 + 几百条通知轰炸），
// 超阈值时必须先落待确认记录、由触发者确认后才批量创建。

// setupMassGroup 建 1 个发送者 + n 个群内人类成员，返回 (conv, 成员ID列表)。
func setupMassGroup(t *testing.T, n int) (*model.Conversation, []uint) {
	t.Helper()
	db := setupTodoTestDB(t)
	setupExtractNotification(t, db)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)

	sender := &model.User{Username: "sender", PasswordHash: "x", Nickname: "张三"}
	require.NoError(t, db.Create(sender).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: sender.ID, Role: "member"}).Error)

	memberIDs := make([]uint, 0, n)
	for i := 0; i < n; i++ {
		u := &model.User{Username: "m" + string(rune('a'+i)), PasswordHash: "x", Nickname: "成员" + string(rune('A'+i))}
		require.NoError(t, db.Create(u).Error)
		require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u.ID, Role: "member"}).Error)
		memberIDs = append(memberIDs, u.ID)
	}
	return conv, memberIDs
}

// TestCreateTodo_MassAssignAboveThresholdRequiresConfirmation 问题取证：
// 指派人数超过阈值时，AI 一次判定立即给全群建任务+推通知，误判无法撤回。
// 应先落待确认记录 + 给发送者发确认通知，不直接创建任何任务。
func TestCreateTodo_MassAssignAboveThresholdRequiresConfirmation(t *testing.T) {
	conv, memberIDs := setupMassGroup(t, 11) // 候选 = 11 > 阈值 10
	senderID := uint(1)

	e := NewTodoExtractor(nil)
	e.createTodo(ExtractedTodo{Title: "全员周报", Assignee: "all"}, senderID, conv.ID)

	// 不应直接创建任何任务
	var taskCount int64
	database.GetDB().Model(&model.Task{}).Count(&taskCount)
	assert.Equal(t, int64(0), taskCount, "超阈值指派不应直接创建任务")

	// 应落一条待确认记录：名单=全部人类成员（排除发送者）
	var pends []model.TodoMassAssign
	require.NoError(t, database.GetDB().Find(&pends).Error)
	require.Len(t, pends, 1, "应落一条待确认记录")
	assert.Equal(t, model.TodoMassAssignStatusPending, pends[0].Status)
	assert.Equal(t, senderID, pends[0].CreatorID)
	var pendingIDs []uint
	require.NoError(t, json.Unmarshal([]byte(pends[0].AssigneeIDs), &pendingIDs))
	assert.ElementsMatch(t, memberIDs, pendingIDs, "待确认名单应为全部人类成员（排除发送者）")
	assert.True(t, pends[0].ExpiresAt.After(time.Now()), "应有过期时间")

	// 应给发送者发一条确认通知（只此一条，不广播给全群）
	var notifications []model.Notification
	require.NoError(t, database.GetDB().Where("user_id = ?", senderID).Find(&notifications).Error)
	require.Len(t, notifications, 1, "应只给发送者发一条确认通知")
	assert.Equal(t, "todo_mass_pending", notifications[0].Type)
	assert.Equal(t, "todo_mass_confirm", notifications[0].ActionType)
	require.Contains(t, notifications[0].ActionPayload, "pending_id")

	// 其他成员不应收到任何通知
	var otherNotifs int64
	database.GetDB().Model(&model.Notification{}).Where("user_id != ?", senderID).Count(&otherNotifs)
	assert.Equal(t, int64(0), otherNotifs, "确认前不应打扰其他成员")
}

// TestConfirmTodoMassAssign_CreatesTasksAndNotifies 确认后按锁定名单批量创建
// 任务并逐人通知，记录置 approved。
func TestConfirmTodoMassAssign_CreatesTasksAndNotifies(t *testing.T) {
	db := setupTodoTestDB(t)
	setupExtractNotification(t, db)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "sender", PasswordHash: "x", Nickname: "张三"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "a", PasswordHash: "x", Nickname: "成员A"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 3, Username: "b", PasswordHash: "x", Nickname: "成员B"}).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)

	ids, _ := json.Marshal([]uint{2, 3})
	pend := &model.TodoMassAssign{
		CreatorID:      1,
		ConversationID: conv.ID,
		Title:          "全员周报",
		Priority:       "high",
		AssigneeIDs:    string(ids),
		Status:         model.TodoMassAssignStatusPending,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, db.Create(pend).Error)

	err := confirmTodoMassAssign(db, 1, pend.ID)
	require.NoError(t, err)

	var tasks []model.Task
	require.NoError(t, db.Find(&tasks).Error)
	require.Len(t, tasks, 2, "确认后应为名单内每个成员创建任务")
	for _, tk := range tasks {
		assert.Contains(t, []uint{2, 3}, tk.UserID)
		assert.Equal(t, "全员周报", tk.Title)
	}

	var approved model.TodoMassAssign
	require.NoError(t, db.First(&approved, pend.ID).Error)
	assert.Equal(t, model.TodoMassAssignStatusApproved, approved.Status)

	// 非创建者不能确认
	pend2 := &model.TodoMassAssign{
		CreatorID:      1,
		ConversationID: conv.ID,
		Title:          "第二个",
		AssigneeIDs:    string(ids),
		Status:         model.TodoMassAssignStatusPending,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, db.Create(pend2).Error)
	assert.Error(t, confirmTodoMassAssign(db, 999, pend2.ID), "非创建者确认应被拒绝")
}

// TestConfirmTodoMassAssign_RejectsExpiredAndNonPending 过期或已处理的记录不能确认，
// 过期确认不产生任何任务。
func TestConfirmTodoMassAssign_RejectsExpiredAndNonPending(t *testing.T) {
	db := setupTodoTestDB(t)
	setupExtractNotification(t, db)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "sender", PasswordHash: "x", Nickname: "张三"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "a", PasswordHash: "x", Nickname: "成员A"}).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)

	ids, _ := json.Marshal([]uint{2})
	expired := &model.TodoMassAssign{
		CreatorID: 1, ConversationID: conv.ID, Title: "过期待办", AssigneeIDs: string(ids),
		Status: model.TodoMassAssignStatusPending, ExpiresAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, db.Create(expired).Error)
	done := &model.TodoMassAssign{
		CreatorID: 1, ConversationID: conv.ID, Title: "已处理待办", AssigneeIDs: string(ids),
		Status: model.TodoMassAssignStatusRejected, ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, db.Create(done).Error)

	assert.Error(t, confirmTodoMassAssign(db, 1, expired.ID), "过期记录确认应被拒绝")
	assert.Error(t, confirmTodoMassAssign(db, 1, done.ID), "非 pending 状态确认应被拒绝")

	var taskCount int64
	db.Model(&model.Task{}).Count(&taskCount)
	assert.Equal(t, int64(0), taskCount, "被拒绝的确认不应创建任何任务")

	var expiredRow model.TodoMassAssign
	require.NoError(t, db.First(&expiredRow, expired.ID).Error)
	assert.Equal(t, model.TodoMassAssignStatusExpired, expiredRow.Status, "过期确认应把记录标记为过期")
}

// TestRejectTodoMassAssign 忽略（reject）后记录置 rejected，不创建任务。
func TestRejectTodoMassAssign(t *testing.T) {
	db := setupTodoTestDB(t)
	setupExtractNotification(t, db)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "sender", PasswordHash: "x", Nickname: "张三"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "a", PasswordHash: "x", Nickname: "成员A"}).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)

	ids, _ := json.Marshal([]uint{2})
	pend := &model.TodoMassAssign{
		CreatorID: 1, ConversationID: conv.ID, Title: "误判待办", AssigneeIDs: string(ids),
		Status: model.TodoMassAssignStatusPending, ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, db.Create(pend).Error)

	require.NoError(t, rejectTodoMassAssign(db, 1, pend.ID))

	var row model.TodoMassAssign
	require.NoError(t, db.First(&row, pend.ID).Error)
	assert.Equal(t, model.TodoMassAssignStatusRejected, row.Status)

	var taskCount int64
	db.Model(&model.Task{}).Count(&taskCount)
	assert.Equal(t, int64(0), taskCount, "忽略后不应创建任务")
}
