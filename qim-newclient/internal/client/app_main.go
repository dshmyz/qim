// qim-newclient 是 qim 的纯原生 mygo 桌面客户端。
// 布局与样式对齐 qim-client 主界面（60px 导航图标栏 + 300px 会话列表 + 聊天区）。
// 登录、会话、消息、收发、实时推送全部由 Go 侧 internal/qapi 直连服务端，
// 界面不含 WebView，也不依赖 qim-client 的前端工程。
package client

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/capture"
)

var (
	// -shot 给出路径时，窗口画好后截一张 PNG 存下来然后退出。
	// 比手按 Cmd+Shift+4 稳定：不受屏幕缩放、窗口遮挡、多显示器影响，
	// 适合做视觉回归和给文档配图。
	shotPath = flag.String("shot", "", "截图输出路径（.png），截完即退出")
	shotWait = flag.Duration("shot-wait", 2*time.Second, "截图前等页面画完的时间")
	// -selftest-shot "100,100,400,300"：不走交互，直接用给定选区跑一遍
	// 截屏→裁剪→写剪贴板链路，PNG 同时落盘，方便自动化验证。
	selfTestShot = flag.String("selftest-shot", "", "自测选区 x,y,w,h，跑完退出")
	// -overlay-test：只验证覆盖层窗口本身画得出来（显示→截自己→退出）。
	overlayTest = flag.Bool("overlay-test", false, "显示截图覆盖层并截自己，落盘 /tmp/qim-overlay.png 后退出")
	// -overlay-cycle N：跑 N 轮「显示→抓真屏→隐藏」，用真实屏幕截图判断
	// 覆盖层到底有没有出现在屏幕上。CapturePage 只重渲染绘制树，
	// 窗口没显示也会出图，所以只能用真屏截图当判据。
	overlayCycle = flag.Int("overlay-cycle", 0, "跑 N 轮覆盖层显示/隐藏，每轮抓真屏存 /tmp/qim-cycle-N.png")
	// -debug-shot：finish/cancel 被调用时打印调用栈，排查"没按就自己完成"。
	debugShotFlag = flag.Bool("debug-shot", false, "截图完成/取消时打印调用栈")
	// -conv：登录后自动选中第一个会话，配合 -shot 做真实数据下的聊天区视觉验证。
	autoConv = flag.Bool("conv", false, "登录成功后自动打开第一个会话（截图自测用）")
	// -shot "out.png"：窗口画好后截一张 PNG 存下来然后退出（纯原生渲染，无网页层）。
)

// Run 是客户端入口：解析 flag、跑接口探针或启动 mygo 窗口。
func Run() {
	flag.Parse()

	// 接口探针不创建窗口：它验证的是 Go 侧网络能力，
	// 和渲染无关，所以要在 mygo 初始化之前就跑完退出。
	if *probeAPI {
		os.Exit(runAPIProbe())
	}
	if *probeActions {
		os.Exit(runActionProbe())
	}
	if *probeMedia {
		os.Exit(runMediaProbe())
	}
	if *probeGroups {
		os.Exit(runGroupsProbe())
	}
	if *probeAnnounce {
		os.Exit(runAnnouncementProbe())
	}
	if *probeReadUsers {
		os.Exit(runReadUsersProbe())
	}
	if *probeRecall {
		os.Exit(runRecallProbe())
	}
	if *probeChannels {
		os.Exit(runChannelsProbe())
	}
	if *probeNotifs {
		os.Exit(runNotifsProbe())
	}
	if *probeProfileF {
		os.Exit(runProfileProbe())
	}
	if *probeQuickF {
		os.Exit(runQuickProbe())
	}
	if *probeCF {
		os.Exit(runCProbe())
	}
	if *probeFilesF {
		os.Exit(runFilesProbe())
	}
	if *probeMsgF {
		os.Exit(runMsgProbe())
	}
	if *probeConvList {
		os.Exit(runConvListProbe())
	}

	debugShot = *debugShotFlag
	a := newApp()

	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "QIM",
			Width:     1366,
			Height:    850,
			MinWidth:  720,
			MinHeight: 520,
			StateKey:  "main",
			// 无边框窗口，对齐 qim-client 的 Electron frameless：
			// 内容顶到窗口顶，左栏标题区可拖动，右上角自绘控制按钮。
			Frameless: true,
			Content:   ui.View(a.view),
		})

		// 命令行给了账号密码就自动登录，方便对着假服务端直接验证原生链路；
		// 没给时先尝试用钥匙串里的凭据免登录，失败再停在原生登录页。
		if *flagUser != "" && *flagPass != "" {
			a.submitLogin()
		} else {
			a.restoreSession()
		}
		if *shotPath != "" {
			go saveShot(a, *shotPath, *shotWait)
		}

		// 微信式截图：Ctrl+Cmd+A（微信是 Alt+A / ⌥A，但 macOS 上 ⌥A 打字符，
		// 用 Ctrl+Cmd+A 不冲突）。覆盖层按显示器建，主屏先跑通。
		ov, err := newOverlay(mygo.Screen.PrimaryDisplay().Bounds, nil, nil)
		if err != nil {
			log.Printf("初始化截图: %v", err)
		} else {
			a.shot = ov
			ov.onDone = func(sel ui.Rect) {
				// 用按下快捷键瞬间的底图裁剪（不能重抓屏——覆盖层自己会入镜）
				d := ov.disp
				dip := image.Rect(0, 0, d.Width, d.Height)
				img := capture.Crop(ov.shot, ov.imageRect(sel), dip)
				if data, err := capture.PNG(img); err != nil {
					log.Printf("编码 PNG: %v", err)
				} else if err := mygo.Clipboard.WriteImage(data); err != nil {
					log.Printf("写剪贴板: %v", err)
				} else {
					log.Printf("截图已复制到剪贴板（%dx%d）", int(sel.W), int(sel.H))
				}
				ov.Close()
			}
			ov.onCancel = ov.Close

			// -overlay-test：把覆盖层显示出来，再截它自己一张，验证原生 ui
			// 在透明 Panel 窗口里到底有没有画出来；顺带跑两轮
			// 显示→隐藏→再显示，验证"框完一次还能不能再框"。
			if *overlayTest {
				go func() {
					for i := 1; i <= 2; i++ {
						if err := ov.Show(); err != nil {
							log.Printf("overlay-test 第%d轮 Show: %v", i, err)
							mygo.App.Exit(1)
							return
						}
						time.Sleep(800 * time.Millisecond)
						log.Printf("第%d轮 visible=%v", i, ov.win.IsVisible())
						data, err := ov.win.CapturePage()
						if err != nil {
							log.Printf("第%d轮 CapturePage: %v", i, err)
						} else {
							path := fmt.Sprintf("/tmp/qim-overlay-%d.png", i)
							if err := os.WriteFile(path, data, 0o644); err != nil {
								log.Printf("写 %s: %v", path, err)
							} else {
								log.Printf("已保存 %s（%d KB）", path, len(data)/1024)
							}
						}
						ov.Close()
						time.Sleep(400 * time.Millisecond)
					}
					mygo.App.Exit(0)
				}()
			}
			// -overlay-cycle：真机验证「框完一次能不能再框」。
			// 判据是真实屏幕截图（CG 抓的是合成后的屏幕，包含我们自己的窗口），
			// 而不是 CapturePage（它只重渲染绘制树，窗口没显示也能出图）。
			if *overlayCycle > 0 {
				n := *overlayCycle
				go func() {
					time.Sleep(3 * time.Second) // 等 App 激活/主窗口稳定，模拟真人点按钮的时机
					for i := 1; i <= n; i++ {
						if err := ov.Show(); err != nil {
							log.Printf("cycle 第%d轮 Show: %v", i, err)
							mygo.App.Exit(1)
							return
						}
						// 固定持有 2.5s，方便外部脚本在轮次中点截图判定
						deadline := time.Now().Add(2500 * time.Millisecond)
						for time.Now().Before(deadline) {
							ov.win.Invalidate()
							time.Sleep(100 * time.Millisecond)
						}
						px, py := ov.win.Position()
						log.Printf("cycle 第%d轮 visible=%v frames=%d disp=%+v pos=(%d,%d) bounds=%+v",
							i, ov.win.IsVisible(), ov.frames, ov.disp, px, py, ov.win.Bounds())
						d := ov.disp
						img, err := capture.Shot(image.Rect(d.X, d.Y, d.X+d.Width, d.Y+d.Height))
						if err == nil {
							// 底图平均亮度 vs 此刻真屏平均亮度：覆盖层在的话，
							// 真屏上是"底图 + 40% 蒙层"，比底图暗约 0.6 倍。
							base := meanLuma(ov.shot)
							now := meanLuma(img)
							log.Printf("cycle 第%d轮 亮度比=%.2f（<0.8 说明蒙层真的在屏幕上）", i, now/base)
							if data, e := capture.PNG(img); e == nil {
								_ = os.WriteFile(fmt.Sprintf("/tmp/qim-cycle-%d.png", i), data, 0o644)
								log.Printf("cycle 第%d轮真屏已存 %d KB", i, len(data)/1024)
							}
						} else {
							log.Printf("cycle 第%d轮抓真屏: %v", i, err)
						}
						ov.Close()
						time.Sleep(1500 * time.Millisecond)
					}
					mygo.App.Exit(0)
				}()
			}
			if *selfTestShot != "" {
				var x, y, w, h int
				fmt.Sscanf(*selfTestShot, "%d,%d,%d,%d", &x, &y, &w, &h)
				go func() {
					if err := ov.Show(); err != nil {
						log.Fatalf("selftest 抓屏: %v", err)
					}
					ov.onDone(ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: float32(h)})
					// 把刚才那张选区图落盘，方便肉眼确认裁得对不对
					d := ov.disp
					dip := image.Rect(0, 0, d.Width, d.Height)
					img := capture.Crop(ov.shot, ov.imageRect(ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: float32(h)}), dip)
					if data, err := capture.PNG(img); err == nil {
						_ = os.WriteFile("/tmp/qim-selftest-crop.png", data, 0o644)
					}
					mygo.App.Exit(0)
				}()
			}
			if err := mygo.GlobalShortcut.Register("Ctrl+Cmd+A", func() {
				a.startCapture()
			}); err != nil {
				log.Printf("注册全局快捷键: %v", err)
			}
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// meanLuma 抽样算图像的平均亮度（0~255），用于自动判断蒙层有没有上屏。
func meanLuma(img image.Image) float64 {
	b := img.Bounds()
	var s, n float64
	for y := b.Min.Y; y < b.Max.Y; y += 13 {
		for x := b.Min.X; x < b.Max.X; x += 13 {
			r, g, bb, _ := img.At(x, y).RGBA()
			s += float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(bb>>8)*0.114
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return s / n
}

// startCapture 开一次截图。延迟一拍再抓屏：让按钮的按下态退掉、
// 主窗口先重绘完，否则截到的图里按钮还是高亮的。
func (a *app) startCapture() {
	log.Printf("startCapture: shot=%v", a.shot != nil)
	if a.shot == nil {
		log.Printf("截图未初始化（screencapture 不可用或权限未给）")
		return
	}
	go func() {
		time.Sleep(150 * time.Millisecond)
		if err := a.shot.Show(); err != nil {
			log.Printf("抓屏失败: %v", err)
			return
		}
		// Show 是投递到主线程的异步调用，这里等一下再读才准
		time.Sleep(200 * time.Millisecond)
		log.Printf("覆盖层 visible=%v frames=%d", a.shot.win.IsVisible(), a.shot.frames)
	}()
}

// saveShot 是 -shot 的实现：等界面画完，把整个窗口截成 PNG 后退出。
// 现在界面是纯原生，Window.CapturePage() 一张图就是完整窗口，
// 不再需要像混合模式那样把网页层单独截出来再拼。
func saveShot(a *app, path string, wait time.Duration) {
	time.Sleep(wait)
	saveFrames(a, path)
	mygo.App.Exit(0)
}

func saveFrames(a *app, path string) {
	log.Printf("[shot] 截图时状态: loggedIn=%v convs=%d msgs=%d active=%d hasOlder=%v",
		a.loggedIn, len(a.convs), len(a.msgs), a.active, a.hasOlder)
	if data, err := a.win.CapturePage(); err != nil {
		log.Printf("Window.CapturePage: %v", err)
	} else if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Printf("写 %s: %v", path, err)
	} else {
		log.Printf("已保存 %s（%d KB）", path, len(data)/1024)
	}
}
