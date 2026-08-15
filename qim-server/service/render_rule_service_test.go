package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRenderRuleTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.SystemConfig{}))
	return db
}

// 合法的 Jira 规则样本
func validJiraRule() RenderRule {
	return RenderRule{
		ID:       "jira_ticket",
		Name:     "Jira 工单卡片化",
		Enabled:  true,
		Priority: 10,
		Scope: RenderScope{
			Groups:            []string{"*"},
			ExcludeGroups:     []string{},
			ConversationTypes: []string{"single", "group", "discussion"},
		},
		Match: RenderMatch{
			Pattern:       `\b([A-Z]{2,6})-(\d{1,6})\b`,
			Flags:         "g",
			CaptureGroups: map[string]int{"project": 1, "number": 2},
		},
		Render: RenderConfig{
			Type:          "link_card",
			URLTemplate:   "http://jira.xxx.com/{{project}}/{{project}}-{{number}}",
			LabelTemplate: "{{project}}-{{number}}",
			TitleTemplate: "查看 Jira 工单 {{project}}-{{number}}",
			Icon:          "fab fa-jira",
			Target:        "_blank",
			Class:         "jira-ticket-card",
		},
	}
}

// 合法规则应通过校验
func TestValidateRule_ValidJiraRule(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	err := svc.validateRule(validJiraRule())
	assert.NoError(t, err)
}

// 嵌套量词（ReDoS 风险）应被拒
func TestValidateRule_ReDoSPattern(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Match.Pattern = `(a+)+`
	err := svc.validateRule(rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ReDoS")
}

// javascript: 协议应被拒
func TestValidateRule_JavascriptURL(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Render.URLTemplate = "javascript:alert(1)"
	err := svc.validateRule(rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "javascript")
}

// 编译失败的正则应被拒
func TestValidateRule_InvalidRegex(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Match.Pattern = `[unclosed`
	err := svc.validateRule(rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "正则编译失败")
}

// class 含空格（注入事件处理器）应被拒
func TestValidateRule_ClassInjection(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Render.Class = "evil onclick=alert(1)"
	err := svc.validateRule(rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "class")
}

// URL 不以 http(s) 或 / 开头应被拒
func TestValidateRule_RelativeURLMustStartWithSlash(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Render.URLTemplate = "ftp://example.com/x"
	err := svc.validateRule(rule)
	assert.Error(t, err)
}

// 相对路径 / 开头应通过
func TestValidateRule_RelativePathURL(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Render.URLTemplate = "/docs/{{number}}"
	err := svc.validateRule(rule)
	assert.NoError(t, err)
}

// 非法的 render.type 应被拒
func TestValidateRule_InvalidRenderType(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Render.Type = "evil_iframe"
	err := svc.validateRule(rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render.type")
}

// 规则 ID 为空应被拒
func TestValidateRule_EmptyID(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.ID = ""
	err := svc.validateRule(rule)
	assert.Error(t, err)
}

// 优先级越界应被拒
func TestValidateRule_PriorityOutOfRange(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()
	rule.Priority = 0
	err := svc.validateRule(rule)
	assert.Error(t, err)
}

// SaveRules 覆盖式保存
func TestSaveRules_Overwrite(t *testing.T) {
	db := setupRenderRuleTestDB(t)
	svc := NewRenderRuleService(db)

	rules := []RenderRule{validJiraRule()}
	require.NoError(t, svc.SaveRules(rules))

	// 覆盖为两条
	rules2 := []RenderRule{
		{ID: "rule_a", Name: "A", Enabled: true, Priority: 10,
			Match:  RenderMatch{Pattern: `A-(\d+)`, CaptureGroups: map[string]int{"n": 1}},
			Render: RenderConfig{Type: "link", URLTemplate: "/a/{{n}}", LabelTemplate: "A-{{n}}"}},
		{ID: "rule_b", Name: "B", Enabled: false, Priority: 20,
			Match:  RenderMatch{Pattern: `B-(\d+)`, CaptureGroups: map[string]int{"n": 1}},
			Render: RenderConfig{Type: "link", URLTemplate: "/b/{{n}}", LabelTemplate: "B-{{n}}"}},
	}
	require.NoError(t, svc.SaveRules(rules2))

	// GetAllRules 应返回 2 条
	got, err := svc.GetAllRules()
	require.NoError(t, err)
	assert.Equal(t, 2, len(got))
}

// SaveRules 后缓存失效，下次 GetRules 拿到新数据
func TestSaveRules_InvalidatesCache(t *testing.T) {
	db := setupRenderRuleTestDB(t)
	svc := NewRenderRuleService(db)

	// 首次保存 + 读取
	require.NoError(t, svc.SaveRules([]RenderRule{validJiraRule()}))
	r1, err := svc.GetRules()
	require.NoError(t, err)
	assert.Equal(t, 1, len(r1))

	// 覆盖保存为 2 条
	require.NoError(t, svc.SaveRules([]RenderRule{
		validJiraRule(),
		{ID: "x", Name: "X", Enabled: true, Priority: 5,
			Match:  RenderMatch{Pattern: `X-(\d+)`, CaptureGroups: map[string]int{"n": 1}},
			Render: RenderConfig{Type: "link", URLTemplate: "/x/{{n}}", LabelTemplate: "X-{{n}}"}},
	}))

	// 重新读取应拿到 2 条（缓存已失效）
	r2, err := svc.GetRules()
	require.NoError(t, err)
	assert.Equal(t, 2, len(r2))
}

// GetRules 无配置时返回空切片不报错
func TestGetRules_EmptyConfig(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rules, err := svc.GetRules()
	require.NoError(t, err)
	assert.Equal(t, 0, len(rules))
}

// GetRules 二次调用命中缓存（不重复查 DB）
func TestGetRules_CacheHit(t *testing.T) {
	db := setupRenderRuleTestDB(t)
	svc := NewRenderRuleService(db)
	require.NoError(t, svc.SaveRules([]RenderRule{validJiraRule()}))

	r1, err := svc.GetRules()
	require.NoError(t, err)
	v1, _ := svc.GetVersion()

	// 直接改 DB 模拟缓存未失效（绕过 SaveRules）
	db.Model(&model.SystemConfig{}).Where("config_key = ?", "render_rules").
		Update("value", "[]")

	// 二次调用应命中缓存，仍返回 1 条
	r2, err := svc.GetRules()
	require.NoError(t, err)
	assert.Equal(t, len(r1), len(r2), "缓存命中应返回相同结果")
	v2, _ := svc.GetVersion()
	assert.Equal(t, v1, v2, "版本号应不变")
}

// TestRule 对 Jira 样例文本应正确匹配
func TestTestRule_JiraSample(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()

	results, err := svc.TestRule(rule, "看一下 NI-30000 这个工单")
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	assert.Equal(t, "NI-30000", results[0].Matched)
	assert.Equal(t, "http://jira.xxx.com/NI/NI-30000", results[0].URL)
	assert.Equal(t, "NI-30000", results[0].Label)
}

// TestRule 应匹配多个工单号
func TestTestRule_MultipleMatches(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()

	results, err := svc.TestRule(rule, "NI-30000 和 AP-3000 都看一下")
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "NI-30000", results[0].Matched)
	assert.Equal(t, "http://jira.xxx.com/NI/NI-30000", results[0].URL)
	assert.Equal(t, "AP-3000", results[1].Matched)
	assert.Equal(t, "http://jira.xxx.com/AP/AP-3000", results[1].URL)
}

// TestRule 对不匹配的文本应返回空结果
func TestTestRule_NoMatch(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rule := validJiraRule()

	results, err := svc.TestRule(rule, "今天天气不错")
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

// SaveRules 任一规则非法应整体拒绝
func TestSaveRules_InvalidRuleRejectsAll(t *testing.T) {
	svc := NewRenderRuleService(setupRenderRuleTestDB(t))
	rules := []RenderRule{
		validJiraRule(),
		{ID: "bad", Name: "Bad", Priority: 10,
			Match:  RenderMatch{Pattern: `[unclosed`},
			Render: RenderConfig{Type: "link", URLTemplate: "/x"}},
	}
	err := svc.SaveRules(rules)
	assert.Error(t, err)
}

// GetVersion 保存后应更新
func TestGetVersion_AfterSave(t *testing.T) {
	db := setupRenderRuleTestDB(t)
	svc := NewRenderRuleService(db)

	v0, err := svc.GetVersion()
	require.NoError(t, err)
	assert.Equal(t, int64(0), v0, "初始版本号应为 0")

	require.NoError(t, svc.SaveRules([]RenderRule{validJiraRule()}))
	v1, err := svc.GetVersion()
	require.NoError(t, err)
	assert.Greater(t, v1, int64(0), "保存后版本号应大于 0")
}
