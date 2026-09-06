package ai

import "strings"

// IsComplexQuery 粗略判断问题是否需要深度推理：命中走 digest 路由（管理员可配思考/推理模型），
// 未命中走 chat 快模型。保守取向——误判为"简单"只是失去深度推理；误判为"复杂"多花 token，
// 且 digest 未配置思考模型时两者走同一模型、无实际差异。
func IsComplexQuery(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	// 长问题倾向多步骤/需推理
	if len([]rune(q)) > 50 {
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
	for _, k := range keywords {
		if strings.Contains(q, k) {
			return true
		}
	}
	return false
}
