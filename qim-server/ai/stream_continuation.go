package ai

import (
	"context"
	"errors"
	"strings"
)

var (
	// ErrStreamIncomplete 上游未发 [DONE] 即结束连接（代理回收连接的典型形态）。
	ErrStreamIncomplete = errors.New("stream ended without [DONE] marker")
	// ErrReplyTruncated 回复因长度上限/中断未能完整结束（截断标记已追加到内容末尾）。
	ErrReplyTruncated = errors.New("reply truncated")
)

// truncatedMarker 回复未完整结束时追加到正文末尾的可见标记，随正文持久化，
// 刷新/回放后仍可见——用户不会再把半截回复误当成完整回复。
const truncatedMarker = "（回复被截断）"

// errFinishLength 内部哨兵：finish_reason=length，随 ErrReplyTruncated Join 保留，
// 供 InterruptReasonFromError 区分「长度触顶」与一般错误。
var errFinishLength = errors.New("finish_reason=length")

// streamPrimaryFunc 执行一次主流式调用。
type streamPrimaryFunc func(ctx context.Context, onChunk func(StreamChunk) error) error

// streamContinueFunc 携带已生成的 partial 执行一次续写调用。
type streamContinueFunc func(ctx context.Context, partial string, onChunk func(StreamChunk) error) error

// runStreamWithContinuation 执行流式调用并处理中断：
//   - finish_reason=length 或无 [DONE] 断连（ErrStreamIncomplete）且已有正文 →
//     携带 partial 自动续写一次；
//   - 续写后仍触顶/失败，或主流在已有正文后发生其他错误 → 追加可见截断标记，
//     返回 ErrReplyTruncated（可 errors.Is 检测，底层错误被 Join 保留）；
//   - 无正文时错误原样返回，交调用方走兜底文案路径。
func runStreamWithContinuation(
	ctx context.Context,
	primary streamPrimaryFunc,
	continueCall streamContinueFunc,
	onChunk func(StreamChunk) error,
) error {
	var partial strings.Builder
	var finalFinish string
	wrapped := func(chunk StreamChunk) error {
		if chunk.Finish != nil {
			finalFinish = *chunk.Finish
		}
		if chunk.Content != "" {
			partial.WriteString(chunk.Content)
		}
		return onChunk(chunk)
	}

	primErr := primary(ctx, wrapped)

	needsContinue := false
	if partial.Len() > 0 {
		if primErr == nil && finalFinish == "length" {
			needsContinue = true
		} else if errors.Is(primErr, ErrStreamIncomplete) {
			needsContinue = true
		}
	}

	if !needsContinue {
		if primErr != nil && partial.Len() > 0 {
			appendTruncatedMarker(onChunk)
			return errors.Join(ErrReplyTruncated, primErr)
		}
		return primErr
	}

	var contFinish string
	contWrapped := func(chunk StreamChunk) error {
		if chunk.Finish != nil {
			contFinish = *chunk.Finish
		}
		return onChunk(chunk)
	}
	contErr := continueCall(ctx, partial.String(), contWrapped)
	if contErr == nil && contFinish != "length" {
		return nil
	}

	appendTruncatedMarker(onChunk)
	if contErr != nil {
		return errors.Join(ErrReplyTruncated, contErr)
	}
	return errors.Join(ErrReplyTruncated, errFinishLength)
}

func appendTruncatedMarker(onChunk func(StreamChunk) error) {
	_ = onChunk(StreamChunk{Content: truncatedMarker})
}

// InterruptReasonFromError 从流式错误推断中断原因标识，供终态事件落库。
func InterruptReasonFromError(err error) string {
	switch {
	case errors.Is(err, errFinishLength):
		return "length"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, ErrStreamIncomplete):
		return "no_done"
	default:
		return "error"
	}
}
