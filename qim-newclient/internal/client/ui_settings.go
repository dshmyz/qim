// 设置页、信息栏开关与占位页。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/theme"
	"strconv"
)

// noticeBar 显示最近一次网络错误，点「×」清掉。不做成自动消失的 toast：
// 发消息失败这种事必须让用户看见并自己确认。
func (a *app) noticeBar(c *ui.Context) {
	ui.Row(c).FillWidth().Padding(6, 20).Gap(10).AlignItems(ui.Center).
		Background(theme.DangerBg).BorderWidth(0, 0, 1, 0).BorderColor(theme.DangerBd).Children(func() {
		ui.Text(c, a.notice).FontSize(12).TextColor(theme.ErrorText).MaxLines(2)
		ui.Spacer(c)
		btn := ui.Box(c).Width(22).Height(22).Radius(4).Center()
		if btn.Clicked() {
			a.notice = ""
		}
		btn.Children(func() {
			ui.Icon(c, iconClose).FontSize(12).TextColor(theme.Gray500)
		})
	})
}

// roleBadge 把后端角色映射成中文短标签，普通成员返回空（不画徽标）。
func roleBadge(role string) string {
	switch role {
	case "owner":
		return "群主"
	case "admin":
		return "管理"
	default:
		return ""
	}
}

// 原版没有黄色「群公告」横幅：公告只在 ChatHeader 里内联成一行胶囊，
// 编辑入口在头部「更多」菜单。这里不再渲染独立横幅，避免顶部平白多一条。

// canEditAnnouncement 判断当前用户能否编辑当前会话的群公告：
// 群/讨论组 + 成员列表里自己的角色是 owner 或 admin。
// 成员未加载时按可编辑处理（显示入口），由后端最终校验——失败会提示并回滚。
func (a *app) canEditAnnouncement() bool {
	if a.active < 0 || a.active >= len(a.convs) {
		return false
	}
	conv := a.convs[a.active]
	if !conv.IsGroupish() {
		return false
	}
	if len(conv.Members) == 0 {
		return true
	}
	selfUID, err := strconv.Atoi(a.selfID)
	if err != nil {
		return false
	}
	for _, m := range conv.Members {
		if m.UserID == uint(selfUID) {
			return m.Role == "owner" || m.Role == "admin"
		}
	}
	return false
}

// toggleRow 是信息栏里一行可点击的开关（标题 + 当前状态），点击执行 onToggle。
func (a *app) toggleRow(c *ui.Context, label string, on bool, onToggle func()) {
	row := ui.Row(c).FillWidth().Padding(10, 16, 4, 16).Gap(6).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Gray100)
	}
	if row.Clicked() {
		onToggle()
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(theme.Gray800)
		ui.Spacer(c)
		ui.Text(c, onOff(on)).FontSize(13).TextColor(theme.Gray500)
	})
}

func yesNo(v bool) string {
	if v {
		return "是"
	}
	return "否"
}

// onOff 把布尔渲染成「开/关」，比「是/否」更贴近开关语义（设置页与信息栏共用）。
func onOff(v bool) string {
	if v {
		return "开"
	}
	return "关"
}

// appVersion 仅在设置页「关于」展示，不接入编译期注入（够用且零依赖）。
const appVersion = "qim-newclient 0.1.0"

// settingsView 是 P4 设置页：登录后整屏替代三栏。含通知开关、账号信息、关于。
// 顶部保留 titleStrip 拖拽条，下面自己的返回栏 + 分组卡片。
func (a *app) settingsView(c *ui.Context) {
	ui.Column(c).Fill().Background(theme.Gray50).Children(func() {
		ui.Row(c).FillWidth().Height(56).Padding(0, 20).Gap(12).AlignItems(ui.Center).
			Background(theme.White).BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
			back := ui.Box(c).Height(32).Radius(6).Padding(0, 12).Center()
			if back.Clicked() {
				a.settingsOpen = false
			}
			back.Children(func() {
				ui.Text(c, "← 返回").FontSize(13).TextColor(theme.Primary)
			})
			ui.Text(c, "设置").FontWeight(600).FontSize(16).TextColor(theme.Gray800)
		})
		ui.Column(c).Grow(1).FillWidth().Padding(24, 24).Gap(16).Children(func() {
			ui.Text(c, "通知").FontSize(13).FontWeight(600).TextColor(theme.Gray500)
			ui.Box(c).FillWidth().Radius(10).Background(theme.White).Padding(4, 16).Children(func() {
				a.toggleRow(c, "新消息桌面通知", a.notifyOnNew, a.toggleNotify)
			})
			ui.Text(c, "账号").FontSize(13).FontWeight(600).TextColor(theme.Gray500)
			ui.Box(c).FillWidth().Radius(10).Background(theme.White).Padding(4, 16).Children(func() {
				a.infoLine(c, "当前账号", a.displayName())
				a.infoLine(c, "服务器", a.loginServer)
				a.statusLine(c)
				a.dangerLine(c, "退出登录", a.logout)
			})
			ui.Text(c, "关于").FontSize(13).FontWeight(600).TextColor(theme.Gray500)
			ui.Box(c).FillWidth().Radius(10).Background(theme.White).Padding(4, 16).Children(func() {
				a.infoLine(c, "客户端", appVersion)
				a.actionLine(c, "检查更新", a.checkUpdate)
			})
			ui.Text(c, "帮助").FontSize(13).FontWeight(600).TextColor(theme.Gray500)
			ui.Box(c).FillWidth().Radius(10).Background(theme.White).Padding(4, 16).Children(func() {
				a.actionLine(c, "问题反馈", a.openFeedback)
			})
		})
	})
}

// infoLine 是设置页里一条只读信息：左侧标签 + 右侧值。
func (a *app) infoLine(c *ui.Context, label, value string) {
	ui.Row(c).FillWidth().Padding(10, 16, 4, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(theme.Gray500).Width(72)
		ui.Text(c, value).FontSize(13).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…")
	})
}

// actionLine 是设置页里一行可点操作：左侧标签 + 右侧「›」，整行可点。
func (a *app) actionLine(c *ui.Context, label string, onClick func()) {
	row := ui.Row(c).FillWidth().Padding(10, 16, 4, 16).Gap(12).AlignItems(ui.Center)
	if row.Clicked() && onClick != nil {
		onClick()
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(theme.Gray800)
		ui.Spacer(c)
		ui.Icon(c, iconChevRight).FontSize(13).TextColor(theme.Gray500)
	})
}

// statusLine 设置页的连接状态：左侧标签 + 右侧带色状态文字。原版把状态点
// 放在顶栏，我们挪进设置页（顶栏只留 logo + 三色灯），颜色仍按连接态区分。
func (a *app) statusLine(c *ui.Context) {
	ui.Row(c).FillWidth().Padding(10, 16, 4, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "连接状态").FontSize(13).TextColor(theme.Gray500).Width(72)
		ui.Text(c, "● "+a.connStatus.String()).FontSize(13).TextColor(a.statusColor())
	})
}

// dangerLine 设置页的破坏性操作行：红字 + 右箭头，点击即执行。
func (a *app) dangerLine(c *ui.Context, label string, onClick func()) {
	row := ui.Row(c).FillWidth().Padding(10, 16, 4, 16).Gap(12).AlignItems(ui.Center)
	if row.Clicked() && onClick != nil {
		onClick()
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(ui.Hex("#e53935"))
		ui.Spacer(c)
		ui.Icon(c, iconChevRight).FontSize(13).TextColor(ui.Hex("#e53935"))
	})
}

// toggleNotify 切换新消息桌面通知开关，设置页与信息栏共用。
func (a *app) toggleNotify() {
	a.notifyOnNew = !a.notifyOnNew
}

// panelHeader 未选会话时的标题栏，高 56（从全局拖动条下方开始）；
// 左侧是蓝色展开图标 + 标题 16/600（对齐原版，无背景块）。
func (a *app) panelHeader(c *ui.Context) {
	ui.Row(c).FillWidth().Height(56).Padding(0, 20).Gap(14).AlignItems(ui.Center).
		Background(theme.White).BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
		ui.Icon(c, iconExpand).FontSize(16).TextColor(theme.Primary)
		ui.Text(c, "最近会话").FontWeight(600).FontSize(16).TextColor(theme.Gray800)
		dragSpacer(c)
	})
}

// placeholder 未选择会话时的占位页，对齐 .empty-state：
// 底色 #f5f5f5、72px 灰色双气泡、间距 32、提示文字 14；右下角 AI 悬浮按钮。
func (a *app) placeholder(c *ui.Context) {
	ui.Column(c).Grow(1).FillWidth().Background(theme.Gray50).Children(func() {
		ui.Spacer(c)
		ui.Column(c).FillWidth().AlignItems(ui.Center).Gap(32).Children(func() {
			ui.Icon(c, iconComments).FontSize(72).TextColor(theme.Gray300)
			ui.Text(c, "选择一个会话开始聊天").FontSize(14).TextColor(theme.Gray500)
		})
		ui.Spacer(c)
		// 右下角悬浮 AI 按钮：52 蓝圆 + 76 浅蓝光圈，距边 20（原版实测）
		ui.Box(c).Width(76).Height(76).Radius(38).Background(theme.Primary50).Absolute().
			Right(20).Bottom(20).Center().Shadow(0, 8, 24, 0, ui.Hex("#3385ff40")).Children(func() {
			ui.Box(c).Width(52).Height(52).Radius(26).Background(theme.Primary).Center().Children(func() {
				ui.Icon(c, iconBot).FontSize(24).TextColor(theme.White)
			})
		})
	})
}
