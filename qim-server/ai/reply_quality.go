package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ReplyQualityVerdict 是发送前的轻量质量判定。
type ReplyQualityVerdict struct {
	Relevant    bool    `json:"relevant"`
	Supported   bool    `json:"supported"`
	Hallucinate bool    `json:"hallucinate"`
	Confidence  float32 `json:"confidence"`
	Reason      string  `json:"reason"`
}

func ParseReplyQualityVerdict(raw string) (ReplyQualityVerdict, error) {
	var verdict ReplyQualityVerdict
	s := strings.TrimSpace(raw)
	if i := strings.Index(s, "{"); i >= 0 {
		s = s[i:]
	}
	if j := strings.LastIndex(s, "}"); j >= 0 {
		s = s[:j+1]
	}
	if err := json.Unmarshal([]byte(s), &verdict); err != nil {
		return ReplyQualityVerdict{}, fmt.Errorf("解析回复质量判定失败: %w", err)
	}
	return verdict, nil
}

// ValidateReply 使用独立分析请求检查回复是否切题、是否有依据、是否产生幻觉。
func (s *AIService) ValidateReply(question, evidence, reply string) (ReplyQualityVerdict, error) {
	prompt := `你是回复质量审核器。只审核，不要改写回复。
判断 AI 回复是否切题、是否有依据、是否编造了资料中不存在的具体事实。
资料为空时，允许回答通用常识，但不能声称资料中明确写过不存在的事实。
只返回 JSON：{"relevant":true或false,"supported":true或false,"hallucinate":true或false,"confidence":0.0到1.0,"reason":"简短原因"}

用户问题：` + question + `

可用资料：
` + evidence + `

AI 回复：
` + reply
	raw, err := s.GetCompletion(TaskTypeAnalysis, []Message{{Role: "user", Content: prompt}})
	if err != nil {
		return ReplyQualityVerdict{}, err
	}
	return ParseReplyQualityVerdict(raw)
}

func (v ReplyQualityVerdict) ShouldSend() bool {
	return v.Relevant && v.Supported && !v.Hallucinate
}
