package ai

// reply_routing.go —— 对话类回复的统一任务路由判定内核。
// 视觉路由与复杂度分级的判定只在这里写一份，各回复入口（AI 工作台 /ai/completion、
// bot 1:1、群 @AI 提及与自动回复）都调本函数；以后调整路由规则只改这里。
// 不适用范围：分身回复（自选模型门控必须与生成共用同一模型，掺入分级会复现门控脱节，
// 见 service/avatar_reply_graph.go）；固定 taskType 的后台任务（intent 检测、总结、
// digest/analysis 作业等，各自路由语义独立）。

// HasImages 报告消息里是否携带图片（ImageURL 单图 / ImageURLs 多图）。
// 供调用方在无视觉路由时做诚实降级/提示（ResolveReplyTaskType 内部判定已收编同一逻辑）。
func HasImages(messages []Message) bool {
	for _, m := range messages {
		if m.ImageURL != "" || len(m.ImageURLs) > 0 {
			return true
		}
	}
	return false
}

// ResolveReplyTaskType 返回一条对话回复应使用的任务路由：
//  1. 消息含图：显式配置了视觉路由 -> TaskTypeVision（图片走专门视觉模型，纯文本模型
//     收到 base64 必然 400）；未配置 -> TaskTypeChat，由调用方决定降级方式（群 @AI 会把
//     被引用图片改写为"看不了"提示语，见 SmartReplyGraph.resolveQuotedImageTaskType）。
//  2. 纯文本：按最后一条用户消息的复杂度分级（ChatTaskType）：复杂 -> digest（管理员可配
//     思考/推理模型），简单 -> chat（快模型）。digest 未显式配置路由时由 router 回退
//     defaultTask，与 chat 同模型，行为无差异。
func (s *AIService) ResolveReplyTaskType(messages []Message) TaskType {
	if HasImages(messages) {
		if s.HasVisionRoute() {
			return TaskTypeVision
		}
		return TaskTypeChat
	}
	return s.ChatTaskType(messages)
}
