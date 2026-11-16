package client

// mdview：markdown 类型消息的气泡（对齐 qim-client MarkdownMessage）。
// 解析用 mygo 内置的 richtext.ParseMarkdown（CommonMark+GFM），渲染自己
// 做：richtext.Editor 在 ui.List 的虚拟化行里固有宽高都不可靠（按视口
// 惰性排版，行内只给出尾部一截），普通文本元素的固有尺寸才是准的。

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

const mdBaseSize = float32(14)

// mdBubble 渲染 markdown 消息。正文用原文（rawContent），气泡样式与
// textBubble 同底色同圆角；列表预览/复制全文用的已剥语法摘要不变。
func (a *app) mdBubble(c *ui.Context, m message) {
	bg, fg := theme.Gray100, theme.Gray800
	if m.Mine {
		bg, fg = selfBubbleBG(), selfBubbleFG()
	}
	link := theme.Primary
	if m.Mine {
		link = selfBubbleLink()
	}
	src, _ := qapi.DecodeMentions(m.RawContent)
	var paras []richtext.Paragraph
	for _, p := range richtext.ParseMarkdown(src).Paragraphs() {
		paras = append(paras, p)
	}
	// markdown 是块级排版，固定 560 卡片宽，与网页端 AI 回复一致。
	box := ui.Box(c).Width(560).Radius(12).Background(bg).Padding(12, 14)
	box.Children(func() {
		ui.Column(c).FillWidth().Gap(4).Children(func() {
			a.mdParas(c, paras, fg, link, m.Mine)
		})
	})
	bubble := a.highlightBubble(box, m)
	a.attachMessageMenu(bubble, m)
}

// mdParas 把段落列表按块渲染：连续的代码行合并成一块，连续引用段合并
// 进同一条竖条，其余逐段输出。
func (a *app) mdParas(c *ui.Context, paras []richtext.Paragraph, fg, link ui.Color, mine bool) {
	muted := fg.Alpha(0.65)
	num := 0         // 当前编号列表的计数
	prevNum := false // 上一段是否编号列表（断开则重新从 1 数）
	for i := 0; i < len(paras); {
		p := paras[i]
		switch {
		case p.Style.CodeBlock:
			j := i
			for j < len(paras) && paras[j].Style.CodeBlock {
				j++
			}
			a.mdCodeBlock(c, paras[i:j], fg, mine)
			i = j
		case p.Style.Quote:
			j := i
			for j < len(paras) && paras[j].Style.Quote {
				j++
			}
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Box(c).Width(3).Radius(2).Background(fg.Alpha(0.32)).FillHeight()
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					for _, q := range paras[i:j] {
						a.mdPara(c, q, muted, link, mine, 0, "")
					}
				})
			})
			i = j
		default:
			prefix := ""
			switch p.Style.List {
			case richtext.Bullets:
				prefix = "•  "
				prevNum = false
			case richtext.Numbers:
				if prevNum {
					num++
				} else {
					num = 1
				}
				prevNum = true
				prefix = strconv.Itoa(num) + ". "
			case richtext.Tasks:
				prefix = "☑ "
				if !p.Style.Checked {
					prefix = "☐ "
				}
				prevNum = false
			default:
				prevNum = false
			}
			if isEmptyPara(p) {
				ui.Box(c).Height(6).FillWidth()
			} else {
				a.mdPara(c, p, fg, link, mine, float32(p.Style.Indent)*18, prefix)
			}
			i++
		}
	}
}

func isEmptyPara(p richtext.Paragraph) bool {
	if len(p.Runs) == 0 {
		return true
	}
	for _, r := range p.Runs {
		if r.Text != "" {
			return false
		}
	}
	return true
}

// mdPara 渲染一个段落：标题提字号字重，列表/缩进交给外层行，
// 行内 runs 转 RichText 子元素（链接可点、行内 code 高亮）。
func (a *app) mdPara(c *ui.Context, p richtext.Paragraph, fg, link ui.Color, mine bool, indent float32, prefix string) {
	size := mdBaseSize
	weight := 0
	if p.Style.Heading > 0 {
		size = mdBaseSize * mdHeadingSize(p.Style.Heading)
		weight = 600
	}
	// 自己气泡改成浅底后，代码块/行内 code 与对方用同一套深色 tint；
	// 原来的白色 18% 叠加在浅底上几乎看不见。
	tint := fg.Alpha(0.08)
	_ = mine
	ui.Row(c).FillWidth().Children(func() {
		if indent > 0 {
			ui.Box(c).Width(indent)
		}
		if prefix != "" {
			markFg := fg
			if p.Style.List == richtext.Tasks && p.Style.Checked {
				markFg = link
			}
			ui.Text(c, prefix).FontSize(size - 1).TextColor(markFg)
		}
		ui.RichText(c).Selectable().FillWidth().FontSize(size).LineHeight(1.6).
			TextColor(fg).MaxWidth(560 - 28 - indent).Children(func() {
			for _, r := range p.Runs {
				a.mdRun(c, r, fg, link, tint, size, weight)
			}
		})
	})
}

// mdRun 输出一段行内文本：链接转 ui.Link，行内 code 用等宽字+背景，
// 其余按 weight/斜体/删除线/下划线覆盖。
func (a *app) mdRun(c *ui.Context, r richtext.Run, fg, link, tint ui.Color, size float32, weight int) {
	if r.Text == "" {
		return
	}
	ra := r.Attributes
	if ra.Link != "" {
		ui.Link(c, r.Text, ra.Link).TextColor(link)
		return
	}
	el := ui.Text(c, r.Text)
	if ra.Code {
		el = el.Font("monospace").FontSize(size * 0.92).Background(tint)
	}
	w := ra.Weight
	if weight > w {
		w = weight
	}
	if w >= 600 {
		el = el.FontWeight(w)
	}
	if ra.Italic {
		el = el.Italic()
	}
	if ra.Strikethrough {
		el = el.Strikethrough()
	}
	if ra.Underline {
		el = el.Underline()
	}
	if ra.Color != (ui.Color{}) {
		el = el.TextColor(ra.Color)
	}
}

// mdCodeBlock 是合并后的代码块：等宽小一号字、低透明度底、圆角。
func (a *app) mdCodeBlock(c *ui.Context, lines []richtext.Paragraph, fg ui.Color, mine bool) {
	// 自己气泡改成浅底后，代码块/行内 code 与对方用同一套深色 tint；
	// 原来的白色 18% 叠加在浅底上几乎看不见。
	tint := fg.Alpha(0.08)
	_ = mine
	ui.Box(c).FillWidth().Radius(8).Background(tint).Padding(8, 10).Children(func() {
		ui.Column(c).FillWidth().Gap(1).Children(func() {
			for _, p := range lines {
				ui.Text(c, paraPlain(p)).Font("monospace").FontSize(12.5).
					LineHeight(1.5).TextColor(fg).FillWidth()
			}
		})
	})
}

func paraPlain(p richtext.Paragraph) string {
	s := ""
	for _, r := range p.Runs {
		s += r.Text
	}
	return s
}

// mdHeadingSize 聊天气泡里的标题比文档页收一档（1.8em 起太夸张）。
func mdHeadingSize(level int) float32 {
	sizes := [6]float32{1.35, 1.22, 1.12, 1.05, 1.0, 0.94}
	if level < 1 || level > 6 {
		return 1
	}
	return sizes[level-1]
}
