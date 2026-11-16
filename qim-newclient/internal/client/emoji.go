package client

// emoji 表情面板（对齐 qim-client EmojiPanel.vue）。
//
// qim-client 面板 440×360，底部两个 tab：默认表情（unicode 图片网格）
// 与经典表情（QQ 风格 gif，点击插入 [名称] 文字标记，网页端渲染时替换为图片）。
// 纯原生端没有 twemoji/gif 资产，unicode 表情直接文本渲染；经典表情网格显示
// 名称文字，点击同样插入 [名称] 标记——消息内容契约与 qim-client 完全一致，
// 对端（网页端）收到后照常渲染成 gif。

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/theme"
)

// unicodeEmojis 默认表情（照搬 qim-client EmojiPanel.vue 的 allEmojis）。
var unicodeEmojis = []string{
	"😊", "😂", "❤️", "👍", "🎉", "🔥", "🤔", "😢", "😡", "👏",
	"😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😇",
	"🙂", "🙃", "😉", "😌", "😍", "🥰", "😘", "😗", "😙", "😚",
	"😋", "😛", "😝", "😜", "🤪", "🤨", "🧐", "🤓", "😎", "🤩",
	"🥳", "😏", "😒", "😞", "😔", "😟", "😕", "🙁", "☹️", "😣",
	"😖", "😫", "😩", "🥺", "😭", "😤", "🤬",
	"🤯", "😳", "🥵", "🥶", "😱", "😨", "😰", "😥", "😓", "🤗",
	"🤭", "🤫", "🤥", "😶", "😐", "😑", "😬", "🙄", "😯",
	"😦", "😧", "😮", "😲", "🥱", "😴", "🤤", "😪", "😵", "🤐",
	"🥴", "🤢", "🤮", "🤧", "🤒", "🤕", "🤠",
	"🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐨", "🐯",
	"🦁", "🐮", "🐷", "🐸", "🐵", "🐔", "🐧", "🐦", "🐤", "🐣",
	"🐥", "🦆", "🦅", "🦉", "🦇", "🐺", "🐗", "🐴", "🦄", "🐝",
	"🐛", "🦋", "🐌", "🐞", "🐜", "🐢", "🐍", "🦎",
	"🐙", "🦑", "🦐", "🦞", "🦀", "🐡", "🐠", "🐟",
	"🐬", "🐳", "🐋", "🐊", "🐅", "🐆", "🐈", "🐩",
}

// classicEmojis 经典表情名称（照搬 qim-client utils/classic-emoji.ts，顺序一致）。
// 点击插入 [名称] 标记，对端网页端会渲染成对应 gif。
var classicEmojis = []string{
	"惊讶", "撇嘴", "色", "发呆", "得意", "流泪", "害羞", "闭嘴",
	"睡", "大哭", "尴尬", "发怒", "调皮", "呲牙", "微笑", "难过",
	"酷", "非典", "抓狂", "吐", "偷笑", "可爱", "白眼", "傲慢",
	"饥饿", "困", "惊恐", "流汗", "憨笑", "大兵", "奋斗", "咒骂",
	"疑问", "嘘...", "晕", "折磨", "衰", "骷髅", "敲打", "再见",
	"猪头", "拥抱", "蛋糕", "闪电", "炸弹", "刀", "足球", "便便",
	"咖啡", "饭", "药丸", "玫瑰", "凋谢", "示爱", "爱心", "心碎",
	"礼物", "太阳", "月亮", "强", "弱", "握手", "胜利", "西瓜",
	"冷汗", "擦汗", "抠鼻", "鼓掌", "糗大了", "坏笑", "左哼哼", "右哼哼",
	"哈欠", "鄙视", "委屈", "快哭了", "阴险", "亲亲", "吓", "可怜",
	"菜刀", "啤酒", "篮球", "乒乓", "瓢虫", "抱拳", "勾引", "拳头",
	"差劲", "爱你", "NO", "OK", "招财进宝", "双喜", "鞭炮", "灯笼",
	"发财", "K歌", "购物", "邮件", "帅", "喝彩", "祈祷", "爆筋",
	"棒棒糖", "喝奶", "下面", "香蕉", "飞机", "开车", "高铁左车头", "车厢",
	"高铁右车头", "多云", "下雨", "钞票", "熊猫", "灯泡", "风车", "闹钟",
	"打伞", "彩球", "钻戒", "沙发", "纸巾", "药", "手枪", "青蛙",
	"茶", "眨眼睛", "泪奔", "无奈", "卖萌", "小纠结", "喷血", "斜眼笑",
	"doge", "惊喜", "骚扰", "笑哭", "我最美", "河蟹", "羊驼", "栗子",
	"幽灵", "蛋", "马赛克", "菊花", "肥皂", "红包",
	"拜托", "点赞", "无聊", "托脸", "吃", "送花", "害怕", "花痴",
	"小样儿", "脸红", "飙泪", "我不看", "托腮",
}

// classicMarker 经典表情插入输入框的文字标记，与 qim-client classicMarker 一致。
func classicMarker(name string) string {
	return "[" + name + "]"
}

// toggleEmoji 开/关表情面板（关闭时 tab 复位到默认表情）。
func (a *app) toggleEmoji() {
	a.emojiOpen = !a.emojiOpen
	if !a.emojiOpen {
		a.emojiTab = ""
	}
}

// insertEmoji 把表情文本追加到输入框（对齐 qim-client：面板不自动关闭，
// 连续输入多个表情；草稿随下次切换会话保存）。
func (a *app) insertEmoji(text string) {
	a.input += text
}

// emojiPanel 是表情面板：挂在 view 根的 Absolute 全屏遮罩层，卡片按 qim-client
// 尺寸 440×360，固定在输入区上方（Left 150 / Bottom 210，居中偏下）。
// 关闭方式与其它弹层一致：再点一次工具栏笑脸图标。
func (a *app) emojiPanel(c *ui.Context) {
	if !a.emojiOpen {
		return
	}
	ui.Box(c).FillWidth().FillHeight().Absolute().Left(0).Top(0).
		Background(ui.Hex("#00000022")).Children(func() {
		ui.Column(c).Width(440).Height(360).Radius(14).Background(theme.White).
			BorderWidth(1).BorderColor(theme.Gray200).
			Absolute().Left(150).Bottom(210).Children(func() {
			// 内容区：当前 tab 的表情网格。必须用 ui.List——mygo 没有
			// ClipContent，普通 Column 里的超高内容会溢出卡片画到外面。
			if a.emojiTab == "classic" {
				a.classicGrid(c)
			} else {
				a.unicodeGrid(c)
			}
			// 底部 tabs（qim-client：默认表情 | 经典表情，active 主色浅底）
			ui.Row(c).FillWidth().Height(42).Padding(0, 10).Gap(4).AlignItems(ui.Center).
				BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray200).Children(func() {
				a.emojiTabBtn(c, "默认表情", "")
				a.emojiTabBtn(c, "经典表情", "classic")
			})
		})
	})
}

// emojiTabBtn tab 按钮，active 用主色浅底（对齐 .emoji-tab.active）。
func (a *app) emojiTabBtn(c *ui.Context, label, tab string) {
	active := (tab == "classic") == (a.emojiTab == "classic")
	btn := ui.Box(c).Height(30).Radius(8).Padding(0, 14).Center()
	bg, fg := theme.White, theme.Gray600
	w := 400
	if active {
		bg, fg = theme.Primary50, theme.Primary
		w = 500
	}
	btn = btn.Background(bg)
	if btn.Clicked() {
		a.emojiTab = tab
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(fg).FontWeight(w)
	})
}

// unicodeGrid 默认表情网格：每行 12 格（30px 格 + 4px 间距 ≈ qim-client
// auto-fill），套在 ui.List 里滚动/裁剪。
func (a *app) unicodeGrid(c *ui.Context) {
	const perRow = 12
	rows := (len(unicodeEmojis) + perRow - 1) / perRow
	a.emojiList.Key = func(i int) any { return i }
	a.emojiList.FollowEnd = false
	ui.List(c, &a.emojiList, rows, func(r int) {
		start, end := r*perRow, (r+1)*perRow
		if end > len(unicodeEmojis) {
			end = len(unicodeEmojis)
		}
		ui.Row(c).FillWidth().Padding(12, 12, 0, 12).Gap(4).Children(func() {
			for _, e := range unicodeEmojis[start:end] {
				emoji := e
				cell := ui.Box(c).Width(30).Height(30).Radius(8).Center()
				if cell.Hovered() {
					cell = cell.Background(theme.Gray100)
				}
				if cell.Clicked() {
					a.insertEmoji(emoji)
				}
				cell.Children(func() {
					ui.Text(c, emoji).FontSize(17).Padding(0, 2)
				})
			}
		})
	}).Grow(1)
}

// classicGrid 经典表情网格：一行 5 个名称（无 gif 资产，用 [名称] 文字占位），
// 点击插入 [名称] 标记。套在 ui.List 里滚动/裁剪。
func (a *app) classicGrid(c *ui.Context) {
	const perRow = 5
	rows := (len(classicEmojis) + perRow - 1) / perRow
	a.emojiList.Key = func(i int) any { return i }
	a.emojiList.FollowEnd = false
	ui.List(c, &a.emojiList, rows, func(r int) {
		start, end := r*perRow, (r+1)*perRow
		if end > len(classicEmojis) {
			end = len(classicEmojis)
		}
		ui.Row(c).FillWidth().Padding(12, 12, 0, 12).Gap(4).Children(func() {
			for _, name := range classicEmojis[start:end] {
				n := name
				cell := ui.Box(c).Grow(1).Height(56).Radius(8).Center()
				if cell.Hovered() {
					cell = cell.Background(theme.Gray100)
				}
				if cell.Clicked() {
					a.insertEmoji(classicMarker(n))
				}
				cell.Children(func() {
					ui.Text(c, "["+n+"]").FontSize(12).TextColor(theme.Gray800).MaxLines(1)
				})
			}
		})
	}).Grow(1)
}
