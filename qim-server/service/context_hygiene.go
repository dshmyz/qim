package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
)

// HistorySegment 描述一份对话历史里「自身（assistant/avatar）回复」的近期/远期分布。
// 供诊断日志与后续「三层分治」的窗口化逻辑共用，避免各组装点重复统计。
type HistorySegment struct {
	Total       int // 注入的消息总条数
	SelfCount   int // 自身(AI)回复条数（assistant 或 avatar）
	NearSelf    int // 近期自身回复条数（<= window 内，多轮指代锚点）
	FarSelf     int // 远期自身回复条数（> window，自我复制污染源）
	filteredOut int // 已按“自身回复”被滤掉的历史条数（分身路径）
	chars       int // 注入历史总字符数
	tokenEst    int // 估算 token（约 3 字符/token）
}

const (
	// selfTurnWindow 近期自身回复的判定窗口：多轮追问“你刚说的/上一个”依赖窗口内的
	// 最近自回复作为指代锚点；窗口之外的自回复是自我复制污染源，应被折叠/丢弃。
	selfTurnWindow = 15 * time.Minute
)

// isNearSelf 判断一条消息是否为“近期自身（AI）回复”——在 selfTurnWindow 内。
// 近期自身回复保留为多轮指代锚点；远期自身回复是自我复制污染源，应折叠而非原样回灌。
func isNearSelf(m model.Message) bool {
	return (m.Origin == "assistant" || m.Origin == "avatar") &&
		time.Since(m.CreatedAt) <= selfTurnWindow
}

// isSelf 判断一条消息是否为自身（AI）回复（不分近远期）。
func isSelf(m model.Message) bool {
	return m.Origin == "assistant" || m.Origin == "avatar"
}

// foldFarSelf 把远期自身回复折叠成一行概要，供追加到 system prompt。
// 逐条截断并合计截断，避免折叠本身成为新的 token 噪音。全程按 rune 计数（中文安全）。
func foldFarSelf(items []string) string {
	const maxPer = 60
	const maxTotal = 160
	var out []string
	total := 0
	for _, it := range items {
		runes := []rune(strings.TrimSpace(it))
		if len(runes) == 0 {
			continue
		}
		if len(runes) > maxPer {
			runes = runes[:maxPer]
		}
		runes = append([]rune(nil), runes...) // 独立副本，避免后续共享
		if total+len(runes) > maxTotal {
			remain := maxTotal - total
			if remain < 1 {
				break
			}
			out = append(out, string(runes[:remain])+"…")
			break
		}
		out = append(out, string(runes))
		total += len(runes)
	}
	if len(out) == 0 {
		return "（无有效内容）"
	}
	return strings.Join(out, " ｜ ")
}

// segmentHistory 遍历（已按时间正序排好的）历史消息，统计自身回复的近期/远期分布。
// msgFiltered 为 nil 时不统计被滤条数（群助手路径不滤自身回复）。
func segmentHistory(messages []model.Message, msgFiltered func(model.Message) bool) HistorySegment {
	seg := HistorySegment{Total: len(messages)}
	for _, m := range messages {
		seg.chars += len(m.Content)
		if isSelf(m) {
			seg.SelfCount++
			if isNearSelf(m) {
				seg.NearSelf++
			} else {
				seg.FarSelf++
			}
		}
		if msgFiltered != nil && msgFiltered(m) {
			seg.filteredOut++
		}
	}
	seg.tokenEst = seg.chars / 3
	return seg
}

// logHistoryDiagnostics 打印一份对话历史的“自回复”注入诊断，供评估上下文污染。
// label 用于区分调用点（群助手 buildHistory 路径 / 群助手 graph 路径 / 分身历史）。
func logHistoryDiagnostics(label string, conversationID uint, messages []model.Message, msgFiltered func(model.Message) bool) {
	seg := segmentHistory(messages, msgFiltered)
	// 纳入窗口聚合（供定时报 [ContextDiagAgg] 快照看占比趋势）
	aggregateHistorySegment(label, seg)

	// 有历史就记；全空则记一行“无历史注入”便于区分“真的空”与“忘统计”。
	if seg.Total == 0 {
		log.Printf("[ContextDiag] %s conv=%d 无历史注入", label, conversationID)
		return
	}

	log.Printf("[ContextDiag] %s conv=%d 注入=%d 自身回复=%d (近期=%d 远期=%d)%s token~%d",
		label, conversationID, seg.Total, seg.SelfCount, seg.NearSelf, seg.FarSelf,
		filteredSuffix(seg.filteredOut), seg.tokenEst)
}

func filteredSuffix(filtered int) string {
	if filtered == 0 {
		return ""
	}
	return fmt.Sprintf(" 已滤自身=%d", filtered)
}
