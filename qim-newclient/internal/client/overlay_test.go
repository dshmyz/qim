package client

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestOverlayToolbarClickable 守住一个回归：工具栏必须叠在交互层之上。
// 之前交互层建在工具栏后面，盖住了按钮，框完一次就"卡住"点不动。
func TestOverlayToolbarClickable(t *testing.T) {
	o := &overlay{w: 800, h: 600}
	o.shot = image.NewRGBA(image.Rect(0, 0, 800, 600))
	o.bmp = ui.NewBitmap(o.shot)

	var got ui.Rect
	done := false
	o.onDone = func(sel ui.Rect) { done = true; got = sel }
	o.onCancel = func() { t.Error("不该走取消") }

	o.start = ui.Point{X: 100, Y: 100}
	o.cur = ui.Point{X: 400, Y: 400}
	o.picked = true

	tt := ui.NewTester(o.view, 800, 600)
	tt.Frame()

	if err := tt.Click("完成"); err != nil {
		t.Fatalf("点不到「完成」按钮: %v", err)
	}
	if !done {
		t.Fatal("点了「完成」但 onDone 没被调用")
	}
	if got.W != 300 || got.H != 300 {
		t.Errorf("选区尺寸不对: %+v", got)
	}
}

// TestOverlayReselect 框完一次后，再按下应当能重新框选（清掉已有选区）。
func TestOverlayReselect(t *testing.T) {
	o := &overlay{w: 800, h: 600}
	o.shot = image.NewRGBA(image.Rect(0, 0, 800, 600))
	o.bmp = ui.NewBitmap(o.shot)
	o.start = ui.Point{X: 100, Y: 100}
	o.cur = ui.Point{X: 400, Y: 400}
	o.picked = true

	// 模拟"再次按下开始新选区"的状态切换（真实拖拽由 Dragged() 驱动）
	o.picked = false
	o.dragging = true
	o.start = ui.Point{X: 50, Y: 50}
	o.cur = ui.Point{X: 60, Y: 60}

	tt := ui.NewTester(o.view, 800, 600)
	tt.Frame()
	if !tt.HasText("10 × 10") {
		t.Errorf("新选区尺寸标签没更新，texts=%q", tt.Texts())
	}
}
