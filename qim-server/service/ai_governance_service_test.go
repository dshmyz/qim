package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"gorm.io/gorm"
)

func newGovernanceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIReplyEvent{},
		&model.AIMessageFeedback{},
		&model.AILearningSignal{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestGovernanceMetrics_Aggregation(t *testing.T) {
	db := newGovernanceTestDB(t)
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	events := []model.AIReplyEvent{
		{Scene: ReplySceneBotDM, Outcome: "completed", ConversationID: 1, CreatedAt: now},
		{Scene: ReplySceneBotDM, Outcome: "completed", ConversationID: 1, CreatedAt: now},
		{Scene: ReplySceneBotDM, Outcome: "interrupted", Reason: "length", ConversationID: 1, CreatedAt: now},
		{Scene: ReplySceneGroupMention, Outcome: "completed", ConversationID: 2, CreatedAt: now},
		{Scene: ReplySceneGroupAuto, Outcome: "not_triggered", ConversationID: 2, CreatedAt: now},
		{Scene: ReplySceneGroupAuto, Outcome: "not_triggered", ConversationID: 2, CreatedAt: now},
		// 窗口外：8 天前，不应计入 7 天窗口
		{Scene: ReplySceneBotDM, Outcome: "completed", ConversationID: 1, CreatedAt: now.AddDate(0, 0, -8)},
		{Scene: ReplySceneBotDM, Outcome: "interrupted", Reason: "no_done", ConversationID: 1, CreatedAt: yesterday},
	}
	for i := range events {
		if err := db.Create(&events[i]).Error; err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	feedbacks := []model.AIMessageFeedback{
		{UserID: 1, MessageID: 10, Rating: 1, CreatedAt: now},
		{UserID: 2, MessageID: 11, Rating: -1, ReasonCategory: "off_topic", CreatedAt: now},
		{UserID: 3, MessageID: 12, Rating: -1, ReasonCategory: "hallucinated", CreatedAt: now},
	}
	for i := range feedbacks {
		if err := db.Create(&feedbacks[i]).Error; err != nil {
			t.Fatalf("fb seed %d: %v", i, err)
		}
	}
	db.Create(&model.AILearningSignal{OwnerID: 1, ConversationID: 1, SignalType: SignalTypeDraftEdit, CreatedAt: now})

	svc := NewAIGovernanceService(db)
	m, err := svc.GetMetrics(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if m.WindowDays != 7 {
		t.Fatalf("days=%d", m.WindowDays)
	}

	var bot SceneGovernance
	for _, s := range m.Scenes {
		if s.Scene == ReplySceneBotDM {
			bot = s
		}
	}
	if bot.Total != 4 {
		t.Fatalf("bot total should be 4 (window excludes -8d), got %d", bot.Total)
	}
	if bot.Completed != 2 || bot.Interrupted != 2 {
		t.Fatalf("bot split: %+v", bot)
	}
	if bot.CompletionRate != 0.5 {
		t.Fatalf("rate=%v", bot.CompletionRate)
	}

	var auto SceneGovernance
	for _, s := range m.Scenes {
		if s.Scene == ReplySceneGroupAuto {
			auto = s
		}
	}
	if auto.NotTriggered != 2 {
		t.Fatalf("auto not_triggered=%d", auto.NotTriggered)
	}
	if auto.CompletionRate != 0 {
		t.Fatalf("auto with no completed/interrupted must have 0 rate, got %v", auto.CompletionRate)
	}

	// 中断原因：length 1 + no_done 1
	reasons := map[string]int64{}
	for _, r := range m.InterruptReasons {
		reasons[r.Name] = r.Count
	}
	if reasons["length"] != 1 || reasons["no_done"] != 1 {
		t.Fatalf("interrupt reasons: %+v", reasons)
	}

	// 反馈
	if m.Feedback.Total != 3 || m.Feedback.Up != 1 || m.Feedback.Down != 2 {
		t.Fatalf("feedback: %+v", m.Feedback)
	}
	dr := map[string]int64{}
	for _, r := range m.Feedback.DownReasons {
		dr[r.Name] = r.Count
	}
	if dr["off_topic"] != 1 || dr["hallucinated"] != 1 {
		t.Fatalf("down reasons: %+v", dr)
	}

	// 学习信号
	foundSignal := false
	for _, s := range m.Signals {
		if s.Name == SignalTypeDraftEdit && s.Count == 1 {
			foundSignal = true
		}
	}
	if !foundSignal {
		t.Fatalf("signals: %+v", m.Signals)
	}

	// 每日趋势包含完成数
	if len(m.DailyTrend) == 0 {
		t.Fatal("daily trend empty")
	}
}

func TestGovernanceMetrics_InvalidDaysDefault(t *testing.T) {
	db := newGovernanceTestDB(t)
	m, err := NewAIGovernanceService(db).GetMetrics(500)
	if err != nil {
		t.Fatal(err)
	}
	if m.WindowDays != 7 {
		t.Fatalf("expected clamp to 7, got %d", m.WindowDays)
	}
}
