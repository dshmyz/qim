package client

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"qim-newclient/internal/theme"
)

// AI 助手：右下角悬浮球 + 右侧 360px 侧边栏。
// 对齐 qim-client FloatingAIBall.vue / AISidebarPanel.vue 的最小闭环：
// 独立输入框、空态 4 张推荐卡、流式逐字回复、停止生成、上下文关联开关。
// 侧边栏从 y=40 开始，给窗口三色灯让位（原版 Electron 的标题栏结构不同）。

// aiBall 右下角 52px 主色圆钮。侧边栏打开时仍保留，点它可收起（原版是 toggle）。
func (a *app) aiBall(c *ui.Context) {
	// 原版 FloatingAIBall 默认从右下角上抬 120px，避免压住输入框发送按钮。
	ball := ui.Box(c).Width(52).Height(52).Radius(26).Background(theme.Primary).Center().
		Absolute().Right(24).Bottom(120)
	if ball.Hovered() {
		ball.Background(theme.PrimaryStrong)
	}
	if ball.Clicked() {
		a.aiToggle()
	}
	ball.Children(func() {
		ui.Icon(c, iconBot).FontSize(22).TextColor(theme.White)
	})
}

func (a *app) aiSidebar(c *ui.Context) {
	if !a.aiOpen {
		return
	}
	panel := ui.Column(c).Width(360).Absolute().Right(0).Top(40).Bottom(0).
		Background(theme.White).BorderWidth(0, 0, 0, 1).BorderColor(theme.Gray200)
	panel.Children(func() {
		a.aiHeader(c)
		a.aiContextBar(c)
		ui.Scroll(c).Grow(1).FillWidth().Padding(16, 16).Children(func() {
			if len(a.aiTurns) == 0 {
				a.aiEmptyState(c)
				return
			}
			ui.Column(c).FillWidth().Gap(12).Children(func() {
				for _, turn := range a.aiTurns {
					a.aiBubble(c, turn)
				}
			})
		})
		a.aiInputArea(c)
	})
}

func (a *app) aiHeader(c *ui.Context) {
	ui.Row(c).FillWidth().Height(56).Padding(0, 16).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
		ui.Icon(c, iconBot).FontSize(17).TextColor(theme.Primary)
		ui.Text(c, "AI 助手").FontSize(15).FontWeight(600).TextColor(theme.Gray800)
		ui.Spacer(c)
		aiIconButton(c, iconTrash, "清空对话", a.aiClear)
		aiIconButton(c, iconClose, "关闭", a.aiClose)
	})
}

// aiContextBar 显示当前关联的会话；点一下切到「自由对话」，
// 再点回来恢复关联。对应原版 .ai-sidebar-context 的解除/重新关联。
func (a *app) aiContextBar(c *ui.Context) {
	bar := ui.Row(c).FillWidth().Height(34).Padding(0, 16).Gap(6).AlignItems(ui.Center).
		Background(theme.Primary50)
	label := "自由对话 · 点击关联当前会话"
	if name := a.aiContextName(); name != "" {
		label = "我已了解「" + truncateRunes(name, 12) + "」的上下文 · 点击解绑"
	}
	if bar.Clicked() {
		a.aiToggleScope()
	}
	bar.Children(func() {
		ui.Icon(c, iconLink).FontSize(12).TextColor(theme.Primary)
		ui.Text(c, label).FontSize(12).TextColor(theme.Primary)
	})
}

func (a *app) aiEmptyState(c *ui.Context) {
	ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(28, 0, 0, 0).Children(func() {
		av := ui.Box(c).Width(48).Height(48).Radius(14).Background(theme.PrimaryBg).Center()
		av.Children(func() {
			ui.Icon(c, iconBot).FontSize(24).TextColor(theme.Primary)
		})
		ui.Text(c, "有什么可以帮你？").FontSize(15).FontWeight(600).TextColor(theme.Gray800)
		ui.Text(c, "基于当前会话上下文回答，也可以自由对话").FontSize(12).TextColor(theme.Muted)
		// 原版是 2 列 grid，mygo 没有 grid 布局，用两行 Row 等分实现。
		ui.Column(c).FillWidth().Gap(8).Padding(16, 0, 0, 0).Children(func() {
			for i := 0; i < len(aiOpenSuggestions); i += 2 {
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					for _, s := range aiOpenSuggestions[i : i+2] {
						a.aiSuggestionCard(c, s[0], s[1])
					}
				})
			}
		})
	})
}

func (a *app) aiSuggestionCard(c *ui.Context, title, prompt string) {
	card := ui.Box(c).Grow(1).Height(58).Radius(10).Background(theme.White).
		BorderWidth(1).BorderColor(theme.Gray200).Padding(10, 12)
	if card.Hovered() {
		card.BorderColor(theme.Primary).Background(theme.Primary50)
	}
	if card.Clicked() {
		a.aiSend(prompt)
	}
	card.Children(func() {
		ui.Column(c).Gap(5).Children(func() {
			ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(theme.Gray800)
			ui.Text(c, truncateRunes(prompt, 14)).FontSize(11).TextColor(theme.Muted)
		})
	})
}

func (a *app) aiBubble(c *ui.Context, turn aiTurn) {
	isUser := turn.Role == "user"
	ui.Row(c).FillWidth().AlignItems(ui.Start).Children(func() {
		if !isUser {
			ui.Spacer(c).Width(28)
		} else {
			ui.Spacer(c).Grow(1)
		}
		box := ui.Box(c).MaxWidth(288).Radius(12).Padding(10, 14)
		if isUser {
			box.Background(theme.Primary)
			box.Children(func() {
				ui.Text(c, turn.Text).FontSize(13).TextColor(theme.White)
			})
		} else {
			box.Background(theme.White).BorderWidth(1).BorderColor(theme.Gray200)
			box.Children(func() {
				if turn.Text == "" && turn.Streaming {
					ui.Text(c, "思考中…").FontSize(13).TextColor(theme.Muted)
				} else {
					a.aiMarkdown(c, turn.Text)
				}
				if turn.Streaming {
					ui.Text(c, "▌").FontSize(13).TextColor(theme.Primary)
				}
			})
		}
		if !isUser {
			ui.Spacer(c).Grow(1)
		}
	})
}

// aiMarkdown 复用聊天区同一套 markdown 渲染（richtext.ParseMarkdown + mdParas），
// 保证 AI 回复里的列表、代码块、链接与主窗口观感一致。
func (a *app) aiMarkdown(c *ui.Context, text string) {
	var paras []richtext.Paragraph
	for _, p := range richtext.ParseMarkdown(text).Paragraphs() {
		paras = append(paras, p)
	}
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		a.mdParas(c, paras, theme.Gray800, theme.Primary, false)
	})
}

func (a *app) aiInputArea(c *ui.Context) {
	ui.Column(c).FillWidth().Padding(10, 12, 12, 12).Children(func() {
		// 持续提示条（来自 /ai/suggested-prompts），空时不占位。
		if len(a.aiPrompts) > 0 && !a.aiBusy {
			ui.Row(c).FillWidth().Children(func() {
				for _, p := range a.aiPrompts[:minInt(2, len(a.aiPrompts))] {
					chip := ui.Box(c).Height(26).Radius(13).Background(theme.Primary50).Padding(0, 12).Center()
					if chip.Clicked() {
						a.aiSend(p)
					}
					chip.Children(func() {
						ui.Text(c, truncateRunes(p, 12)).FontSize(11).TextColor(theme.Primary)
					})
				}
			})
		}
		box := ui.Box(c).FillWidth().Height(88).Radius(10).
			BorderWidth(1).BorderColor(theme.Gray200).Background(theme.White)
		box.Children(func() {
			ta := ui.TextArea(c, &a.aiInput).Placeholder("问我任何事…").FillWidth().FillHeight().
				FontSize(13).BorderWidth(0).Background(theme.White).Padding(10, 12).
				OnTextKey(func(k ui.TextKey) bool {
					if k.Key != ui.KeyEnter || k.Mods != 0 || a.aiBusy {
						return false
					}
					a.aiSend(k.Text())
					k.SetText("")
					return true
				})
			if ta.Focused() || box.Focused() {
				box.BorderColor(theme.Primary)
			}
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(6, 2, 0, 2).Children(func() {
			ui.Text(c, "Enter 发送 · Shift+Enter 换行").FontSize(10).TextColor(theme.Muted)
			ui.Spacer(c).Grow(1)
			if a.aiBusy {
				aiIconButton(c, iconStop, "停止生成", a.aiStop)
			} else {
				aiIconButton(c, iconSend, "发送", func() { a.aiSend(a.aiInput) })
			}
		})
	})
}

func aiIconButton(c *ui.Context, icon *ui.SVG, tip string, onClick func()) {
	b := ui.Box(c).Width(28).Height(28).Radius(6).Center()
	if b.Hovered() {
		b.Background(theme.Primary50)
	}
	if b.Clicked() {
		onClick()
	}
	b.Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(theme.Gray600)
	})
	_ = tip // mygo 没有 tooltip API；保留参数便于后续接入
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
