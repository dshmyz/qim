package client

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// dragSpacer 回归：Spacer 在 AlignItems(Center) 的 Row 里会被压成 0 高度，
// Rect.Contains 用 y < Y+H，零高度元素永远命中不到指针，DragWindow 失效，
// 表现为顶栏/聊天头部拖不动。FillHeight 后必须有实际可点的宽高。
func TestDragSpacerHasHitArea(t *testing.T) {
	build := func(withFill bool) func(*ui.Context) {
		return func(c *ui.Context) {
			ui.Row(c).FillWidth().Height(40).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Width(60).Height(40).Background(ui.Hex("#eee"))
				sp := ui.Spacer(c).DragWindow().Label("dragzone")
				if withFill {
					sp.FillHeight()
				}
			})
		}
	}

	r0, ok := findZone(build(false))
	if !ok {
		t.Fatal("探针没找到 dragzone")
	}
	if r0.W > 0 && r0.H > 0 {
		t.Logf("注意：裸 Spacer 命中区为 %+v（框架行为可能已变）", r0)
	}

	r1, ok := findZone(build(true))
	if !ok {
		t.Fatal("FillHeight 后没找到 dragzone")
	}
	if r1.W <= 0 || r1.H <= 0 {
		t.Fatalf("拖拽区仍无命中面积: %+v", r1)
	}
	if r1.H != 40 {
		t.Errorf("拖拽区应吃满 Row 的 40px 高度，实际 H=%v: %+v", r1.H, r1)
	}
}

// findZone 渲染一帧并返回 dragzone 的矩形，避免测试体重复样板。
func findZone(build func(*ui.Context)) (ui.Rect, bool) {
	tt := ui.NewTester(build, 1050, 703)
	tt.Frame()
	tt.Frame()
	return tt.Find("dragzone")
}
