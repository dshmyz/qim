package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_ParseCron_Presets 验证预设表达式仍可解析
func Test_ParseCron_Presets(t *testing.T) {
	cases := []string{"@hourly", "@daily", "@midnight", "@weekly", "@monthly"}
	for _, sched := range cases {
		t.Run(sched, func(t *testing.T) {
			res, err := ParseCron(sched)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Positive(t, res.NextRun, "NextRun 应为正数")
			assert.Positive(t, res.Interval, "Interval 应为正数")
		})
	}
}

// Test_ParseCron_EveryKeywords 验证 every N minutes/hours 关键字
func Test_ParseCron_EveryKeywords(t *testing.T) {
	res, err := ParseCron("every 30 minutes")
	require.NoError(t, err)
	assert.Positive(t, res.NextRun)
	// every 30 minutes → 30 分钟间隔
	assert.InDelta(t, 30*time.Minute, res.Interval, float64(time.Second))

	res, err = ParseCron("every 2 hours")
	require.NoError(t, err)
	assert.InDelta(t, 2*time.Hour, res.Interval, float64(time.Second))
}

// Test_ParseCron_Standard5Field_Regression
// 关键回归测试：管理员后台提示 "Cron 表达式，如 0 2 * * * 表示每天凌晨2点"，
// 但原实现把所有 5 段式字符串都当成 24 小时间隔（无视真实语义）。
// 重构后应真正按 cron 语义计算下次触发时刻。
func Test_ParseCron_Standard5Field_Regression(t *testing.T) {
	// 0 2 * * * = 每天凌晨 2 点
	res, err := ParseCron("0 2 * * *")
	require.NoError(t, err)
	require.NotNil(t, res)

	now := time.Now()
	nextRunAt := now.Add(res.NextRun)

	// 下次触发应在明天的 02:00（如果还没到今天 02:00 则今天 02:00）
	assert.Equal(t, 2, nextRunAt.Hour(), "下次触发时刻应在 2 点")
	assert.Equal(t, 0, nextRunAt.Minute(), "下次触发时刻应在 02:00")
}

// Test_ParseCron_Standard5Field_Every5Minutes
func Test_ParseCron_Standard5Field_Every5Minutes(t *testing.T) {
	// */5 * * * * = 每 5 分钟
	res, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)
	assert.Positive(t, res.NextRun)
	assert.Less(t, res.NextRun, 6*time.Minute, "*/5 的下次触发应在 5 分钟内")
}

// Test_ParseCron_EmptyRejected
func Test_ParseCron_EmptyRejected(t *testing.T) {
	_, err := ParseCron("")
	assert.Error(t, err)
}

// Test_ParseCron_InvalidExpressionRejected
// 重构后非法 cron 表达式应被拒绝（原实现是 len>=5 && spaces==4 → 24h，掩盖错误）
func Test_ParseCron_InvalidExpressionRejected(t *testing.T) {
	_, err := ParseCron("not a cron")
	assert.Error(t, err, "非法表达式应被拒绝，而不是当成 24h 间隔")
}
