package service

import (
	"strings"
	"testing"
)

// reflectConsolidated 在 aiService 为 nil 时应安全降级：
// evaluateRemember 返回 ShouldRemember=false（不落库），summary 用确定性兜底生成，不 panic。
func TestReflectConsolidated_NilAI_NoPanic(t *testing.T) {
	ref, verdict, err := reflectConsolidated(nil, "我们决定周五下午开例会", []string{"之前约定每周五例会"}, []string{})
	if err != nil {
		t.Fatalf("nil aiService 不应报错: %v", err)
	}
	if verdict.ShouldRemember {
		t.Error("nil aiService 判定应默认不记忆")
	}
	if strings.TrimSpace(ref.Summary) == "" {
		t.Error("summary 应有确定性兜底值")
	}
	if ref.Importance != 3 {
		t.Errorf("默认重要度应为 3，got %v", ref.Importance)
	}
}

func TestReflectConsolidated_MergesMemory(t *testing.T) {
	// 有既有记忆时，确定性兜底 summary 应包含该记忆（折叠合并的原材料）
	ref, _, _ := reflectConsolidated(nil, "新消息", []string{"旧记忆A", "旧记忆A"}, []string{"知识1"})
	if !strings.Contains(ref.Summary, "旧记忆A") {
		t.Errorf("summary 应包含既有记忆，got %q", ref.Summary)
	}
	if !strings.Contains(ref.Summary, "知识1") {
		t.Errorf("summary 应包含知识片段，got %q", ref.Summary)
	}
}

func TestParseReflectionJSON(t *testing.T) {
	s := `{"summary":"关于项目进度的总结","facts":["A","B"],"themes":["工作"]}`
	ref, ok := parseReflectionJSON(s)
	if !ok {
		t.Fatal("应能解析合法 JSON")
	}
	if ref.Summary != "关于项目进度的总结" {
		t.Errorf("summary 解析错误: %q", ref.Summary)
	}
	if len(ref.Facts) != 2 || len(ref.Themes) != 1 {
		t.Errorf("facts/themes 解析错误: %+v", ref)
	}
}

func TestParseReflectionJSON_Entities(t *testing.T) {
	// 反射结构化里应能解析出实体/主题（供知识图谱聚合），这是「记忆实体图谱」的数据来源。
	s := `{"summary":"小明决定每周五下午开项目例会","facts":["每周五例会"],"themes":["项目","会议"],"entities":["小明"]}`
	ref, ok := parseReflectionJSON(s)
	if !ok {
		t.Fatal("应能解析合法 JSON")
	}
	if len(ref.Entities) != 1 || ref.Entities[0] != "小明" {
		t.Errorf("entities 解析错误: %+v", ref.Entities)
	}
	if len(ref.Themes) != 2 {
		t.Errorf("themes 解析错误: %+v", ref.Themes)
	}
	if ref.Summary == "" {
		t.Error("summary 不应为空")
	}
}

func TestReflectionExtractPrompt_ComposesContext(t *testing.T) {
	// 结构化反射提示应包含消息与既有记忆/知识，便于 LLM 折叠合并。
	p := reflectionExtractPrompt("我们周五开会", []string{"旧记忆"}, []string{"知识片段"})
	if !strings.Contains(p, "旧记忆") {
		t.Error("反射提示应包含既有记忆")
	}
	if !strings.Contains(p, "知识片段") {
		t.Error("反射提示应包含知识片段")
	}
	if !strings.Contains(p, "我们周五开会") {
		t.Error("反射提示应包含当前消息")
	}
}

func TestParseReflectionJSON_Invalid(t *testing.T) {
	if _, ok := parseReflectionJSON("不是 JSON"); ok {
		t.Error("非法输入不应解析成功")
	}
}

func TestTruncateForSummary(t *testing.T) {
	long := strings.Repeat("长", 200)
	if got := truncateForSummary(long); len([]rune(got)) > 121 {
		t.Errorf("超长文本应截断，len=%d", len([]rune(got)))
	}
	if got := truncateForSummary("短文本"); got != "短文本" {
		t.Errorf("短文本不应截断，got %q", got)
	}
}
