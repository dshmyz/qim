package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"gorm.io/gorm"
)

func newDecisionLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AvatarDecisionEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAvatarDecisionLog_RecordAndList(t *testing.T) {
	db := newDecisionLogTestDB(t)
	logSvc := NewAvatarDecisionLogService(db)

	if err := logSvc.Record(AvatarDecisionInput{
		OwnerID:        7,
		ConversationID: 11,
		TriggerUserID:  3,
		TriggerName:    "张三",
		MessageSnippet: "这个方案什么时候能定？",
		Decision:       "skipped",
		Stage:          "trigger",
		Reason:         "接管期内",
	}); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := logSvc.Record(AvatarDecisionInput{
		OwnerID:        7,
		ConversationID: 11,
		MessageSnippet: "好的我来处理",
		Decision:       "replied",
		Stage:          "sent",
		Reason:         "关键词触发",
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	events := logSvc.List(7, 0, 100)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	// 倒序：最新一条在前
	if events[0].Decision != "replied" || events[1].Decision != "skipped" {
		t.Fatalf("unexpected order: %s before %s", events[0].Decision, events[1].Decision)
	}

	// 按会话过滤
	if got := logSvc.List(7, 99, 100); len(got) != 0 {
		t.Fatalf("expected no events for other conversation, got %d", len(got))
	}

	// 归属隔离
	if got := logSvc.List(8, 0, 100); len(got) != 0 {
		t.Fatalf("expected no events for other owner, got %d", len(got))
	}
}

func TestAvatarDecisionLog_Prune(t *testing.T) {
	db := newDecisionLogTestDB(t)
	logSvc := NewAvatarDecisionLogService(db)

	for i := 0; i < avatarDecisionRetention+20; i++ {
		if err := logSvc.Record(AvatarDecisionInput{
			OwnerID:  1,
			Decision: "skipped",
			Stage:    "trigger",
		}); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}

	var count int64
	db.Model(&model.AvatarDecisionEvent{}).Where("owner_id = 1").Count(&count)
	if count != avatarDecisionRetention {
		t.Fatalf("expected retention cap %d, got %d", avatarDecisionRetention, count)
	}
}

func TestAvatarDecisionLog_TruncatesLongFields(t *testing.T) {
	db := newDecisionLogTestDB(t)
	logSvc := NewAvatarDecisionLogService(db)

	long := make([]rune, 300)
	for i := range long {
		long[i] = '很'
	}
	if err := logSvc.Record(AvatarDecisionInput{
		OwnerID:        1,
		ConversationID: 1,
		MessageSnippet: string(long),
		Decision:       "skipped",
		Stage:          "trigger",
		Reason:         string(long),
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	events := logSvc.List(1, 0, 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// truncateRunes 截断后追加省略号（99+…、254+…）
	if got := len([]rune(events[0].MessageSnippet)); got != 100 {
		t.Fatalf("expected snippet capped at 99 runes + ellipsis, got %d", got)
	}
	if got := len([]rune(events[0].Reason)); got != 255 {
		t.Fatalf("expected reason capped at 254 runes + ellipsis, got %d", got)
	}
}
