package client

// /quick 快捷回复（对齐 qim-client utils/slashCommands/quickCommand.ts + api/quickReplies.ts）。
//
// 触发：输入框以 /quick 开头（后面可带过滤词）→ 打开短语面板。
// 数据：GET /user-settings/quick_replies；未自定义（has_value=false、非数组、
// 空数组）或请求失败时兜底 DEFAULT_QUICK_REPLIES，断网也能用核心短语。
// 选中：把输入整体替换成「短语 + 空格」（对齐 getInsertText），用户可改可发。
// 与 /note 的区别：/quick 只插入文本，不直接发送。

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// quickRepliesKey 是后端 user_settings 表里的 setting_key。
const quickRepliesKey = "quick_replies"

// defaultQuickReplies 用户未自定义时的兜底短语（照搬 qim-client，按使用频率粗排）。
var defaultQuickReplies = []string{
	"收到", "好的", "明白", "了解", "没问题",
	"稍后回复", "正在处理", "马上看",
	"谢谢", "感谢", "辛苦了",
	"抱歉", "不好意思", "久等了",
	"请确认", "请查收", "帮忙看一下",
	"好的就这样", "晚点再聊", "先这样",
}

// parseQuickReplies 把 GET 响应的 value 解析成短语数组。
// 未设置 / 非数组 / 空数组 → 返回默认短语（复制一份，避免调用方误改常量）。
func parseQuickReplies(v *qapi.UserSettingValue) []string {
	out := []string{}
	if v != nil && v.HasValue && len(v.Value) > 0 {
		var items []string
		if err := json.Unmarshal(v.Value, &items); err == nil {
			for _, s := range items {
				if strings.TrimSpace(s) == "" {
					continue
				}
				out = append(out, s)
			}
		}
	}
	if len(out) == 0 {
		return append(out, defaultQuickReplies...)
	}
	return out
}

// quickTriggerText 判断输入是否处于命令触发态，返回面板过滤词。
// 只有 /quick 本身或 /quick 后跟空白才算，/quickly 之类不算。
// 裸 / 也触发（对齐 SlashCommandHint「试一下」插入 / 后弹命令列表）；
// 本端只实现了短语命令，文件 / 笔记 / 任务按需求搁置，故面板仍是短语列表。
func quickTriggerText(input string) (query string, ok bool) {
	t := strings.TrimLeft(input, " \t")
	if t == "/" {
		return "", true
	}
	if strings.HasPrefix(t, "/ ") {
		return strings.TrimSpace(t[2:]), true
	}
	const trig = "/quick"
	if !strings.HasPrefix(t, trig) {
		return "", false
	}
	rest := t[len(trig):]
	if rest == "" {
		return "", true
	}
	if rest[0] != ' ' && rest[0] != '\t' && rest[0] != '\n' && rest[0] != '\r' {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

// maybeTriggerSlash 每帧检查输入：/quick 态就开面板（首次打开拉一次数据），
// 离开触发态就关面板。语义与 maybeTriggerMention 一致。
func (a *app) maybeTriggerSlash() {
	_, ok := quickTriggerText(a.input)
	if !ok {
		if a.quickOpen || a.quickBusy {
			a.quickOpen = false
			a.quickBusy = false
			a.quickItems = nil
		}
		return
	}
	if a.quickOpen {
		return
	}
	a.quickOpen = true
	if a.client == nil || !a.loggedIn {
		// headless / 未登录：直接用默认短语，别转圈
		a.quickItems = append([]string{}, defaultQuickReplies...)
		a.quickBusy = false
		return
	}
	a.quickBusy = true
	ctx := a.ctx
	go func() {
		v, err := a.client.GetUserSetting(ctx, quickRepliesKey)
		if err != nil {
			log.Printf("[quick] 拉取快捷回复失败: %v", err)
		}
		items := parseQuickReplies(v)
		a.mutate(func() {
			if !a.quickOpen {
				return // 面板已关，丢弃迟到响应
			}
			a.quickItems = items
			a.quickBusy = false
		})
	}()
}

// filteredQuickReplies 按过滤词筛选短语（包含匹配，忽略大小写）。
func (a *app) filteredQuickReplies() []string {
	query, ok := quickTriggerText(a.input)
	if !ok {
		return nil
	}
	items := a.quickItems
	if len(items) == 0 && !a.quickBusy {
		items = defaultQuickReplies
	}
	q := strings.ToLower(query)
	if q == "" {
		return items
	}
	out := make([]string, 0, len(items))
	for _, s := range items {
		if strings.Contains(strings.ToLower(s), q) {
			out = append(out, s)
		}
	}
	return out
}

// applyQuickReply 采纳一条短语：输入整体替换为「短语 + 空格」，面板关闭。
func (a *app) applyQuickReply(text string) {
	a.input = strings.TrimRight(text, " \t\n") + " "
	a.quickOpen = false
	a.quickBusy = false
	a.quickItems = nil
}

// closeQuick 关闭面板并清状态（发送失败等原因要关时用）。
func (a *app) closeQuick() {
	a.quickOpen = false
	a.quickBusy = false
	a.quickItems = nil
}

// quickPanel 是 /quick 短语面板，位置与 @ 提及面板一致（composer 输入区上方一行，
// 流式布局）。样式对齐 qim-client 的 SlashCommandPanel：白底圆角卡片 + 短语行。
// 高度封顶 260px，短语多时列表内滚动——面板是流式元素，不封顶会把输入框挤出屏幕。
func (a *app) quickPanel(c *ui.Context) {
	if !a.quickOpen {
		return
	}
	ui.Row(c).FillWidth().Padding(0, 15, 4, 15).Children(func() {
		ui.Column(c).FillWidth().MaxHeight(260).Radius(8).BorderWidth(1).BorderColor(theme.Gray200).
			Background(theme.White).Children(func() {
			title := "快速回复（/ 或 /quick）"
			if a.quickBusy {
				title = "快速回复 · 加载中…"
			}
			ui.Text(c, title).FontSize(11).TextColor(theme.Gray500).Padding(6, 12)
			hits := a.filteredQuickReplies()
			if len(hits) == 0 {
				ui.Text(c, "没有匹配的短语").FontSize(12).TextColor(theme.Gray500).Padding(4, 12, 8, 12)
				return
			}
			a.quickList.Key = func(i int) any { return hits[i] }
			a.quickList.FollowEnd = false
			ui.List(c, &a.quickList, len(hits), func(i int) {
				a.quickRow(c, hits[i])
			}).Grow(1)
			ui.Text(c, "点击或回车插入第一条，可编辑后再发送").FontSize(11).TextColor(theme.Gray500).
				Padding(4, 12, 6, 12)
		})
	})
}

func (a *app) quickRow(c *ui.Context, text string) {
	row := ui.Row(c).FillWidth().Height(30).Padding(0, 12).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Primary50)
	}
	if row.Clicked() {
		a.applyQuickReply(text)
	}
	row.Children(func() {
		ui.Text(c, text).FontSize(13).TextColor(theme.Gray800).MaxLines(1)
	})
}
