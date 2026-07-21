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
	assert.Equal(t, "回复可以详细，但仍需自然", avatarLengthHint("Long")) // 大小写不敏感
	assert.Equal(t, "回复要简洁，不要过长", avatarLengthHint(""))
	assert.Equal(t, "回复要简洁，不要过长", avatarLengthHint("unknown"))
}

func TestAvatarMaxReplyChars(t *testing.T) {
	assert.Equal(t, 100, avatarMaxReplyChars("short"))
	assert.Equal(t, 300, avatarMaxReplyChars("medium"))
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
		{"mention/私聊自动触发", config(`{"mode":"mention"}`), 1, "在吗", false, nil, true, false},
		{"keyword/命中", config(`{"mode":"keyword","keywords":["请假"]}`), 1, "我请假一天", true, nil, true, false},
		{"keyword/未命中", config(`{"mode":"keyword","keywords":["请假"]}`), 1, "今天天气好", true, nil, false, false},
		{"keyword/大小写不敏感", config(`{"mode":"keyword","keywords":["HELP"]}`), 1, "can anyone help me?", true, nil, true, false},
		{"keyword/无关键词默认触发", config(`{"mode":"keyword"}`), 1, "任意", true, nil, true, false},
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
