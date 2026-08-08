package service

import (
	"strings"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSegmentHistory 验证上下文污染诊断的统计口径：
// “自身回复”按窗口切分为近期（多轮指代锚点）与远期（自我复制污染源），
// 并支持统计“分身路径单独过滤”掉的自身回复条数。步骤 2 的窗口化逻辑复用同一口径，
// 此测试锁死测量仪器的行为，防止后续改动静默改坏统计。
func TestSegmentHistory(t *testing.T) {
	now := time.Now()

	newMsg := func(origin, content string, ago time.Duration) model.Message {
		return model.Message{Origin: origin, Content: content, CreatedAt: now.Add(-ago)}
	}

	nearSelf := newMsg("assistant", "近期助手回复", 1*time.Minute)
	farSelf := newMsg("assistant", "远期助手回复", 30*time.Minute)
	nearUser := newMsg("", "近期用户消息", 2*time.Minute)
	farUser := newMsg("", "远期用户消息", time.Hour)
	filteredAvatar := newMsg("avatar", "被滤分身回复", 3*time.Minute)

	msgs := []model.Message{nearSelf, farSelf, nearUser, farUser, filteredAvatar}

	seg := segmentHistory(msgs, nil)

	if seg.Total != 5 {
		t.Fatalf("Total = %d, want 5", seg.Total)
	}
	if seg.SelfCount != 3 { // 2 assistant + 1 avatar
		t.Fatalf("SelfCount = %d, want 3", seg.SelfCount)
	}
	// 近期窗口内：nearSelf(1m) + filteredAvatar(3m)；远期：farSelf(30m)
	if seg.NearSelf != 2 {
		t.Fatalf("NearSelf = %d, want 2", seg.NearSelf)
	}
	if seg.FarSelf != 1 {
		t.Fatalf("FarSelf = %d, want 1", seg.FarSelf)
	}
	// token 估算 = chars/3
	expectToken := 0
	for _, m := range msgs {
		expectToken += len(m.Content) / 3
	}
	if seg.tokenEst != expectToken {
		t.Fatalf("tokenEst = %d, want %d", seg.tokenEst, expectToken)
	}
}

// TestSegmentHistoryFiltered 验证分身路径：单独把「分身自己回复」滤掉时能统计到被滤条数，
// 供诊断“分身潜在多轮失忆”（它刚说的话是否被历史排除）。
func TestSegmentHistoryFiltered(t *testing.T) {
	now := time.Now()
	msgs := []model.Message{
		{Origin: "avatar", Content: "分身回复A", CreatedAt: now.Add(-2 * time.Minute)},
		{Origin: "", Content: "用户追问B", CreatedAt: now.Add(-1 * time.Minute)},
	}
	// 记录被滤计数：avatar 回复会被分身历史过滤排除
	filtered := func(m model.Message) bool { return m.Origin == "avatar" }

	seg := segmentHistory(msgs, filtered)

	if seg.filteredOut != 1 {
		t.Fatalf("filteredOut = %d, want 1", seg.filteredOut)
	}
	if seg.NearSelf != 1 {
		t.Fatalf("NearSelf = %d, want 1（分身回复仍在窗口内，本应保留作多轮锚点）", seg.NearSelf)
	}
}

// TestSegmentHistoryNearFarBounds 窗口近/远边界：窗口内记近期、窗口外记远期。
// 用“略小于窗口”与“略大于窗口”各构造一条，避免依赖 time.Now() 的精确相等（时钟漂移会让
// 恰好等于窗口的消息在统计时略微超出，产生瞬时 flake）。
func TestSegmentHistoryNearFarBounds(t *testing.T) {
	now := time.Now()
	msgs := []model.Message{
		{Origin: "assistant", Content: "窗内", CreatedAt: now.Add(-selfTurnWindow + time.Minute)}, // 略小于窗口 → 近期
		{Origin: "assistant", Content: "窗外", CreatedAt: now.Add(-selfTurnWindow - time.Minute)}, // 略大于窗口 → 远期
	}
	seg := segmentHistory(msgs, nil)
	if seg.NearSelf != 1 {
		t.Fatalf("NearSelf = %d, want 1", seg.NearSelf)
	}
	if seg.FarSelf != 1 {
		t.Fatalf("FarSelf = %d, want 1", seg.FarSelf)
	}
}

// TestIsNearSelf 验证近期/远期自身回复的判定：assistant 与 avatar 都算自身，仅窗口内算近期。
func TestIsNearSelf(t *testing.T) {
	now := time.Now()
	within := model.Message{Origin: "assistant", CreatedAt: now.Add(-2 * time.Minute)}
	beyond := model.Message{Origin: "assistant", CreatedAt: now.Add(-selfTurnWindow - time.Minute)}
	avatarNear := model.Message{Origin: "avatar", CreatedAt: now.Add(-2 * time.Minute)}
	user := model.Message{Origin: "", CreatedAt: now.Add(-time.Minute)}

	if !isNearSelf(within) {
		t.Fatalf("窗口内 assistant 应判近期")
	}
	if isNearSelf(beyond) {
		t.Fatalf("窗口外 assistant 不应判近期")
	}
	if !isNearSelf(avatarNear) {
		t.Fatalf("分身回复同样应视为自身回复（多轮锚点）")
	}
	if isNearSelf(user) {
		t.Fatalf("用户消息不算自身回复")
	}
	if !isSelf(within) || !isSelf(beyond) || !isSelf(avatarNear) {
		t.Fatalf("assistant/avatar 都应算自身回复（不分近远期）")
	}
}

// TestFoldFarSelf 验证远期自身回复折叠：逐条截断、合计截断、空输入兜底。
func TestFoldFarSelf(t *testing.T) {
	if got := foldFarSelf(nil); got != "（无有效内容）" {
		t.Fatalf("空输入 = %q, want 兜底", got)
	}

	long := strings.Repeat("甲", 200)
	out := foldFarSelf([]string{long})
	if len([]rune(out)) >= 160 {
		t.Fatalf("折叠结果应被合计截断在 160 rune 内，got len=%d", len([]rune(out)))
	}

	out2 := foldFarSelf([]string{"方案A可行", "方案B更优"})
	if !strings.Contains(out2, "方案A可行") || !strings.Contains(out2, "方案B更优") {
		t.Fatalf("多条约远期应合并折叠，got=%q", out2)
	}
}

// TestBuildHistoryMessagesWindowed 端到端验证群助手历史组装的分层行为：
// 近期自身回复保留为 assistant 轮次（多轮锚点），远期自身回复不逐条回灌，
// 而是折叠成一句追加到 system prompt（避免自我复制）。
func TestBuildHistoryMessagesWindowed(t *testing.T) {
	// 依赖全局 database.GetDB() 的组装路径：临时把全局 DB 指向内存库。
	db := setupServiceTestDB(t)
	old := database.DB
	t.Cleanup(func() { database.DB = old })
	database.DB = db

	require.NoError(t, db.Create(&model.Conversation{ID: 30, Type: "group"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 7, Username: "u7", Nickname: "甲"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 8, Username: "u8", Nickname: "乙"}).Error)

	now := time.Now()
	gmsgs := []model.Message{
		{ConversationID: 30, SenderID: 7, Type: "text", Content: "远期用户消息", CreatedAt: now.Add(-30 * time.Minute)},
		{ConversationID: 30, SenderID: 8, Type: "text", Origin: "assistant", Content: "远期助手回复-方案A可行", CreatedAt: now.Add(-25 * time.Minute)},
		{ConversationID: 30, SenderID: 7, Type: "text", Content: "近期用户消息", CreatedAt: now.Add(-5 * time.Minute)},
		{ConversationID: 30, SenderID: 8, Type: "text", Origin: "assistant", Content: "近期助手回复-对的", CreatedAt: now.Add(-2 * time.Minute)},
	}
	for i := range gmsgs {
		require.NoError(t, db.Create(&gmsgs[i]).Error)
	}

	sg := &SmartReplyGraph{}
	in := &SmartReplyContext{ConversationID: 30, UserID: 7, Message: "那第二个方案呢？"}
	msgs := sg.buildHistoryMessages(in)

	systemContent := msgs[0].Content
	assert.Contains(t, systemContent, "远期助手回复-方案A可行", "远期自身回复应折叠进 system note（保留可索引、避免盲从）")
	assert.Contains(t, systemContent, "更早", "system note 应明确标注是历史记录")

	// 近期自身回复应作为独立的 assistant 轮次出现（多轮指代锚点）。
	assistantTurnFound := false
	for _, m := range msgs[1:] {
		if m.Role == schema.Assistant && m.Content == "近期助手回复-对的" {
			assistantTurnFound = true
		}
	}
	assert.True(t, assistantTurnFound, "近期自身回复应作为 assistant 轮次保留")

	// 远期自身回复不应作为独立 assistant 轮次回灌。
	for _, m := range msgs[1:] {
		if m.Role == schema.Assistant && strings.Contains(m.Content, "远期助手回复") {
			t.Fatalf("远期自身回复不应作为独立 assistant 轮次回灌: %q", m.Content)
		}
	}
}
