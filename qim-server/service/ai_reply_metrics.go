package service

import "sync/atomic"

// AIReplyMetricsSnapshot 是 AI 回复质量指标的只读快照。
type AIReplyMetricsSnapshot struct {
	AutoAttempts       uint64 `json:"autoAttempts"`
	AutoReplies        uint64 `json:"autoReplies"`
	QualityRejected    uint64 `json:"qualityRejected"`
	QualityAdvisory    uint64 `json:"qualityAdvisory"`
	QualitySkipped     uint64 `json:"qualitySkipped"`
	QualityCheckFailed uint64 `json:"qualityCheckFailed"`
	ManualMentions     uint64 `json:"manualMentions"`
	UserIgnored        uint64 `json:"userIgnored"`
}

// AIReplyMetrics 使用进程内原子计数，避免在回复热路径引入数据库写入。
// 后续可将 Snapshot 接入现有监控出口或 Prometheus exporter。
//
// 质量门三个口径（统筹设计，勿混用）：
//   - QualityRejected：核验不通过且阻断发送（auto 自动回复；@AI 仅在 mention_block 配置下）。
//   - QualityAdvisory：核验不通过但旁路放行（@AI 默认策略），正文照发仅记录。
//   - QualitySkipped：evidence 全空跳过核验（judge 无判据，verdict 是噪声）。
type AIReplyMetrics struct {
	autoAttempts       atomic.Uint64
	autoReplies        atomic.Uint64
	qualityRejected    atomic.Uint64
	qualityAdvisory    atomic.Uint64
	qualitySkipped     atomic.Uint64
	qualityCheckFailed atomic.Uint64
	manualMentions     atomic.Uint64
	userIgnored        atomic.Uint64
}

var GlobalAIReplyMetrics = &AIReplyMetrics{}

func (m *AIReplyMetrics) RecordAutoAttempt()        { m.autoAttempts.Add(1) }
func (m *AIReplyMetrics) RecordAutoReply()          { m.autoReplies.Add(1) }
func (m *AIReplyMetrics) RecordQualityRejected()    { m.qualityRejected.Add(1) }
func (m *AIReplyMetrics) RecordQualityAdvisory()    { m.qualityAdvisory.Add(1) }
func (m *AIReplyMetrics) RecordQualitySkipped()     { m.qualitySkipped.Add(1) }
func (m *AIReplyMetrics) RecordQualityCheckFailed() { m.qualityCheckFailed.Add(1) }
func (m *AIReplyMetrics) RecordManualMention()      { m.manualMentions.Add(1) }
func (m *AIReplyMetrics) RecordUserIgnored()        { m.userIgnored.Add(1) }

func (m *AIReplyMetrics) Snapshot() AIReplyMetricsSnapshot {
	return AIReplyMetricsSnapshot{
		AutoAttempts:       m.autoAttempts.Load(),
		AutoReplies:        m.autoReplies.Load(),
		QualityRejected:    m.qualityRejected.Load(),
		QualityAdvisory:    m.qualityAdvisory.Load(),
		QualitySkipped:     m.qualitySkipped.Load(),
		QualityCheckFailed: m.qualityCheckFailed.Load(),
		ManualMentions:     m.manualMentions.Load(),
		UserIgnored:        m.userIgnored.Load(),
	}
}
