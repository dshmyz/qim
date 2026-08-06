package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAvatarExcluded(t *testing.T) {
	rules := model.AvatarTriggerRules{ExcludedConversations: []uint{3, 7}}
	assert.True(t, IsAvatarExcluded(rules, 3))
	assert.False(t, IsAvatarExcluded(rules, 5))
	assert.False(t, IsAvatarExcluded(model.AvatarTriggerRules{}, 1), "无排除列表时不应排除")
}

func TestIsAvatarInTimeRange(t *testing.T) {
	// 无配置：始终允许
	assert.True(t, IsAvatarInTimeRange(model.AvatarTriggerRules{}))

	now := time.Now()
	today := int(now.Weekday())
	hour := now.Hour()

	// 命中当前星期 + 当前小时
	in := model.AvatarTriggerRules{TimeRanges: []model.AvatarTimeRange{{
		DayOfWeek: []int{today}, StartHour: hour, EndHour: hour,
	}}}
	assert.True(t, IsAvatarInTimeRange(in), "当前星期且小时命中应允许")

	// 命中星期但小时范围不覆盖当前时刻（用 0-0 范围在非 0 点必不相交）
	otherHour := (hour + 1) % 24
	endHour := (hour + 2) % 24
	outHour := model.AvatarTriggerRules{TimeRanges: []model.AvatarTimeRange{{
		DayOfWeek: []int{today}, StartHour: otherHour, EndHour: endHour,
	}}}
	// 仅当 otherHour/endHour 恰好跨越 hour 时才允许；绝大多数情况下应不命中
	if !(otherHour <= hour && hour <= endHour) {
		assert.False(t, IsAvatarInTimeRange(outHour), "小时不命中应禁止")
	}

	// 非当日：全天范围也不应命中
	otherDay := (today + 1) % 7
	outDay := model.AvatarTriggerRules{TimeRanges: []model.AvatarTimeRange{{
		DayOfWeek: []int{otherDay}, StartHour: 0, EndHour: 23,
	}}}
	assert.False(t, IsAvatarInTimeRange(outDay), "非当日应禁止")
}

func TestAvatarLengthHint(t *testing.T) {
	assert.Equal(t, "回复尽量简短，以一句话为主", avatarLengthHint("short"))
	assert.Equal(t, "回复长度适中", avatarLengthHint("medium"))
	assert.Equal(t, "回复可以较详细，控制在 400 字以内", avatarLengthHint("very_long"))
	assert.Equal(t, "回复可以详细，但仍需自然", avatarLengthHint("Long")) // 大小写不敏感
	assert.Equal(t, "回复要简洁，不要过长", avatarLengthHint(""))
	assert.Equal(t, "回复要简洁，不要过长", avatarLengthHint("unknown"))
}

func TestAvatarMaxReplyChars(t *testing.T) {
	assert.Equal(t, 100, avatarMaxReplyChars("short"))
	assert.Equal(t, 300, avatarMaxReplyChars("medium"))
	assert.Equal(t, 400, avatarMaxReplyChars("very_long"))
	assert.Equal(t, 2000, avatarMaxReplyChars("long"))
	assert.Equal(t, 0, avatarMaxReplyChars(""), "未配置不截断")
}

func TestExtractJSONObject(t *testing.T) {
	assert.Equal(t, `{"a":1}`, extractJSONObject(`{"a":1}`))
	assert.Equal(t, `{"should_reply":true}`, extractJSONObject("前缀文字\n```json\n{\"should_reply\":true}\n```"))
	assert.Equal(t, `{"a":1,"b":{"c":2}}`, extractJSONObject(`noise {"a":1,"b":{"c":2}} trailing`))
	assert.Equal(t, "", extractJSONObject("no json here"))
	assert.Equal(t, "", extractJSONObject("{ broken"))
}

func TestAvatarDecideReplyNonSmartModes(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := &AvatarTriggerService{db: db} // aiService 留空：非 smart 模式不触达 LLM

	const avatarUID = uint(1)
	require.NoError(t, db.Create(&model.User{ID: avatarUID, Username: "ava", PasswordHash: "h", Status: "offline"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "snd", PasswordHash: "h", Status: "online"}).Error)

	config := func(rules string) model.AvatarConfig {
		return model.AvatarConfig{UserID: avatarUID, Enabled: true, Name: "我的分身", TriggerRulesJSON: rules}
	}

	cases := []struct {
		name     string
		config   model.AvatarConfig
		convID   uint
		message  string
		isGroup  bool
		mentions []uint
		want     bool
		wantErr  bool
	}{
		{"mention/群/被@命中", config(`{"mode":"mention"}`), 1, "在吗", true, []uint{avatarUID}, true, false},
		{"mention/群/未@", config(`{"mode":"mention"}`), 1, "在吗", true, nil, false, false},
		// 私聊 mention 不再无条件触发：改走智能意图判断，无 AI 服务时 fail-closed 静默，
		// 避免对方随便说一句就触发分身导致乱回复
		{"mention/私聊无AI服务则静默", config(`{"mode":"mention"}`), 1, "在吗", false, nil, false, false},
		{"keyword/命中", config(`{"mode":"keyword","keywords":["请假"]}`), 1, "我请假一天", true, nil, true, false},
		{"keyword/未命中", config(`{"mode":"keyword","keywords":["请假"]}`), 1, "今天天气好", true, nil, false, false},
		{"keyword/大小写不敏感", config(`{"mode":"keyword","keywords":["HELP"]}`), 1, "can anyone help me?", true, nil, true, false},
		{"keyword/无关键词默认不触发", config(`{"mode":"keyword"}`), 1, "任意", true, nil, false, false},
		{"all 模式", config(`{"mode":"all"}`), 1, "任意", true, nil, true, false},
		{"排除会话", config(`{"mode":"all","excludedConversations":[42]}`), 42, "任意", true, nil, false, false},
		{"未启用", model.AvatarConfig{UserID: avatarUID, Enabled: false, TriggerRulesJSON: `{"mode":"all"}`}, 1, "任意", true, nil, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason, err := svc.DecideReply(tc.config, tc.convID, tc.message, "snd", tc.isGroup, tc.mentions)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got, "reason=%s", reason)
		})
	}
}

func TestAvatarDecideReplyOfflineMode(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := &AvatarTriggerService{db: db}

	require.NoError(t, db.Create(&model.User{ID: 1, Username: "offlineu", PasswordHash: "h", Status: "offline"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "onlineu", PasswordHash: "h", Status: "online"}).Error)

	cfg := model.AvatarConfig{UserID: 1, Enabled: true, Name: "分身", TriggerRulesJSON: `{"mode":"offline"}`}
	got, _, err := svc.DecideReply(cfg, 1, "在吗", "snd", true, nil)
	require.NoError(t, err)
	assert.True(t, got, "离线用户应触发")

	cfg.UserID = 2
	got, _, err = svc.DecideReply(cfg, 1, "在吗", "snd", true, nil)
	require.NoError(t, err)
	assert.False(t, got, "在线用户不应触发")
}

func TestAvatarDecideReplyTimeRangeGatesAllModes(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := &AvatarTriggerService{db: db}

	otherDay := (int(time.Now().Weekday()) + 1) % 7
	// all 模式但时间窗落在非今日 → 即便 all 也被时间窗挡下
	cfg := model.AvatarConfig{UserID: 1, Enabled: true, Name: "分身",
		TriggerRulesJSON: mustJSON(t, model.AvatarTriggerRules{
			Mode:       "all",
			TimeRanges: []model.AvatarTimeRange{{DayOfWeek: []int{otherDay}, StartHour: 0, EndHour: 23}},
		})}
	got, reason, err := svc.DecideReply(cfg, 1, "任意", "snd", true, nil)
	require.NoError(t, err)
	assert.False(t, got, "非活跃时间窗应挡下 all 模式; reason=%s", reason)
}

func mustJSON(t *testing.T, v interface{}) string {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// newTriggerSvcConfig 构造一个注入了 fake LLM 的 AvatarTriggerService，
// reply 为 LLMShouldReply 应返回的原文（fake provider 的回应）。
func newTriggerSvcConfig(reply string) (*AvatarTriggerService, model.AvatarConfig) {
	svc := &AvatarTriggerService{
		aiService: newFakeAvatarAIService(reply),
		db:        nil, // 意图判断不依赖 DB
	}
	cfg := model.AvatarConfig{
		UserID: 1, Enabled: true, Name: "我的分身",
		TriggerRulesJSON:  `{"mode":"smart"}`,
		ReplyStrategyJSON: `{"confidenceThreshold":0.8}`, // 显式阈值，供置信度门控用例
	}
	return svc, cfg
}

// TestAvatarDecideReplyIntentFailClosed 覆盖「乱回复」核心防线：意图判断在各种
// 输入下都应 fail-closed——除非 LLM 明确且高置信地判定需要回复，否则一律静默。
func TestAvatarDecideReplyIntentFailClosed(t *testing.T) {
	cases := []struct {
		name    string
		reply   string // fake LLM 返回原文
		overT   float64
		confThr float64 // confidenceThreshold；0 表示不门控
		want    bool
	}{
		// 明确需要回复 + 高置信 → 触发
		{"明确提问高置信触发", `{"should_reply":true,"confidence":0.95,"reason":"在问主人事务"}`, 0, 0.8, true},
		// should_reply=false → 静默（无关闲聊）
		{"无关闲聊静默", `{"should_reply":false,"confidence":0.9,"reason":"寒暄"}`, 0, 0.8, false},
		// should_reply=true 但置信度低于阈值 → 降级不回复（fail-closed）
		{"低置信度低于阈值静默", `{"should_reply":true,"confidence":0.5,"reason":"不确定"}`, 0, 0.8, false},
		// should_reply=true、置信度等于阈值 → 触发
		{"置信度等于阈值触发", `{"should_reply":true,"confidence":0.8,"reason":"确定"}`, 0, 0.8, true},
		// LLM 返回不可解析内容 → fail-closed 静默（不 panic、不触发）
		{"LLM返回非法JSON静默", `抱歉我无法判断`, 0, 0.8, false},
		// markdown 围栏包裹的 JSON → 仍应正确解析（extractJSONObject 兜底）
		{"markdown围栏JSON解析", "```json\n{\"should_reply\":true,\"confidence\":0.9,\"reason\":\"提问\"}\n```", 0, 0.8, true},
		// 未配置置信度阈值（0）→ 不门控，should_reply=true 直接触发
		{"无阈值不门控", `{"should_reply":true,"confidence":0.1,"reason":"低置信"}`, 0.1, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, cfg := newTriggerSvcConfig(tc.reply)
			cfg.ReplyStrategyJSON = mustJSON(t, model.AvatarReplyStrategy{ConfidenceThreshold: tc.confThr})
			got, reason, err := svc.DecideReply(cfg, 1, "今天会议改期到几点", "同事", false, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got, "reason=%s", reason)
		})
	}
}

// TestAvatarDecideReplyIntentMarkdownOrderMatchesSinglePath 确保 intent 判断走的是
// smart 与私聊 mention 共用的同一通道（而非两条各自为政的逻辑），用同一消息断言
// 两种入口结果一致。
func TestAvatarDecideReplyIntentMarkdownOrderMatchesSinglePath(t *testing.T) {
	reply := `{"should_reply":true,"confidence":0.9,"reason":"提问"}`
	svc, cfg := newTriggerSvcConfig(reply)

	// smart 模式（群）
	smartCfg := cfg
	smartCfg.TriggerRulesJSON = `{"mode":"smart"}`
	gotSmart, _, err := svc.DecideReply(smartCfg, 1, "你是不是被裁员了？", "同事", true, nil)
	require.NoError(t, err)

	// 私聊 mention 模式
	mentionCfg := cfg
	mentionCfg.TriggerRulesJSON = `{"mode":"mention"}`
	gotMention, _, err := svc.DecideReply(mentionCfg, 1, "你是不是被裁员了？", "同事", false, nil)
	require.NoError(t, err)

	assert.Equal(t, gotSmart, gotMention, "smart 与私聊 mention 应共用同一意图判断")
}
