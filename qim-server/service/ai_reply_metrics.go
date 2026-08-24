package service

import "sync/atomic"

// AIReplyMetricsSnapshot 是 AI 回复质量指标的只读快照。
type AIReplyMetricsSnapshot struct {
	AutoAttempts       uint64 `json:"autoAttempts"`
	AutoReplies        uint64 `json:"autoReplies"`
	QualityRejected    uint64 `json:"qualityRejected"`
	QualityCheckFailed uint64 `json:"qualityCheckFailed"`
	ManualMentions     uint64 `json:"manualMentions"`
	UserIgnored        uint64 `json:"userIgnored"`
}

// AIReplyMetrics 使用进程内原子计数，避免在回复热路径引入数据库写入。
// 后续可将 Snapshot 接入现有监控出口或 Prometheus exporter。
type AIReplyMetrics struct {
	autoAttempts       atomic.Uint64
	autoReplies        atomic.Uint64
	qualityRejected    atomic.Uint64
	qualityCheckFailed atomic.Uint64
	manualMentions     atomic.Uint64
	userIgnored        atomic.Uint64
}

var GlobalAIReplyMetrics = &AIReplyMetrics{}

func (m *AIReplyMetrics) RecordAutoAttempt()        { m.autoAttempts.Add(1) }
func (m *AIReplyMetrics) RecordAutoReply()          { m.autoReplies.Add(1) }
func (m *AIReplyMetrics) RecordQualityRejected()    { m.qualityRejected.Add(1) }
func (m *AIReplyMetrics) RecordQualityCheckFailed() { m.qualityCheckFailed.Add(1) }
func (m *AIReplyMetrics) RecordManualMention()      { m.manualMentions.Add(1) }
func (m *AIReplyMetrics) RecordUserIgnored()        { m.userIgnored.Add(1) }

func (m *AIReplyMetrics) Snapshot() AIReplyMetricsSnapshot {
	return AIReplyMetricsSnapshot{
		AutoAttempts:       m.autoAttempts.Load(),
		AutoReplies:        m.autoReplies.Load(),
		QualityRejected:    m.qualityRejected.Load(),
		QualityCheckFailed: m.qualityCheckFailed.Load(),
		ManualMentions:     m.manualMentions.Load(),
		UserIgnored:        m.userIgnored.Load(),
	}
}
