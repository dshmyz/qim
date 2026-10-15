package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTodoMassDB(t *testing.T) (*gorm.DB, *ws.Hub) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// 单连接：:memory: 每个连接是独立空库，不限制会让并发测试的 goroutine 看不到表
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Bot{}, &model.Conversation{}, &model.ConversationMember{},
		&model.Message{}, &model.Task{}, &model.Notification{},
		&model.TodoMassAssign{}, &model.CardActionRecord{},
	))
	hub := ws.NewHub(db, "test-secret", "http")
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = nil })
	return db, hub
}

// newTodoMassFixture 建：默认 AI 助手 bot、发送者、成员A、成员B、群会话，
// 返回 (db, svc, bot, convID, creatorID, memberAID, memberBID)。
func newTodoMassFixture(t *testing.T) (*gorm.DB, *BotMessagingService, model.Bot, uint, uint, uint, uint) {
	t.Helper()
	db, hub := setupTodoMassDB(t)

	aiUser := &model.User{Username: "bot_ai_assistant", PasswordHash: "x", Nickname: "AI 助手", Type: "bot"}
	require.NoError(t, db.Create(aiUser).Error)
	bot := model.Bot{Name: "AI 助手", Type: model.BotTypeAssistant, VirtualUserID: &aiUser.ID, IsActive: true}
	require.NoError(t, db.Create(&bot).Error)

	sender := &model.User{Username: "sender", PasswordHash: "x", Nickname: "张三"}
	require.NoError(t, db.Create(sender).Error)
	memberA := &model.User{Username: "a", PasswordHash: "x", Nickname: "成员A"}
	require.NoError(t, db.Create(memberA).Error)
	memberB := &model.User{Username: "b", PasswordHash: "x", Nickname: "成员B"}
	require.NoError(t, db.Create(memberB).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	for _, uid := range []uint{aiUser.ID, sender.ID, memberA.ID, memberB.ID} {
		require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: uid, Role: "member"}).Error)
	}

	svc := NewBotMessagingService(db, hub)
	return db, svc, bot, conv.ID, sender.ID, memberA.ID, memberB.ID
}

// TestSendTodoConfirmCard_PostsCardToGroup 群内确认卡：以默认 AI 助手身份发一张
// 可点击卡片（kind=todo_confirm），带确认/忽略按钮与发起人信息。
func TestSendTodoConfirmCard_PostsCardToGroup(t *testing.T) {
	db, svc, bot, convID, _, _, _ := newTodoMassFixture(t)

	require.NoError(t, svc.SendTodoConfirmCard(convID, bot, 42, "全员周报", 11, "张三"))

	var msg model.Message
	require.NoError(t, db.Where("conversation_id = ? AND type = ?", convID, "card").First(&msg).Error)
	assert.Equal(t, *bot.VirtualUserID, msg.SenderID, "卡片应以默认 AI 助手身份发送")

	var payload struct {
		Kind      string `json:"kind"`
		PendingID uint   `json:"pending_id"`
		Title     string `json:"title"`
		Text      string `json:"text"`
		Buttons   []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
		} `json:"buttons"`
	}
	require.NoError(t, json.Unmarshal([]byte(msg.Content), &payload))
	assert.Equal(t, "todo_confirm", payload.Kind)
	assert.Equal(t, uint(42), payload.PendingID)
	assert.Contains(t, payload.Text, "全员周报")
	assert.Contains(t, payload.Text, "张三", "应带发起人信息（被指派人可见任务来源）")
	assert.Len(t, payload.Buttons, 2)
	assert.Equal(t, "confirm", payload.Buttons[0].ID)
	assert.Equal(t, "cancel", payload.Buttons[1].ID)
}

// TestHandleTodoConfirmCard_ConfirmCreatesTasksAndRewrites 点击确认：
// 按锁定名单批量创建任务、落点击记录与 card_action 气泡、卡片回写终态。
func TestHandleTodoConfirmCard_ConfirmCreatesTasksAndRewrites(t *testing.T) {
	db, svc, bot, convID, creatorID, memberAID, memberBID := newTodoMassFixture(t)

	ids, _ := json.Marshal([]uint{memberAID, memberBID})
	pend := &model.TodoMassAssign{
		CreatorID: creatorID, ConversationID: convID, Title: "全员周报", Priority: "high",
		AssigneeIDs: string(ids), Status: model.TodoMassAssignStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, db.Create(pend).Error)

	cardContent, err := json.Marshal(map[string]interface{}{
		"kind": "todo_confirm", "pending_id": pend.ID,
		"title": "待确认的全员待办", "text": "AI 从群消息提取到「全员周报」",
		"buttons": []map[string]string{
			{"id": "confirm", "text": "确认创建", "style": "primary", "value": "pending:x"},
			{"id": "cancel", "text": "忽略", "style": "secondary", "value": "pending:x"},
		},
	})
	require.NoError(t, err)
	cardMsg := &model.Message{ConversationID: convID, SenderID: *bot.VirtualUserID, Type: "card", Content: string(cardContent)}
	require.NoError(t, db.Create(cardMsg).Error)

	// 创建者本人确认
	require.NoError(t, svc.HandleTodoConfirmCard(cardMsg, creatorID, "confirm"))

	var tasks []model.Task
	require.NoError(t, db.Find(&tasks).Error)
	require.Len(t, tasks, 2, "确认后应为名单内每个成员创建任务")
	for _, tk := range tasks {
		assert.Contains(t, []uint{memberAID, memberBID}, tk.UserID)
		assert.Equal(t, "全员周报", tk.Title)
	}

	var approved model.TodoMassAssign
	require.NoError(t, db.First(&approved, pend.ID).Error)
	assert.Equal(t, model.TodoMassAssignStatusApproved, approved.Status)

	var record model.CardActionRecord
	require.NoError(t, db.Where("message_id = ? AND user_id = ?", cardMsg.ID, creatorID).First(&record).Error)

	var bubble model.Message
	require.NoError(t, db.Where("conversation_id = ? AND type = ?", convID, "card_action").First(&bubble).Error)

	var latest model.Message
	require.NoError(t, db.First(&latest, cardMsg.ID).Error)
	assert.Contains(t, latest.Content, "已确认", "卡片应回写终态")

	// 重复确认：状态机拒绝
	assert.Error(t, svc.HandleTodoConfirmCard(cardMsg, creatorID, "confirm"))
}

// TestHandleTodoConfirmCard_NonMemberRejected 群外用户点击应被拒绝且不产生任务。
func TestHandleTodoConfirmCard_NonMemberRejected(t *testing.T) {
	db, svc, bot, convID, creatorID, memberAID, _ := newTodoMassFixture(t)

	ids, _ := json.Marshal([]uint{memberAID})
	pend := &model.TodoMassAssign{
		CreatorID: creatorID, ConversationID: convID, Title: "全员周报",
		AssigneeIDs: string(ids), Status: model.TodoMassAssignStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, db.Create(pend).Error)

	cardContent, _ := json.Marshal(map[string]interface{}{
		"kind": "todo_confirm", "pending_id": pend.ID, "title": "t", "text": "x",
		"buttons": []map[string]string{{"id": "confirm", "text": "确认创建", "value": "pending:x"}},
	})
	cardMsg := &model.Message{ConversationID: convID, SenderID: *bot.VirtualUserID, Type: "card", Content: string(cardContent)}
	require.NoError(t, db.Create(cardMsg).Error)

	assert.Error(t, svc.HandleTodoConfirmCard(cardMsg, 999, "confirm"), "群外用户点击应被拒绝")

	var taskCount int64
	db.Model(&model.Task{}).Count(&taskCount)
	assert.Equal(t, int64(0), taskCount)
}

// TestConfirmTodoMassAssign_StateMachine 状态机：过期置 expired、非 pending 拒绝、
// 忽略置 rejected，均不创建任务。
func TestConfirmTodoMassAssign_StateMachine(t *testing.T) {
	db, _, _, convID, creatorID, memberAID, _ := newTodoMassFixture(t)

	ids, _ := json.Marshal([]uint{memberAID})
	expired := &model.TodoMassAssign{CreatorID: creatorID, ConversationID: convID, Title: "过期", AssigneeIDs: string(ids),
		Status: model.TodoMassAssignStatusPending, ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, db.Create(expired).Error)
	done := &model.TodoMassAssign{CreatorID: creatorID, ConversationID: convID, Title: "已处理", AssigneeIDs: string(ids),
		Status: model.TodoMassAssignStatusRejected, ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, db.Create(done).Error)
	rejectable := &model.TodoMassAssign{CreatorID: creatorID, ConversationID: convID, Title: "可忽略", AssigneeIDs: string(ids),
		Status: model.TodoMassAssignStatusPending, ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, db.Create(rejectable).Error)

	assert.Error(t, ConfirmTodoMassAssign(db, creatorID, expired.ID))
	assert.Error(t, ConfirmTodoMassAssign(db, creatorID, done.ID))
	assert.Error(t, ConfirmTodoMassAssign(db, 999, rejectable.ID), "非创建者确认应被拒绝")

	var expiredRow model.TodoMassAssign
	require.NoError(t, db.First(&expiredRow, expired.ID).Error)
	assert.Equal(t, model.TodoMassAssignStatusExpired, expiredRow.Status)

	require.NoError(t, RejectTodoMassAssign(db, creatorID, rejectable.ID))
	var rejectedRow model.TodoMassAssign
	require.NoError(t, db.First(&rejectedRow, rejectable.ID).Error)
	assert.Equal(t, model.TodoMassAssignStatusRejected, rejectedRow.Status)

	var taskCount int64
	db.Model(&model.Task{}).Count(&taskCount)
	assert.Equal(t, int64(0), taskCount, "被拒/过期的确认不应创建任务")
}

// ── 评审修复：去重/原子性/双通道同步/伪造卡/空名 ─────────────────────────────

// TestHandleTodoConfirmCard_SyncsNotificationHandled 群卡确认成功后，触发者通知
// 中心那条 todo_mass_pending 通知应同步置 Handled，按钮不再悬空可点。
func TestHandleTodoConfirmCard_SyncsNotificationHandled(t *testing.T) {
	db, svc, bot, convID, creatorID, memberAID, _ := newTodoMassFixture(t)

	ids, _ := json.Marshal([]uint{memberAID})
	pend := &model.TodoMassAssign{CreatorID: creatorID, ConversationID: convID, Title: "全员周报",
		AssigneeIDs: string(ids), Status: model.TodoMassAssignStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, db.Create(pend).Error)
	notification := &model.Notification{UserID: creatorID, Type: "todo_mass_pending",
		Title: "待确认的全员待办", Content: "x", ActionType: "todo_mass_confirm",
		ActionPayload: fmt.Sprintf(`{"pending_id":%d}`, pend.ID)}
	require.NoError(t, db.Create(notification).Error)

	cardContent, _ := json.Marshal(map[string]interface{}{
		"kind": "todo_confirm", "pending_id": pend.ID, "title": "t", "text": "x",
		"buttons": []map[string]string{{"id": "confirm", "text": "确认创建", "value": "pending:x"}},
	})
	cardMsg := &model.Message{ConversationID: convID, SenderID: *bot.VirtualUserID, Type: "card", Content: string(cardContent)}
	require.NoError(t, db.Create(cardMsg).Error)

	require.NoError(t, svc.HandleTodoConfirmCard(cardMsg, creatorID, "confirm"))

	var refreshed model.Notification
	require.NoError(t, db.First(&refreshed, notification.ID).Error)
	assert.True(t, refreshed.Handled, "群卡确认后通知应同步置 Handled")
}

// TestHandleTodoConfirmCard_RejectsForgedCard 伪造卡防护：非默认 AI 助手（如用户自建
// 外部 agent bot）发的 todo_confirm 卡不得触发平台侧确认状态机。
func TestHandleTodoConfirmCard_RejectsForgedCard(t *testing.T) {
	db, svc, _, convID, creatorID, memberAID, _ := newTodoMassFixture(t)

	forger := &model.User{Username: "bot_forge", PasswordHash: "x", Nickname: "伪造Bot", Type: "bot"}
	require.NoError(t, db.Create(forger).Error)
	forgeBot := model.Bot{Name: "伪造Bot", Type: model.BotTypeCustom, CreatorID: 7, VirtualUserID: &forger.ID, IsActive: true}
	require.NoError(t, db.Create(&forgeBot).Error)

	ids, _ := json.Marshal([]uint{memberAID})
	pend := &model.TodoMassAssign{CreatorID: creatorID, ConversationID: convID, Title: "全员周报",
		AssigneeIDs: string(ids), Status: model.TodoMassAssignStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, db.Create(pend).Error)

	cardContent, _ := json.Marshal(map[string]interface{}{
		"kind": "todo_confirm", "pending_id": pend.ID, "title": "t", "text": "x",
		"buttons": []map[string]string{{"id": "confirm", "text": "确认创建", "value": "pending:x"}},
	})
	cardMsg := &model.Message{ConversationID: convID, SenderID: forger.ID, Type: "card", Content: string(cardContent)}
	require.NoError(t, db.Create(cardMsg).Error)

	assert.Error(t, svc.HandleTodoConfirmCard(cardMsg, creatorID, "confirm"), "伪造卡应被拒绝")

	var taskCount int64
	db.Model(&model.Task{}).Count(&taskCount)
	assert.Equal(t, int64(0), taskCount, "伪造卡不得创建任务")
	var still model.TodoMassAssign
	require.NoError(t, db.First(&still, pend.ID).Error)
	assert.Equal(t, model.TodoMassAssignStatusPending, still.Status, "伪造卡不得改动状态")
}

// TestConfirmTodoMassAssign_ConcurrentSingleCreation 并发确认原子性：多人（或群卡+
// 通知中心两通道）同时点击，仅一次抢到 approved 并创建任务，任务数恰为名单人数。
func TestConfirmTodoMassAssign_ConcurrentSingleCreation(t *testing.T) {
	db, _, _, convID, creatorID, memberAID, memberBID := newTodoMassFixture(t)

	ids, _ := json.Marshal([]uint{memberAID, memberBID})
	pend := &model.TodoMassAssign{CreatorID: creatorID, ConversationID: convID, Title: "全员周报",
		AssigneeIDs: string(ids), Status: model.TodoMassAssignStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, db.Create(pend).Error)

	const concurrency = 8
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = ConfirmTodoMassAssign(db, creatorID, pend.ID)
		}(i)
	}
	wg.Wait()

	var tasks []model.Task
	require.NoError(t, db.Find(&tasks).Error)
	assert.Len(t, tasks, 2, "并发确认下任务数应恰为名单人数，不得重复创建")

	var success int
	for _, err := range errs {
		if err == nil {
			success++
		}
	}
	assert.Equal(t, 1, success, "仅一次确认应成功")
}

// TestCreateTasksForAssignees_DeletedCreatorFallback 指派人查询失败（如已注销）时
// 文案不得渲染出空名字。
func TestCreateTasksForAssignees_DeletedCreatorFallback(t *testing.T) {
	db, _, _, convID, _, memberAID, _ := newTodoMassFixture(t)

	ghost := &model.User{Username: "ghost", PasswordHash: "x", Nickname: "幽灵"}
	require.NoError(t, db.Create(ghost).Error)
	// 硬删除：模拟记录彻底不存在（软删除用户 Unscoped 仍可查到真名，是更好的行为）
	require.NoError(t, db.Unscoped().Delete(ghost).Error)

	CreateTasksForAssignees(db, MassTaskSpec{Title: "交接任务", Priority: "medium"},
		ghost.ID, convID, []uint{memberAID})

	var notification model.Notification
	require.NoError(t, db.Where("user_id = ? AND type = ?", memberAID, "todo_assigned").First(&notification).Error)
	assert.NotContains(t, notification.Content, "由  指派", "不得渲染空名字")
	assert.Contains(t, notification.Content, "未知成员", "查询失败应回退占位名")
}

// TestSyncTodoConfirmCardTerminal_RewritesCard 通知中心通道确认/忽略后，群内确认卡
// 应同步回写终态，按钮不再悬空。
func TestSyncTodoConfirmCardTerminal_RewritesCard(t *testing.T) {
	db, svc, bot, convID, _, _, _ := newTodoMassFixture(t)

	ids, _ := json.Marshal([]uint{3})
	pend := &model.TodoMassAssign{CreatorID: 2, ConversationID: convID, Title: "全员周报",
		AssigneeIDs: string(ids), Status: model.TodoMassAssignStatusApproved,
		ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, db.Create(pend).Error)

	cardContent, _ := json.Marshal(map[string]interface{}{
		"kind": "todo_confirm", "pending_id": pend.ID, "title": "t", "text": "原始文案",
		"buttons": []map[string]string{{"id": "confirm", "text": "确认创建", "value": "pending:x"}},
	})
	cardMsg := &model.Message{ConversationID: convID, SenderID: *bot.VirtualUserID, Type: "card", Content: string(cardContent)}
	require.NoError(t, db.Create(cardMsg).Error)

	svc.SyncTodoConfirmCardTerminal(convID, pend.ID, "✓ 已确认创建")

	var latest model.Message
	require.NoError(t, db.First(&latest, cardMsg.ID).Error)
	assert.Contains(t, latest.Content, "✓ 已确认创建", "群卡应同步回写终态")
}
