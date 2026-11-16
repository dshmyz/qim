package client

// @ 提及的输入补全。
//
// 输入框里始终是可读的 "@名字"，只有发送前才由 encodeMentions 换成后端
// 认识的内嵌 token（见 internal/qapi/mention.go）。这样不用在原生 TextArea
// 上模拟前端那套 mentionSpans，也不会让用户看见 @{mention:...}。
// 候选来自已有的 /api/v1/users/search，不需要额外的群成员接口。

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

const maxMentionHits = 6

// activeMention 找输入框末尾正在输入的 @xxx。
// TextArea 只暴露全文、拿不到光标位置，所以只认「末尾的 @token」：
// '@' 前必须是空白或开头（避免邮箱/URL 里的 @），'@' 后不能有空白。
func activeMention(input string) (start int, query string, ok bool) {
	i := strings.LastIndex(input, "@")
	if i < 0 {
		return 0, "", false
	}
	query = input[i+1:]
	if strings.ContainsAny(query, " \t\n\r") || len([]rune(query)) > 20 {
		return 0, "", false
	}
	if i > 0 {
		switch input[i-1] {
		case ' ', '\t', '\n', '\r':
		default:
			return 0, "", false
		}
	}
	return i, query, true
}

// maybeTriggerMention 每帧比较末尾的 @ 词，变了就发起一次搜索。
// seq 作废旧响应，逻辑与三路搜索一致。
func (a *app) maybeTriggerMention() {
	_, q, ok := activeMention(a.input)
	if !ok {
		if a.mentionQuery != "" || len(a.mentionHits) > 0 || a.mentionBusy {
			atomic.AddUint64(&a.mentionSeq, 1)
			a.mentionQuery = ""
			a.lastMentionQ = ""
			a.mentionHits = nil
			a.mentionBusy = false
		}
		return
	}
	a.mentionQuery = q
	if q == a.lastMentionQ {
		return
	}
	// 后端还没就绪（登录中 / headless 渲染）时只记住当前词，不要清空
	// 已有候选，也别把 query 记成「已处理」——否则等 client 一好就再也不搜了。
	if q == "" || a.client == nil || !a.loggedIn {
		a.mentionBusy = false
		return
	}
	a.lastMentionQ = q
	// 群会话优先用已加载的本地成员做候选：即时、无需请求，群外的人本来就 @ 不到。
	// 没有成员（单聊 / 还没拉到详情）才回退全局用户搜索。
	if hits := a.localMentionHits(q); hits != nil {
		atomic.AddUint64(&a.mentionSeq, 1) // 作废正在飞的网络请求
		a.mentionHits = hits
		a.mentionBusy = false
		return
	}
	seq := atomic.AddUint64(&a.mentionSeq, 1)
	a.mentionHits = nil
	a.mentionBusy = true
	go a.runMentionSearch(a.ctx, a.client, q, seq)
}

// localMentionHits 从当前群会话的本地成员里筛出匹配 query 的候选。
// 返回 nil 表示当前没有可用成员（单聊或详情还没加载），调用方改走全局搜索。
func (a *app) localMentionHits(q string) []qapi.SearchUserHit {
	i := a.active
	if i < 0 || i >= len(a.convs) {
		return nil
	}
	mem := a.convs[i].Members
	if len(mem) == 0 {
		return nil
	}
	q = strings.ToLower(q)
	hits := make([]qapi.SearchUserHit, 0, len(mem))
	for _, m := range mem {
		if q == "" || strings.Contains(strings.ToLower(m.Name), q) {
			hits = append(hits, qapi.SearchUserHit{
				ID:       qapi.ID(strconv.Itoa(int(m.UserID))),
				Name:     m.Name,
				Username: m.Name,
				Avatar:   m.Avatar,
				Type:     "user",
			})
		}
		if len(hits) >= maxMentionHits {
			break
		}
	}
	return hits
}

func (a *app) runMentionSearch(parent context.Context, client *qapi.Client, q string, seq uint64) {
	time.Sleep(220 * time.Millisecond)
	if atomic.LoadUint64(&a.mentionSeq) != seq {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	users, err := client.SearchUsers(ctx, q)
	if atomic.LoadUint64(&a.mentionSeq) != seq {
		return
	}
	hits := make([]qapi.SearchUserHit, 0, maxMentionHits)
	for _, u := range users {
		if u.ID.String() == "" || (u.Type != "" && u.Type != "user") {
			continue
		}
		hits = append(hits, u)
		if len(hits) >= maxMentionHits {
			break
		}
	}
	if err != nil {
		log.Printf("[mention] 搜索联系人 %q 失败: %v", q, err)
	}
	a.mutate(func() {
		if atomic.LoadUint64(&a.mentionSeq) != seq {
			return
		}
		a.mentionHits = hits
		a.mentionBusy = false
	})
}

// applyMention 把末尾的 @query 换成 "@名字"，并记下发送时要用的 token。
func (a *app) applyMention(name, token string) {
	start, _, ok := activeMention(a.input)
	if !ok {
		return
	}
	visible := "@" + name
	a.input = a.input[:start] + visible + " "
	if a.mentionPicks == nil {
		a.mentionPicks = make(map[string]string)
	}
	a.mentionPicks[visible] = token
	atomic.AddUint64(&a.mentionSeq, 1)
	a.mentionQuery = ""
	a.mentionHits = nil
	a.mentionBusy = false
}

func (a *app) applyMentionHit(h qapi.SearchUserHit) {
	name := h.Label()
	if name == "" {
		return
	}
	a.applyMention(name, qapi.EncodeMention(h.ID.String(), name))
}

func (a *app) applyMentionAll() {
	a.applyMention("所有人", qapi.EncodeMentionAll())
}

// insertMention 是信息栏里「点成员头像/名字」触发的 @：把「@昵称 」塞进输入框，
// 并登记发送时要用的后端 token。和键盘打 @ 完全等价，只是入口换成点击：
//   - 输入框末尾正在打 @xxx 时，直接替换那个未完成的 @token；
//   - 否则追加到末尾（前面补一个空格，避免和上一句黏在一起）。
//
// 末尾补的空格让 activeMention 不再匹配，不会弹出候选面板抢输入。
func (a *app) insertMention(m member) {
	token := qapi.EncodeMention(strconv.Itoa(int(m.UserID)), m.Name)
	visible := "@" + m.Name
	if start, _, ok := activeMention(a.input); ok {
		a.input = a.input[:start] + visible + " "
	} else {
		if a.input != "" && !strings.HasSuffix(a.input, " ") {
			a.input += " "
		}
		a.input += visible + " "
	}
	if a.mentionPicks == nil {
		a.mentionPicks = make(map[string]string)
	}
	a.mentionPicks[visible] = token
	atomic.AddUint64(&a.mentionSeq, 1)
	a.mentionQuery = ""
	a.mentionHits = nil
	a.mentionBusy = false
}

// mentionPanel 画在输入框上方，只在末尾有 @token 时出现。
func (a *app) mentionPanel(c *ui.Context) {
	if _, _, ok := activeMention(a.input); !ok {
		return
	}
	if !a.loggedIn {
		return
	}
	ui.Row(c).FillWidth().Padding(0, 15, 4, 15).Children(func() {
		ui.Column(c).FillWidth().Radius(8).BorderWidth(1).BorderColor(theme.Gray200).
			Background(theme.White).Children(func() {
			a.mentionRow(c, "@所有人", "通知会话内全部成员", a.applyMentionAll)
			if a.mentionBusy {
				ui.Text(c, "  正在搜索联系人…").FontSize(11).TextColor(theme.Gray500).Padding(4, 12)
			}
			for _, h := range a.mentionHits {
				hit := h
				sub := "@" + hit.Username
				if hit.Username == "" {
					sub = "点击提及"
				}
				a.mentionRow(c, hit.Label(), sub, func() { a.applyMentionHit(hit) })
			}
		})
	})
}

func (a *app) mentionRow(c *ui.Context, title, sub string, onPick func()) {
	row := ui.Row(c).FillWidth().Padding(6, 12).Gap(8).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(theme.Primary50)
	}
	if row.Clicked() {
		onPick()
	}
	row.Children(func() {
		ui.Text(c, title).FontSize(12).TextColor(theme.Gray800).MaxLines(1)
		ui.Text(c, sub).FontSize(11).TextColor(theme.Gray500).MaxLines(1)
	})
}
