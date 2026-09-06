package ai

import (
	"strings"
	"unicode/utf8"
)

// IsComplexQuery 粗略判断问题是否需要深度推理：命中走 digest 路由（管理员可配思考/推理模型），
// 未命中走 chat 快模型。保守取向——误判为"简单"只是失去深度推理；误判为"复杂"多花 token，
// 且 digest 未配置思考模型时两者走同一模型、无实际差异。
//
// 关键词判定的门槛：单个关键词必须配问号才触发（"为什么今天这么累""有什么建议"这类
// 日常口语不误路由到思考模型）；多个关键词或长问题/多问句即使无问号也判复杂。
func IsComplexQuery(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	// 长问题倾向多步骤/需推理
	if utf8.RuneCountInString(q) > 50 {
		return true
	}
	// 多问句 = 复合问题
	if strings.Count(q, "？")+strings.Count(q, "?") >= 2 {
		return true
	}
	// 推理触发词
	keywords := []string{
		"为什么", "如何", "怎么解决", "怎样", "分析", "比较", "对比", "区别", "差异",
		"推导", "证明", "计算", "评估", "预测", "规划", "方案", "建议", "原理",
		"原因", "总结", "归纳", "代码", "bug", "调试", "算法", "优化", "设计",
	}
	hits := 0
	for _, k := range keywords {
		if strings.Contains(q, k) {
			hits++
		}
	}
	if hits == 0 {
		return false
	}
	// 单关键词命中：必须是疑问句才判复杂，排除"如何下载""有什么建议"等短口语
	if hits == 1 {
		return strings.ContainsAny(q, "？?")
	}
	// ≥2 个关键词命中：复合提问特征明显，无问号也判复杂
	return true
}
