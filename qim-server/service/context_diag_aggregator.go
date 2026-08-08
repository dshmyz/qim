package service

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// contextDiagAggLabel 聚合统计的维度：沿用 [ContextDiag] 打点的 label。
// 新增 label 时只需在 logHistoryDiagnostics / avatar 组装处调用对应聚合函数，无需改这里。
type contextDiagAggLabel string

const (
	aggLabelGroupBuildHistory contextDiagAggLabel = "群助手/buildHistory"
	aggLabelGroupGraph        contextDiagAggLabel = "群助手/graph"
	aggLabelAvatarHistory     contextDiagAggLabel = "分身/历史"
)

// contextDiagAgg 上下文污染聚合统计的窗口式计数器。
// 按 label 累加每一次历史注入的分布（近期/远期自身回复、token 等），
// 供定时任务把「一段时间内的占比趋势」打成一期快照，避免人肉翻 [ContextDiag] 单条日志。
// 设计：纯内存自包含，不依赖外部指标系统；不自行管理 goroutine，
// 由 main.go 挂 scheduler AddIntervalJob 周期性触发 reportAndReset。
type contextDiagAgg struct {
	mu     sync.Mutex
	window time.Time // 本期窗口起点，仅用于快照标注（秒级，Y-m-d H:i）
	cells  map[contextDiagAggLabel]*aggCell

	// avatarFilter 分身「过滤有效性」附加指标（保留/已滤），与 HistorySegment 正交，
	// 由 avatar_reply_graph.go:530 手动打点处单独聚合。
	avatarFilter *avatarFilterCell
}

type aggCell struct {
	samples   int   // 有注入的历史样本数（Total>0 才计入）
	totalMsgs int   // 累计注入消息总条数
	selfCount int   // 累计自身(AI)回复条数
	nearSelf  int   // 累计近期自身回复（窗口内，多轮锚点）
	farSelf   int   // 累计远期自身回复（窗口外，自我复制源）
	chars     int64 // 累计注入字符数
}

// avatarFilterCell 分身历史“过滤有效性”的窗口聚合（判断是否仍失忆）。
type avatarFilterCell struct {
	samples     int // 打点次数
	avatarTotal int // 累计分身回复总条数
	kept        int // 累计保留的近期回复
	filtered    int // 累计滤掉的远期回复
}

// contextDiagAggregator 全局聚合器实例。
var contextDiagAggregator = &contextDiagAgg{
	window: time.Now(),
	cells:  make(map[contextDiagAggLabel]*aggCell),
}

// isAggLabel 判断 name 是否为合法的聚合维度。
func isAggLabel(name contextDiagAggLabel) bool {
	switch name {
	case aggLabelGroupBuildHistory, aggLabelGroupGraph, aggLabelAvatarHistory:
		return true
	}
	return false
}

// aggregateHistorySegment 由 logHistoryDiagnostics 调用，把一次历史注入纳入窗口聚合。
// Total==0 的是「无历史注入」样本，不是污染信号，不计入（避免稀释占比）。
func aggregateHistorySegment(label string, seg HistorySegment) {
	aggLabel := contextDiagAggLabel(label)
	if _, ok := contextDiagAggregator.cells[aggLabel]; !ok && !isAggLabel(aggLabel) {
		return // 非聚合 label 的调用（防御：不应发生），静默忽略
	}
	contextDiagAggregator.mu.Lock()
	defer contextDiagAggregator.mu.Unlock()
	if seg.Total == 0 {
		return
	}
	cell := contextDiagAggregator.cells[aggLabel]
	if cell == nil {
		cell = &aggCell{}
		contextDiagAggregator.cells[aggLabel] = cell
	}
	cell.samples++
	cell.totalMsgs += seg.Total
	cell.selfCount += seg.SelfCount
	cell.nearSelf += seg.NearSelf
	cell.farSelf += seg.FarSelf
	cell.chars += int64(seg.chars)
}

// aggregateAvatarFilter 由分身历史打点处聚合「保留/已滤」，判断过滤是否有失忆残留。
func aggregateAvatarFilter(avatarTotal, kept, filtered int) {
	contextDiagAggregator.mu.Lock()
	defer contextDiagAggregator.mu.Unlock()
	c := contextDiagAggregator.avatarFilter
	if c == nil {
		c = &avatarFilterCell{}
		contextDiagAggregator.avatarFilter = c
	}
	c.samples++
	c.avatarTotal += avatarTotal
	c.kept += kept
	c.filtered += filtered
}

// percentages 计算本窗口的自身回复占比、远期自身回复占比、平均 token（每个样本）。
func (c *aggCell) percentages() (selfPct, nearPct, farPct int, avgTok int) {
	if c.totalMsgs > 0 {
		selfPct = c.selfCount * 100 / c.totalMsgs
	}
	if c.selfCount > 0 {
		nearPct = c.nearSelf * 100 / c.selfCount
		farPct = c.farSelf * 100 / c.selfCount
	}
	if c.samples > 0 {
		avgTok = int(c.chars/3) / c.samples // token 估算沿用 ~3 字符/token，按样本平均
	}
	return selfPct, nearPct, farPct, avgTok
}

// aggLabelOrder 返回固定顺序，便于快照输出稳定（不依赖 map 遍历顺序）。
func aggLabelOrder() []contextDiagAggLabel {
	return []contextDiagAggLabel{
		aggLabelGroupBuildHistory,
		aggLabelGroupGraph,
		aggLabelAvatarHistory,
	}
}

// contextDiagAggReportAndReset 生成一期聚合快照文本并清零窗口。
// 窗口式：看到的是「最近一段时间的占比」，而非历史总和，便于观察调参前后的变化。
// 由 main.go 的 scheduler AddIntervalJob 周期性调用。
func contextDiagAggReportAndReset() {
	contextDiagAggregator.mu.Lock()
	defer contextDiagAggregator.mu.Unlock()

	var b strings.Builder
	hasAny := false

	for _, label := range aggLabelOrder() {
		cell := contextDiagAggregator.cells[label]
		if cell == nil || cell.samples == 0 {
			continue
		}
		hasAny = true
		selfPct, nearPct, farPct, avgTok := cell.percentages()
		fmt.Fprintf(&b, "\n  · %s: 样本=%d 注入=%d 自身回复占比=%d%% (近期=%d%% 远期=%d%%) 平均token~%d",
			label, cell.samples, cell.totalMsgs, selfPct, nearPct, farPct, avgTok)
	}

	// 分身过滤有效性：保留率/已滤率都相对「分身回复总条数」。
	if c := contextDiagAggregator.avatarFilter; c != nil && c.samples > 0 {
		hasAny = true
		keptPct, filteredPct := 0, 0
		if c.avatarTotal > 0 {
			keptPct = c.kept * 100 / c.avatarTotal
			filteredPct = c.filtered * 100 / c.avatarTotal
		}
		fmt.Fprintf(&b, "\n  · 分身过滤有效性: 样本=%d 分身总=%d 保留=%d(%d%%) 已滤=%d(%d%%)",
			c.samples, c.avatarTotal, c.kept, keptPct, c.filtered, filteredPct)
	}

	if !hasAny {
		log.Printf("[ContextDiagAgg] 本期无任何历史注入样本（各 label 均无数据）")
	} else {
		log.Printf("[ContextDiagAgg] 上一窗口 %s 各 label 趋势：%s",
			time.Since(contextDiagAggregator.window).Round(time.Minute), b.String())
	}

	// 清零，进入下一窗口
	contextDiagAggregator.window = time.Now()
	contextDiagAggregator.cells = make(map[contextDiagAggLabel]*aggCell)
	contextDiagAggregator.avatarFilter = nil
}

// ContextDiagAggReport 导出一次聚合上报（取快照+清零当前窗口）。
// 供 main.go 的 scheduler 周期性触发；非调度器环境可用 StartContextDiagAggregator 自起 goroutine。
func ContextDiagAggReport() {
	contextDiagAggReportAndReset()
}

// StartContextDiagAggregator 在独立 goroutine 按时上报聚合快照。
// 主要用于非调度器环境（如本地调试/测试）；生产由 main.go 挂 scheduler 周期触发。
// 二者不冲突——都是「取快照+清零」，重复定时也只是多打几期日志。
func StartContextDiagAggregator(interval time.Duration, stop <-chan struct{}) {
	logger.WithModule("ContextDiagAgg").Info("启动上下文污染聚合定时上报", "interval", interval.String())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logger.WithModule("ContextDiagAgg").Info("定时上报上下文污染聚合快照")
				contextDiagAggReportAndReset()
			case <-stop:
				contextDiagAggReportAndReset() // 收尾上报一期
				return
			}
		}
	}()
}
