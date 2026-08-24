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

// rememberVerdictNegativeClause 判定提示中通用的"不值得记忆"负面清单，供分身/群共用，避免两处改得不一致。
const rememberVerdictNegativeClause = "不值得记忆：寒暄问候、确认/感谢短句、情绪化表达、日常流水（吃饭、天气、出行等琐碎日常）、针对某人的临时答复、闲聊式问答（未形成可复用知识）、一次性/过期即失效的信息（“今天”“这周”等相对时间）、即时操作指令（“把文件放到桌面”这类当下动作）、敏感凭据（密码、token、密钥、证件号、卡号）、吐槽与主观宣泄。"

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
- importance: 1-5 的整数档位，参考锚点：
  - 5: 影响长期目标/项目成败的关键决定
  - 4: 明确约定、承诺、会议安排
  - 3: 明确偏好/习惯/共识
  - 2: 偶发、短效信息
  - 1: 琐碎
- 联动规则：若判定 importance ≤ 2（偶发或琐碎、记了没长期价值），通常也应 remember=false，不要记。

示例：
- 内容："项目 Alpha 上线定在下周一" → {"remember": true, "importance": 4}
- 内容："和张三约好周五晚上一起吃饭" → {"remember": true, "importance": 4}
- 内容："嗯嗯知道了" → {"remember": false, "importance": 1}

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
		Remember       json.RawMessage `json:"remember"`
		ShouldRemember json.RawMessage `json:"should_remember"`
		Importance     json.RawMessage `json:"importance"`
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
