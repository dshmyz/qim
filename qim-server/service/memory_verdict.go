package service

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
)

// RememberVerdict 是 LLM 对"某段内容是否值得记忆并记多重要"的判定结果。
type RememberVerdict struct {
	ShouldRemember bool    `json:"should_remember"`
	Importance     float64 `json:"importance"` // 1-5
}

// evaluateRemember 用 LLM 判断内容是否值得记，并给出重要度（1-5）。
//
// 返回的 verdict.Importance 为用户可读的 1-5 档位；调用方落库时应换算成 [0,1]
// （gracedb 的 importanceScore = clamp01(record.Importance)）。
//
// 兼容旧逻辑：LLM 只回 true/false 时也能正确解析（importance 取默认 3）。
func evaluateRemember(aiService *ai.AIService, taskPrompt string, message string) (RememberVerdict, error) {
	verdict := RememberVerdict{ShouldRemember: false, Importance: 3}
	if aiService == nil {
		return verdict, nil
	}

	prompt := taskPrompt + `

请以 JSON 返回，形如 {"remember": true, "importance": 3}。
- remember: 是否值得记忆
- importance: 1(极不重要) 到 5(极重要) 的整数档位

内容：` + message

	aiMessages := []ai.Message{{Role: "user", Content: prompt}}
	result, err := aiService.GetCompletion(ai.TaskTypeAnalysis, aiMessages)
	if err != nil {
		return verdict, err
	}

	// 优先尝试结构化解析
	if v, ok := parseRememberVerdictJSON(result); ok {
		return v, nil
	}

	// 兜底：旧式仅 true/false 解析
	lower := strings.ToLower(strings.TrimSpace(result))
	verdict.ShouldRemember = strings.Contains(lower, "true") && !strings.Contains(lower, "false")
	return verdict, nil
}

var verdictJSONRe = regexp.MustCompile(`\{[^{}]*\}`)

func parseRememberVerdictJSON(s string) (RememberVerdict, bool) {
	block := verdictJSONRe.FindString(s)
	if block == "" {
		return RememberVerdict{}, false
	}
	var raw struct {
		Remember   json.RawMessage `json:"remember"`
		ShouldRemember json.RawMessage `json:"should_remember"`
		Importance json.RawMessage `json:"importance"`
	}
	if err := json.Unmarshal([]byte(block), &raw); err != nil {
		return RememberVerdict{}, false
	}

	verdict := RememberVerdict{ShouldRemember: false, Importance: 3}

	// remember
	rememberRaw := raw.Remember
	if len(rememberRaw) == 0 {
		rememberRaw = raw.ShouldRemember
	}
	if len(rememberRaw) > 0 {
		b := false
		_ = json.Unmarshal(rememberRaw, &b)
		verdict.ShouldRemember = b
	}

	// importance：LLM 可能返回数字或带引号的字符串档位，都解析成 float64
	if len(raw.Importance) > 0 {
		if f, ok := parseJsonNumber(string(raw.Importance)); ok {
			verdict = applyImportanceClamp(verdict, f)
		}
	}

	return verdict, true
}

// parseJsonNumber 把 JSON 值解析成 float64，兼容数字（5）与带引号字符串（"5"）。
func parseJsonNumber(raw string) (float64, bool) {
	var f float64
	if err := json.Unmarshal([]byte(raw), &f); err == nil {
		return f, true
	}
	var s string
	if err := json.Unmarshal([]byte(raw), &s); err == nil {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// applyImportanceClamp 把重要度限制到 1-5 区间并取整到有效档位。
func applyImportanceClamp(v RememberVerdict, importance float64) RememberVerdict {
	if importance < 1 {
		importance = 1
	}
	if importance > 5 {
		importance = 5
	}
	v.Importance = float64(int(importance + 0.5))
	if v.Importance < 1 {
		v.Importance = 1
	}
	return v
}

// importance01 把 1-5 档位换算成 gracedb 需要的 [0,1] 值（importanceScore 会 clamp01）。
func importance01(importance float64) float64 {
	if importance > 5 {
		importance = 5
	}
	if importance < 1 {
		importance = 1
	}
	return importance / 5
}
