// 微信式截图的覆盖层：一个透明的全屏 Panel 窗口，
// 底图是按下快捷键瞬间抓到的屏幕原图，用户在上面拖一个选区。
//
// 交互对齐微信截图：
//   - 按下拖动画选区，选区外蒙 40% 黑，选区内是清晰的原图
//   - 拖动时选区上方显示实时尺寸
//   - 抬手后选区右下出现「完成 / 取消」工具栏
//   - 已选区状态下再按下 = 重新框选
//   - Enter=完成（图片进剪贴板），Esc=取消
package client

import (
	"image"
	"log"
	"qim-newclient/internal/theme"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/capture"
)

// debugShot 打开后 finish/cancel 会打印调用栈，用于排查"没人按就自己完成"。
var debugShot = false

type overlay struct {
	win  *mygo.Window
	disp mygo.Rectangle // 该显示器的 DIP bounds
	w, h float32

	shot image.Image // 快捷键瞬间的全屏原图（裁剪用）
	bmp  *ui.Bitmap  // shot 的 GPU 位图，只建一次；每帧重建会把拖拽帧率拖死

	// macOS 会把无边框窗口往下压，避开菜单栏，所以窗口实际位置可能是
	// y=30 而不是 0。底图是全屏截图（含菜单栏），必须往上挪同样的量
	// 才能和真实屏幕对齐；裁剪时再用同样的偏移取原图。
	offX, offY float32
	aligned    bool

	start, cur ui.Point // 拖拽起点 / 当前点（窗口局部 DIP）
	dragging   bool
	picked     bool // 抬手后选区有效
	frames     int  // 诊断：view 被调用的次数（>0 说明真的在出帧）
	onDone     func(sel ui.Rect)
	onCancel   func()
}

// newOverlay 截一张该显示器的屏，建覆盖层窗口。
// 必须 Hidden:true —— NewWindow 默认是可见的，而这个窗口全屏、不透明、
// AlwaysOnTop，创建即显示会把主窗口整个盖住，用户看到的是启动瞬间抓的
// 整屏底图，表现就是"应用打开了却什么界面都没有"。只有 Show() 才亮出来。
func newOverlay(disp mygo.Rectangle, onDone func(ui.Rect), onCancel func()) (*overlay, error) {
	o := &overlay{
		disp:     disp,
		w:        float32(disp.Width),
		h:        float32(disp.Height),
		onDone:   onDone,
		onCancel: onCancel,
	}
	if err := o.grab(); err != nil {
		return nil, err
	}

	o.win = mygo.NewWindow(mygo.WindowOptions{
		Title:     "截图",
		X:         disp.X,
		Y:         disp.Y,
		Width:     disp.Width,
		Height:    disp.Height,
		Frameless: true,
		Hidden:    true, // 等 Show()（快捷键/按钮）才显示，别挡住主窗口
		// 关键：**不要**用 Transparent。实测 mygo 0.7.4 上透明窗口
		// 的 ui.Content 不会被合成上屏（IsVisible=true、每帧都在出，
		// 屏幕上却什么都看不到），表现就是"第一次按有反应、再按没反应"。
		// 我们本来就铺满全屏底图，不需要窗口透明。
		Transparent:     false,
		Panel:           true, // 浮在所有应用上、不把用户从当前 App 切走
		AlwaysOnTop:     true,
		DisableResize:   true,
		DisableMove:     true,
		DisableMinimize: true,
		DisableMaximize: true,
		Content:         ui.View(o.view),
	})
	return o, nil
}

// grab 抓一张该显示器的全屏图，转成位图缓存。
func (o *overlay) grab() error {
	img, err := capture.Shot(image.Rect(o.disp.X, o.disp.Y, o.disp.X+o.disp.Width, o.disp.Y+o.disp.Height))
	if err != nil {
		return err
	}
	o.shot = img
	o.bmp = ui.NewBitmap(img)
	return nil
}

// Show 每次按快捷键都重新抓屏（保证底图是"此刻"的屏幕），再亮出覆盖层。
func (o *overlay) Show() error {
	if err := o.grab(); err != nil {
		return err
	}
	o.start, o.cur = ui.Point{}, ui.Point{}
	o.dragging, o.picked = false, false
	o.aligned = false // 每次显示都重新对一次位置（多显示器/切换用户后坐标会变）
	o.win.Show()
	// Hide 过的窗口重新 Show 必须主动刷一帧，否则还停在上次的画面。
	o.win.Invalidate()
	return nil
}

// Close 隐藏覆盖层。窗口保留，下次快捷键直接复用；
// 原图留在内存里，确认裁剪时用的就是按下快捷键瞬间的画面。
func (o *overlay) Close() {
	o.dragging, o.picked = false, false
	o.start, o.cur = ui.Point{}, ui.Point{}
	o.win.Hide()
}

// selRect 把起点/当前点归一化成选区（拖拽方向任意）。
func (o *overlay) selRect() ui.Rect {
	x1, x2 := o.start.X, o.cur.X
	y1, y2 := o.start.Y, o.cur.Y
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	return ui.Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

func (o *overlay) view(c *ui.Context) {
	o.frames++
	// 第一帧读一次窗口的真实位置，算出底图要往上挪多少。
	// （只读一次：native 调用不该每帧都做。）
	if !o.aligned {
		// headless 测试里没有真窗口（win==nil），直接按无偏移处理。
		if o.win != nil {
			b := o.win.Bounds()
			o.offX = float32(b.X - o.disp.X)
			o.offY = float32(b.Y - o.disp.Y)
		}
		o.aligned = true
	}
	ui.Box(c).FillWidth().FillHeight().Children(func() {
		// 层序很关键：底图 → 蒙层 → 边框/标签 → 交互层 → 工具栏。
		// 工具栏最后建才叠在交互层之上、点得到；交互层要盖住整屏
		// 接拖拽，但不能盖住工具栏（上一版正好反了，「完成」点不动）。
		ui.Image(c, o.bmp).Width(o.w).Height(o.h).
			Absolute().Left(-o.offX).Top(-o.offY)

		sel := o.selRect()
		hasSel := (o.dragging && sel.W > 3 && sel.H > 3) || o.picked

		// 选区外蒙层：上下左右四块
		if hasSel {
			mask(c, 0, 0, o.w, sel.Y)
			mask(c, 0, sel.Y+sel.H, o.w, o.h-sel.Y-sel.H)
			mask(c, 0, sel.Y, sel.X, sel.H)
			mask(c, sel.X+sel.W, sel.Y, o.w-sel.X-sel.W, sel.H)
		} else {
			mask(c, 0, 0, o.w, o.h)
		}

		// 选区边框 + 实时尺寸标签
		if hasSel {
			ui.Box(c).Width(sel.W).Height(sel.H).Absolute().Left(sel.X).Top(sel.Y).
				Border(1.5, theme.SelBd)
			label := itoa(int(sel.W)) + " × " + itoa(int(sel.H))
			tagY := sel.Y - 30
			if tagY < 4 {
				tagY = sel.Y + 6 // 贴到屏幕上沿时挪到选区内侧
			}
			ui.Box(c).Absolute().Left(sel.X + sel.W/2 - 45).Top(tagY).
				Width(90).Height(24).Radius(4).Background(theme.TipBg).Center().Children(func() {
				ui.Text(c, label).FontSize(12).TextColor(theme.White)
			})
		}

		// 交互层：全屏，接指针和键盘。用 PointerPosition 的绝对坐标，
		// 不再累加 Dragged 的每帧位移——那样会漂、会跳帧，跟手差。
		dragLayer := ui.Box(c).FillWidth().FillHeight().Absolute().Left(0).Top(0).
			Cursor(ui.CursorCrosshair).AutoFocus().
			OnShortcut(ui.Modifiers(0), ui.KeyEscape, o.cancel).
			OnShortcut(ui.Modifiers(0), ui.KeyEnter, o.finish)
		dragLayer.Children(func() {
			_, _, pressed := dragLayer.Dragged()
			px, py, over := dragLayer.PointerPosition()
			if pressed {
				if !o.dragging {
					// 已选区状态下再按下 = 重新框选（微信也是这样）
					o.dragging = true
					o.picked = false
					o.start = ui.Point{X: px, Y: py}
					o.cur = o.start
				} else if over {
					o.cur = ui.Point{X: px, Y: py}
				}
			} else if o.dragging {
				o.dragging = false
				s := o.selRect()
				if s.W > 3 && s.H > 3 {
					o.picked = true
				} else {
					o.start, o.cur = ui.Point{}, ui.Point{}
				}
			}
		})

		// 工具栏（抬手后出现，贴选区右下；空间不够依次翻到上方、内侧）
		if o.picked && hasSel {
			by := sel.Y + sel.H + 10
			if by+36 > o.h {
				by = sel.Y - 46
			}
			if by < 4 {
				by = sel.Y + sel.H - 46
			}
			bx := sel.X + sel.W - 168
			if bx < 4 {
				bx = 4
			}
			ui.Row(c).Gap(8).Absolute().Left(bx).Top(by).
				Width(168).Height(36).Children(func() {
				ui.Box(c).Grow(1).FillHeight().Radius(6).Background(theme.BtnBg2).Center().
					Cursor(ui.CursorPointer).OnClick(o.cancel).Children(func() {
					ui.Text(c, "取消").FontSize(13).TextColor(theme.White)
				})
				ui.Box(c).Grow(1).FillHeight().Radius(6).Background(theme.BtnBg).Center().
					Cursor(ui.CursorPointer).OnClick(o.finish).Children(func() {
					ui.Text(c, "完成").FontSize(13).TextColor(theme.White)
				})
			})
		}
	})
}

// imageRect 把窗口内坐标的选区换算成底图（真实屏幕）坐标：
// 加上窗口被系统下移的那段偏移，保证抠出来的就是用户在屏幕上框到的东西。
func (o *overlay) imageRect(s ui.Rect) image.Rectangle {
	return image.Rect(
		int(s.X+o.offX), int(s.Y+o.offY),
		int(s.X+s.W+o.offX), int(s.Y+s.H+o.offY),
	)
}

func (o *overlay) finish() {
	if debugShot {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		log.Printf("finish 被调用:\n%s", buf[:n])
	}
	s := o.selRect()
	if s.W < 3 || s.H < 3 {
		o.cancel()
		return
	}
	if o.onDone != nil {
		o.onDone(s)
	}
}

func (o *overlay) cancel() {
	if debugShot {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		log.Printf("cancel 被调用:\n%s", buf[:n])
	}
	if o.onCancel != nil {
		o.onCancel()
	}
}

func mask(c *ui.Context, x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	ui.Box(c).Width(w).Height(h).Absolute().Left(x).Top(y).Background(theme.Mask)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
