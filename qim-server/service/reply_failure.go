package service

import (
	"fmt"
	"strings"
)

// reply_failure.go —— 回复失败的统一话术。
//
// 失败分三类，用户能据此知道下一步该做什么：
//   - 看不了（MediaUnreadable）：图片/文件读不出来 → 换格式、压缩、用文字描述；
//   - 查不到（NoResult）：模型没给出结果 / 工具没成功 → 换问法、补充线索；
//   - 用不了（ModelUnavailable）：模型或服务不可用 → 稍后再试、联系管理员。
//
// 以前三种原因散落在 prepareInput、BuildEmptyReplyFallback、handler 兜底里各写一套，
// 用户看到的措辞与建议不一致。这里收敛成唯一来源：改文案只改这个文件。

type FailureKind string

const (
	FailureMediaUnreadable  FailureKind = "media_unreadable"
	FailureNoResult         FailureKind = "no_result"
	FailureModelUnavailable FailureKind = "model_unavailable"
)

// FailureDetail 失败的具体信息，用于把话术说得更具体（哪张图、哪个工具）。
type FailureDetail struct {
	// MediaType 媒体类型名，如"图片""文件"。
	MediaType string
	// Name 媒体文件名（可为空）。
	Name string
	// TooLarge 是否因体积超限失败（与格式/读取失败区分）。
	TooLarge bool
	// FailedTools 失败的工具标签（NoResult 时用于点名）。
	FailedTools []string
}

// BuildFailureReply 生成用户可见的失败话术（单一来源）。
func BuildFailureReply(kind FailureKind, d FailureDetail) string {
	switch kind {
	case FailureMediaUnreadable:
		media := d.MediaType
		if media == "" {
			media = "内容"
		}
		subject := media
		if d.Name != "" {
			subject = fmt.Sprintf("%s「%s」", media, d.Name)
		}
		if d.TooLarge {
			return fmt.Sprintf("这条%s太大了，超出了我能读取的上限。压缩一下重新发，或者把关键内容用文字发我。", subject)
		}
		return fmt.Sprintf("这条%s我读不出来（格式不支持或文件已失效）。换一种格式重发，或者把关键内容用文字发我。", subject)
	case FailureNoResult:
		if len(d.FailedTools) > 0 {
			return fmt.Sprintf("我调用了「%s」但没有成功，暂时给不出结果。可以稍后再试，或者告诉我更具体的线索（人名、群名、时间）。",
				strings.Join(d.FailedTools, "」「"))
		}
		return "这个问题我没找到可靠的答案。换个更具体的问法，或者告诉我相关的人、群和时间，我再查一次。"
	case FailureModelUnavailable:
		return "我这边暂时响应不了（服务异常），这次没能生成回复。稍后再试一次；如果一直这样，请联系管理员。"
	}
	return BuildFailureReply(FailureNoResult, d)
}

// BuildMediaUnreadablePrompt 把「看不了」的结论转成给模型看的提示：
// 模型据此如实说明，而不是编造图片/文件内容。用户可见措辞取自同一份话术。
func BuildMediaUnreadablePrompt(refPrefix, mediaType, name string, tooLarge bool, extraClause string) string {
	subject := fmt.Sprintf("一条%s消息", mediaType)
	if name != "" {
		subject = fmt.Sprintf("一条%s消息「%s」", mediaType, name)
	}
	stmt := BuildFailureReply(FailureMediaUnreadable, FailureDetail{
		MediaType: mediaType, Name: name, TooLarge: tooLarge,
	})
	prompt := fmt.Sprintf("%s%s：%s 请如实说明你看不到，不要猜测或编造其中的内容。", refPrefix, subject, stmt)
	if extraClause != "" {
		prompt += extraClause
	}
	return prompt
}
