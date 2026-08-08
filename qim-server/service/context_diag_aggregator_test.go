package service

import (
	"testing"
)

// resetContextDiagAgg 清空全局聚合器，保证测试相互独立、不互相污染。
func resetContextDiagAgg() {
	contextDiagAggregator.mu.Lock()
	contextDiagAggregator.cells = make(map[contextDiagAggLabel]*aggCell)
	contextDiagAggregator.avatarFilter = nil
	contextDiagAggregator.mu.Unlock()
}

// TestAggregateHistorySegment 验证按 label 累加：注入条数/自身回复/近远期/样本数正确，
// 且「无注入」样本（Total==0）不计入聚合（避免稀释占比）。
func TestAggregateHistorySegment(t *testing.T) {
	resetContextDiagAgg()
	defer resetContextDiagAgg()

	// 两个有注入的样本
	aggregateHistorySegment(string(aggLabelGroupBuildHistory), HistorySegment{Total: 10, SelfCount: 5, NearSelf: 1, FarSelf: 4, chars: 300})
	aggregateHistorySegment(string(aggLabelGroupBuildHistory), HistorySegment{Total: 4, SelfCount: 1, NearSelf: 1, FarSelf: 0, chars: 0})
	// 无注入样本：不应计入 samples
	aggregateHistorySegment(string(aggLabelGroupBuildHistory), HistorySegment{Total: 0})

	contextDiagAggregator.mu.Lock()
	cell := contextDiagAggregator.cells[aggLabelGroupBuildHistory]
	contextDiagAggregator.mu.Unlock()

	if cell == nil {
		t.Fatal("cell 不应为 nil")
	}
	if cell.samples != 2 {
		t.Fatalf("samples = %d, want 2（无注入样本不应计入）", cell.samples)
	}
	if cell.totalMsgs != 14 {
		t.Fatalf("totalMsgs = %d, want 14", cell.totalMsgs)
	}
	if cell.selfCount != 6 {
		t.Fatalf("selfCount = %d, want 6", cell.selfCount)
	}
	if cell.nearSelf != 2 || cell.farSelf != 4 {
		t.Fatalf("near/far = %d/%d, want 2/4", cell.nearSelf, cell.farSelf)
	}
}

// TestAggregateAvatarFilter 验证分身「保留/已滤」聚合累加与占比。
func TestAggregateAvatarFilter(t *testing.T) {
	resetContextDiagAgg()
	defer resetContextDiagAgg()

	aggregateAvatarFilter(10, 2, 8) // 分身总10，保留2，已滤8
	aggregateAvatarFilter(5, 5, 0)  // 分身总5，保留5，已滤0

	contextDiagAggregator.mu.Lock()
	c := contextDiagAggregator.avatarFilter
	contextDiagAggregator.mu.Unlock()
	if c == nil {
		t.Fatal("avatarFilter 不应为 nil")
	}
	if c.samples != 2 || c.avatarTotal != 15 || c.kept != 7 || c.filtered != 8 {
		t.Fatalf("聚合值不对: samples=%d total=%d kept=%d filtered=%d", c.samples, c.avatarTotal, c.kept, c.filtered)
	}
}

// TestPercentages 验证占比计算口径：自身回复占比（相对注入总条数）、
// 近/远期占比（相对自身回复条数）、平均 token（相对样本数）。
func TestPercentages(t *testing.T) {
	cell := &aggCell{samples: 2, totalMsgs: 100, selfCount: 40, nearSelf: 10, farSelf: 30, chars: 900}
	selfPct, nearPct, farPct, avgTok := cell.percentages()
	if selfPct != 40 { // 40/100
		t.Fatalf("selfPct = %d, want 40", selfPct)
	}
	if nearPct != 25 { // 10/40
		t.Fatalf("nearPct = %d, want 25", nearPct)
	}
	if farPct != 75 { // 30/40
		t.Fatalf("farPct = %d, want 75", farPct)
	}
	if avgTok != 150 { // (900/3)/2
		t.Fatalf("avgTok = %d, want 150", avgTok)
	}

	// 除零保护：无自身回复 / 无注入时应为 0，不 panic
	empty := &aggCell{}
	if _, _, farPct, _ := empty.percentages(); farPct != 0 {
		t.Fatalf("空 cell 的 farPct 应为 0, got %d", farPct)
	}
}

// TestContextDiagAggReportAndReset 验证上报是「取快照+清零」语义：产生快照后窗口被清空，
// 下一期重新从 0 累计（窗口式趋势，而非历史总和）。
func TestContextDiagAggReportAndReset(t *testing.T) {
	resetContextDiagAgg()
	defer resetContextDiagAgg()

	aggregateHistorySegment(string(aggLabelGroupBuildHistory), HistorySegment{Total: 100, SelfCount: 40, NearSelf: 10, FarSelf: 30, chars: 900})
	aggregateAvatarFilter(10, 2, 8)

	// 上报不应 panic，且触发清零
	contextDiagAggReportAndReset()

	contextDiagAggregator.mu.Lock()
	cellsEmpty := len(contextDiagAggregator.cells) == 0
	avatarNil := contextDiagAggregator.avatarFilter == nil
	contextDiagAggregator.mu.Unlock()
	if !cellsEmpty {
		t.Fatalf("上报后 cells 应为空")
	}
	if !avatarNil {
		t.Fatal("上报后 avatarFilter 应为 nil")
	}

	// 上报后再聚合，应只反映新窗口（从 0 开始）。
	// 注意：读取 cell 前必须释放锁，aggregateHistorySegment 内部会再拿锁（非重入）。
	aggregateHistorySegment(string(aggLabelGroupBuildHistory), HistorySegment{Total: 10, SelfCount: 1, NearSelf: 1, FarSelf: 0})
	contextDiagAggregator.mu.Lock()
	cell := contextDiagAggregator.cells[aggLabelGroupBuildHistory]
	contextDiagAggregator.mu.Unlock()
	if cell.samples != 1 || cell.totalMsgs != 10 {
		t.Fatalf("上报后应重新累计，got samples=%d totalMsgs=%d", cell.samples, cell.totalMsgs)
	}
}

// TestAggregateHistorySegmentNonAggLabel 防御：非聚合 label 的调用应静默忽略。
func TestAggregateHistorySegmentNonAggLabel(t *testing.T) {
	resetContextDiagAgg()
	defer resetContextDiagAgg()

	aggregateHistorySegment("某未知label", HistorySegment{Total: 5, SelfCount: 1})

	contextDiagAggregator.mu.Lock()
	defer contextDiagAggregator.mu.Unlock()
	if len(contextDiagAggregator.cells) != 0 {
		t.Fatalf("非聚合 label 不应写进 cells，got %d", len(contextDiagAggregator.cells))
	}
}
