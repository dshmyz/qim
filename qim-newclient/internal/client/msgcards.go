package client

// 消息卡片弹层：合并转发的「聊天记录」查看弹窗、share 卡片的原文查看弹窗。
// 与 feedbackModal 同款形态：全屏 Absolute 遮罩 + 居中白卡。

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// openMergedRecord 打开合并转发的完整记录弹窗。
func (a *app) openMergedRecord(mf *qapi.MergedForward) {
	a.mfView = mf
	a.mfOpen = true
}

func (a *app) closeMergedRecord() {
	a.mfOpen = false
	a.mfView = nil
}

// openShareOriginal 打开 share 卡片的原文查看弹窗。
func (a *app) openShareOriginal(sh *qapi.ShareCard) {
	a.shareView = sh
	a.shareOpen = true
}

func (a *app) closeShareOriginal() {
	a.shareOpen = false
	a.shareView = nil
}

// mergedRecordModal 聊天记录弹窗：标题 + 全量条目（发送者/时间/预览）。
// 条目上限 20，对齐网页端 MergedForwardRecordDialog 的预览上限。
func (a *app) mergedRecordModal(c *ui.Context) {
	if !a.mfOpen || a.mfView == nil {
		return
	}
	mf := a.mfView
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(480).Height(460).Radius(12).Background(theme.White).
				Padding(20, 20).Gap(10).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Text(c, "💬 "+mf.Title).FontSize(15).FontWeight(600).TextColor(theme.Gray800)
					ui.Spacer(c)
					x := ui.Box(c).Width(24).Height(24).Radius(6).Center()
					if x.Clicked() {
						a.closeMergedRecord()
					}
					x.Children(func() {
						ui.Text(c, "✕").FontSize(12).TextColor(theme.Gray500)
					})
				})
				n := len(mf.Messages)
				if n > 20 {
					n = 20
				}
				a.mfList.Key = func(i int) any { return mf.Messages[i].ID + strconv.Itoa(i) }
				ui.List(c, &a.mfList, n, func(i int) {
					it := mf.Messages[i]
					ui.Column(c).FillWidth().Gap(3).Padding(8, 4).BorderWidth(0, 0, 1, 0).
						BorderColor(theme.Gray200).Children(func() {
						ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Text(c, it.SenderName).FontSize(12).FontWeight(500).TextColor(theme.Gray600)
							ui.Spacer(c)
							ui.Text(c, stampLabel(it.Timestamp)).FontSize(11).TextColor(theme.Gray500)
						})
						ui.Text(c, qapi.ItemPreview(it)).FontSize(13).TextColor(theme.Gray800).MaxLines(3)
					})
				}).Grow(1)
				if len(mf.Messages) > 20 {
					ui.Text(c, fmt.Sprintf("仅展示前 20 条，共 %d 条", len(mf.Messages))).
						FontSize(11).TextColor(theme.Gray500)
				}
			})
		})
	})
}

// shareOriginalModal share 原文弹窗：类型徽标 + 名称 + 原文。
func (a *app) shareOriginalModal(c *ui.Context) {
	if !a.shareOpen || a.shareView == nil {
		return
	}
	sh := a.shareView
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(420).MaxHeight(360).Radius(12).Background(theme.White).
				Padding(20, 20).Gap(8).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Text(c, qapi.ShareKindLabel(sh.Type)+"分享").FontSize(15).FontWeight(600).TextColor(theme.Gray800)
					ui.Spacer(c)
					x := ui.Box(c).Width(24).Height(24).Radius(6).Center()
					if x.Clicked() {
						a.closeShareOriginal()
					}
					x.Children(func() {
						ui.Text(c, "✕").FontSize(12).TextColor(theme.Gray500)
					})
				})
				ui.Text(c, sh.Name).FontSize(13).FontWeight(500).TextColor(theme.Gray800)
				ui.Text(c, sh.Original).FontSize(13).TextColor(theme.Gray600).LineHeight(1.6)
			})
		})
	})
}
