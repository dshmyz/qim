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

// 确认/忽略状态机与群内确认卡点击链路的测试已搬到 service/todo_mass_assign_test.go
//（ConfirmTodoMassAssign / RejectTodoMassAssign / HandleTodoConfirmCard 收编到 service 层，
// 群卡点击与通知中心动作分发共用同一状态机）。此处保留提取侧的阈值分流测试。

// TestCreateTodo_MassAssignDedup 同一待办被多条消息反复提取时，pending/通知/确认卡
// 只落第一份（任务级幂等在确认后才生效，拦不住确认前的重复卡片轰炸）。
func TestCreateTodo_MassAssignDedup(t *testing.T) {
	conv, _ := setupMassGroup(t, 11) // 候选 = 11 > 阈值 10
	senderID := uint(1)

	e := NewTodoExtractor(nil)
	e.createTodo(ExtractedTodo{Title: "全员周报", Assignee: "all"}, senderID, conv.ID)
	e.createTodo(ExtractedTodo{Title: "全员周报", Assignee: "all"}, senderID, conv.ID)

	var pendCount int64
	database.GetDB().Model(&model.TodoMassAssign{}).Count(&pendCount)
	assert.Equal(t, int64(1), pendCount, "重复提取不应新建第二条待确认记录")

	var notifCount int64
	database.GetDB().Model(&model.Notification{}).Where("type = ?", "todo_mass_pending").Count(&notifCount)
	assert.Equal(t, int64(1), notifCount, "重复提取不应重复发确认通知")

	// 不同标题是不同待办，正常落第二条
	e.createTodo(ExtractedTodo{Title: "另一件事", Assignee: "all"}, senderID, conv.ID)
	database.GetDB().Model(&model.TodoMassAssign{}).Count(&pendCount)
	assert.Equal(t, int64(2), pendCount, "不同标题的待办不受去重影响")
}

// TestCreateTodo_MassAssignSkipWhenApprovedAndTasksAlive 已确认过的同名待办：
// 名单内全部成员仍有未完成任务时，重复提取只是重申，不再发新确认卡；
// 一旦有人完成（名单不再全员命中），允许重新走确认流程补任务。
func TestCreateTodo_MassAssignSkipWhenApprovedAndTasksAlive(t *testing.T) {
	conv, _ := setupMassGroup(t, 11)
	senderID := uint(1)

	e := NewTodoExtractor(nil)
	// 第一次提取 + 模拟已确认且任务已创建
	e.createTodo(ExtractedTodo{Title: "全员周报", Assignee: "all"}, senderID, conv.ID)
	var pend model.TodoMassAssign
	require.NoError(t, database.GetDB().First(&pend).Error)
	database.GetDB().Model(&pend).Update("status", model.TodoMassAssignStatusApproved)
	var ids []uint
	require.NoError(t, json.Unmarshal([]byte(pend.AssigneeIDs), &ids))
	for _, uid := range ids {
		require.NoError(t, database.GetDB().Create(&model.Task{
			UserID: uid, ConversationID: conv.ID, Title: "全员周报", Status: "todo",
		}).Error)
	}

	// 全员任务仍在：重复提取应跳过
	e.createTodo(ExtractedTodo{Title: "全员周报", Assignee: "all"}, senderID, conv.ID)
	var pendCount int64
	database.GetDB().Model(&model.TodoMassAssign{}).Count(&pendCount)
	assert.Equal(t, int64(1), pendCount, "已确认且任务全员在，重复提取不应新建确认流程")

	// 有人完成：允许重新确认（给漏掉的人补任务）
	database.GetDB().Model(&model.Task{}).Where("user_id = ?", ids[0]).Update("status", "done")
	e.createTodo(ExtractedTodo{Title: "全员周报", Assignee: "all"}, senderID, conv.ID)
	database.GetDB().Model(&model.TodoMassAssign{}).Count(&pendCount)
	assert.Equal(t, int64(2), pendCount, "名单不再全员命中时应允许重新确认")
}
