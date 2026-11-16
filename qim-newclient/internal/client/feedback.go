package client

// 问题反馈（对齐 qim-client FeedbackModal.vue + 后端 FeedbackHandler）。
//
// 两个视图：提交（类型三选 + 内容 ≥10 字 + 可选截图）与「我的反馈」列表
// （时间/类型/状态标签，展开看管理员回复）。截图走原生文件对话框，
// multipart 直传 POST /feedbacks；列表 GET /my-feedbacks 分页。

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// feedbackMinContent 与 qim-client 的「至少 10 个字」提示一致。
const feedbackMinContent = 10

// feedbackTypes 反馈类型（照搬 FeedbackModal.vue）。
var feedbackTypes = []struct {
	value string
	label string
}{
	{"bug", "Bug反馈"},
	{"feature", "功能建议"},
	{"other", "其他"},
}

func fbTypeLabel(t string) string {
	for _, ft := range feedbackTypes {
		if ft.value == t {
			return ft.label
		}
	}
	return t
}

// fbStatusLabel 状态标签文案（对齐 FeedbackModal 的 statusLabel）。
func fbStatusLabel(s string) string {
	switch s {
	case "pending":
		return "待处理"
	case "processing":
		return "处理中"
	case "resolved":
		return "已解决"
	case "closed":
		return "已关闭"
	}
	return s
}

// fbTimeLabel RFC3339 → 「MM-DD HH:MM」，解析失败截前 16 字符兜底。
func fbTimeLabel(s string) string {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local().Format("01-02 15:04")
	}
	if len(s) > 16 {
		return s[:16]
	}
	return s
}

// canSubmitFeedback 内容（去空白）不少于 10 字才可提交。
func (a *app) canSubmitFeedback() bool {
	return utf8.RuneCountInString(strings.TrimSpace(a.fbContent)) >= feedbackMinContent
}

func (a *app) openFeedback() {
	a.feedbackOpen = true
	a.fbView = "submit"
	a.fbErr = ""
}

func (a *app) closeFeedback() {
	a.feedbackOpen = false
	a.fbErr = ""
}

// switchFbList 切到「我的反馈」列表视图并拉第一页。
func (a *app) switchFbList() {
	a.fbView = "list"
	a.fbListPage = 0
	a.fbHasMore = false
	a.loadMyFeedbacks()
}

// loadMyFeedbacks 分页拉我的反馈（page 从 1 计；首页整表替换，翻页追加）。
func (a *app) loadMyFeedbacks() {
	if a.client == nil || !a.loggedIn || a.fbLoading {
		return
	}
	page := a.fbListPage + 1
	a.fbLoading = true
	ctx := a.ctx
	go func() {
		list, err := a.client.MyFeedbacks(ctx, page, 20)
		a.mutate(func() {
			a.fbLoading = false
			if err != nil {
				log.Printf("[feedback] 拉反馈列表失败: %v", err)
				a.fbErr = friendlyError(err)
				return
			}
			if page == 1 {
				a.fbList = list
			} else {
				a.fbList = append(a.fbList, list...)
			}
			a.fbListPage = page
			a.fbHasMore = len(list) >= 20
		})
	}()
}

// pickFbScreenshot 原生文件框选截图（必须 goroutine 里调 Dialog）。
func (a *app) pickFbScreenshot() {
	if a.win == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "选择反馈截图",
			ButtonLabel: "确定",
			Parent:      a.win,
			Filters: []mygo.FileFilter{{
				Name:       "图片",
				Extensions: []string{"png", "jpg", "jpeg", "gif", "webp"},
			}},
		})
		if err != nil || len(paths) == 0 || paths[0] == "" {
			return
		}
		a.mutate(func() { a.fbShotPath = paths[0] })
	}()
}

func (a *app) removeFbScreenshot() {
	a.fbShotPath = ""
}

// submitFeedback 提交反馈：成功后切到列表视图让用户看到刚提交的记录
// （对齐 FeedbackModal 的提交流转）。
func (a *app) submitFeedback() {
	if a.fbBusy || !a.canSubmitFeedback() {
		return
	}
	if a.client == nil || !a.loggedIn {
		a.fbErr = "未登录，无法提交反馈"
		return
	}
	typ, content, shot := a.fbType, strings.TrimSpace(a.fbContent), a.fbShotPath
	a.fbBusy = true
	a.fbErr = ""
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	go func() {
		defer cancel()
		err := a.client.SubmitFeedback(ctx, typ, content, shot)
		a.mutate(func() {
			a.fbBusy = false
			if err != nil {
				log.Printf("[feedback] 提交失败: %v", err)
				a.fbErr = "提交失败：" + friendlyError(err)
				return
			}
			a.fbContent = ""
			a.fbShotPath = ""
			a.fbType = "feature"
			a.switchFbList()
		})
	}()
}

// feedbackModal 反馈弹层（500px，对齐 FeedbackModal 的两视图结构）。
func (a *app) feedbackModal(c *ui.Context) {
	if !a.feedbackOpen {
		return
	}
	title := "问题反馈"
	if a.fbView == "list" {
		title = "我的反馈"
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(500).MaxHeight(640).Radius(12).Background(theme.White).
				Padding(20, 20).Gap(10).Children(func() {
				// 头部：标题 + 关闭
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(theme.Gray800)
					ui.Spacer(c)
					x := ui.Box(c).Width(24).Height(24).Radius(6).Center()
					if x.Clicked() {
						a.closeFeedback()
					}
					x.Children(func() {
						ui.Icon(c, iconClose).FontSize(14).TextColor(theme.Gray500)
					})
				})
				if a.fbView == "submit" {
					a.feedbackSubmitView(c)
				} else {
					a.feedbackListView(c)
				}
			})
		})
	})
}

// feedbackSubmitView 提交视图：类型胶囊 + 内容 TextArea + 截图选择 + 底部按钮。
func (a *app) feedbackSubmitView(c *ui.Context) {
	// 反馈类型
	ui.Text(c, "反馈类型").FontSize(12).TextColor(theme.Gray800)
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		for _, ft := range feedbackTypes {
			ft := ft
			btn := ui.Box(c).Grow(1).Height(34).Radius(8).Center().
				BorderWidth(1).BorderColor(theme.Gray200)
			bg, fg := theme.White, theme.Gray600
			if a.fbType == ft.value {
				bg, fg = theme.Primary50, theme.Primary
			}
			btn = btn.Background(bg)
			if btn.Clicked() {
				a.fbType = ft.value
			}
			btn.Children(func() {
				ui.Text(c, ft.label).FontSize(13).TextColor(fg)
			})
		}
	})
	// 反馈内容
	ui.Text(c, "反馈内容").FontSize(12).TextColor(theme.Gray800)
	ui.Box(c).FillWidth().Height(130).Radius(8).BorderWidth(1).BorderColor(theme.Gray200).
		Background(theme.White).Children(func() {
		ui.TextArea(c, &a.fbContent).FillWidth().FillHeight().FontSize(13).
			BorderWidth(0).Padding(10, 12).
			Placeholder("请详细描述您遇到的问题或建议…")
	})
	ui.Text(c, "请详细描述您遇到的问题或建议，以便我们更好地改进（至少 10 个字）").
		FontSize(11).TextColor(theme.Gray500)
	// 截图（可选）
	ui.Text(c, "截图（可选）").FontSize(12).TextColor(theme.Gray800)
	if a.fbShotPath != "" {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "📎 "+filepath.Base(a.fbShotPath)).FontSize(12).TextColor(theme.Gray800).MaxLines(1)
			ui.Spacer(c)
			rm := ui.Box(c).Height(24).Radius(5).Padding(0, 10).Background(theme.Gray100).Center()
			if rm.Clicked() {
				a.removeFbScreenshot()
			}
			rm.Children(func() {
				ui.Text(c, "移除").FontSize(11).TextColor(theme.Gray600)
			})
		})
	} else {
		up := ui.Box(c).FillWidth().Height(44).Radius(8).Center().
			BorderWidth(1).BorderColor(theme.Gray200).Background(theme.Gray50)
		if up.Clicked() {
			a.pickFbScreenshot()
		}
		up.Children(func() {
			ui.Text(c, "点击上传截图").FontSize(13).TextColor(theme.Gray600)
		})
	}
	if a.fbErr != "" {
		ui.Text(c, a.fbErr).FontSize(12).TextColor(theme.ErrorText).MaxLines(3)
	}
	// 底部：我的反馈 | 取消 | 提交
	ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
		link := ui.Box(c).Height(30).Radius(6).Padding(0, 10).Center()
		if link.Clicked() {
			a.switchFbList()
		}
		link.Children(func() {
			ui.Text(c, "我的反馈").FontSize(12).TextColor(theme.Primary)
		})
		ui.Spacer(c)
		cancel := ui.Box(c).Height(32).Radius(8).Padding(0, 16).
			Background(theme.Gray100).Center()
		if cancel.Clicked() {
			a.closeFeedback()
		}
		cancel.Children(func() {
			ui.Text(c, "取消").FontSize(13).TextColor(theme.Gray600)
		})
		disabled := !a.canSubmitFeedback() || a.fbBusy
		label := "提交反馈"
		if a.fbBusy {
			label = "提交中..."
		}
		submit := ui.Box(c).Height(32).Radius(8).Padding(0, 16).Center()
		bg := theme.Primary
		fg := ui.Color(theme.White)
		if disabled {
			bg = theme.Gray300
			fg = ui.Color(theme.White)
		}
		submit = submit.Background(bg)
		if !disabled && submit.Clicked() {
			a.submitFeedback()
		}
		submit.Children(func() {
			ui.Text(c, label).FontSize(13).TextColor(fg)
		})
	})
}

// feedbackListView 我的反馈列表：时间/类型/状态标签，点开看回复。
func (a *app) feedbackListView(c *ui.Context) {
	ui.Column(c).FillWidth().Grow(1).Gap(8).Children(func() {
		if a.fbLoading && len(a.fbList) == 0 {
			ui.Text(c, "加载中...").FontSize(12).TextColor(theme.Gray500).Padding(12, 0)
			return
		}
		if len(a.fbList) == 0 {
			ui.Text(c, "暂无反馈记录").FontSize(12).TextColor(theme.Gray500).Padding(12, 0)
			return
		}
		// 列表区给固定高度：自适应容器里 List 的视口会塌缩，只画得出一条。
		ui.Box(c).FillWidth().Height(320).Children(func() {
			a.fbListState.Key = func(i int) any { return a.fbList[i].ID }
			a.fbListState.FollowEnd = false
			ui.List(c, &a.fbListState, len(a.fbList), func(i int) {
				a.feedbackRow(c, a.fbList[i])
			}).Grow(1)
		})
		// 返回提交视图
		back := ui.Box(c).Height(30).Radius(6).Padding(0, 12).Background(theme.Gray100).Center()
		if back.Clicked() {
			a.fbView = "submit"
			a.fbErr = ""
		}
		back.Children(func() {
			ui.Text(c, "返回提交").FontSize(12).TextColor(theme.Gray600)
		})
	})
}

// feedbackRow 单条反馈：头部（时间/类型/状态）+ 内容 + 展开的管理员回复。
func (a *app) feedbackRow(c *ui.Context, fb qapi.UserFeedback) {
	expanded := a.fbExpanded == fb.ID
	ui.Column(c).FillWidth().Radius(8).BorderWidth(1).BorderColor(theme.Gray200).
		Padding(10, 12).Gap(6).Children(func() {
		row := ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center)
		if row.Clicked() {
			if expanded {
				a.fbExpanded = 0
			} else {
				a.fbExpanded = fb.ID
			}
		}
		row.Children(func() {
			ui.Text(c, fbTypeLabel(fb.Type)).FontSize(12).TextColor(theme.Gray800)
			ui.Text(c, fbTimeLabel(fb.CreatedAt)).FontSize(11).TextColor(theme.Gray500)
			ui.Spacer(c)
			// 状态标签：主色浅底胶囊（对齐 feedback-status-tag）
			ui.Box(c).Radius(4).Padding(1, 8).Background(theme.Primary50).Children(func() {
				ui.Text(c, fbStatusLabel(fb.Status)).FontSize(10).TextColor(theme.Primary)
			})
		})
		ui.Text(c, fb.Content).FontSize(12).TextColor(theme.Gray800).MaxLines(3)
		if expanded && fb.Reply != "" {
			ui.Column(c).FillWidth().Radius(6).Background(theme.Gray50).Padding(8, 10).Gap(4).Children(func() {
				name := "管理员回复"
				if fb.HandlerName != "" {
					name = "管理员回复（" + fb.HandlerName + "）"
				}
				ui.Text(c, name).FontSize(11).TextColor(theme.Primary)
				ui.Text(c, fb.Reply).FontSize(12).TextColor(theme.Gray800)
			})
		} else if expanded && fb.Reply == "" && fb.Status == "pending" {
			ui.Text(c, "等待处理中...").FontSize(11).TextColor(theme.Gray500)
		}
	})
}
