package client

// 消息级操作：引用回复、右键菜单、撤回 / 删除 / 提醒、多选。
//
// 所有 REST 调用都走 goroutine，结果通过 a.mutate 回主线程；菜单回调本身
// 是在绘制帧里执行的（mygo 的 ContextMenu 语义），所以改界面状态是同步的。

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

const (
	// 后端默认 messageRecallTime=120s（系统配置可改）。这里只用来禁用明显
	// 不可能成功的菜单项，最终仍以后端校验为准。
	recallWindow = 2 * time.Minute
	// 提醒的门槛是 messageRemindTime，默认 3600s。实测真后端对新消息
	// 会直接返回「消息发送未满 60 分钟」；不加上这条，菜单里就会有一个
	// 点了必然失败的选项。系统配置改小的时候后端仍会兜底放行。
	remindMinAge = time.Hour
)

// setQuote 把一条消息挂到输入框上方，下一次发送会带 quoted_message_id。
func (a *app) setQuote(m message) {
	if m.ServerID == "" {
		return
	}
	name := m.Sender
	if name == "" {
		name = "我"
	}
	a.quote = &messageQuote{ID: m.ServerID, Name: name, Text: m.Text}
	a.notice = ""
}

func (a *app) clearQuote() { a.quote = nil }

// copyMessageText 复制解码后的正文（不是带 token 的原始 content）。
func (a *app) copyMessageText(text string) {
	if a.win == nil || text == "" {
		return // headless 渲染测试没有 app 运行时，剪贴板不可用
	}
	mygo.Clipboard.WriteText(text)
}

// canRecall 是撤回菜单的可用条件：自己发的、有服务端 id、没撤回过、
// 且还在时限内。服务端仍会再校验一次。
func (a *app) canRecall(m message) bool {
	if !m.Mine || m.ServerID == "" || m.Recalled || m.Failed || m.Pending || m.Stamp <= 0 {
		return false
	}
	return time.Since(time.UnixMilli(m.Stamp)) <= recallWindow
}

// canRemind 与后端 handler.RemindMessage 的校验对齐：
// 发送者本人 + 单聊 + 消息未读 + 发出已满 messageRemindTime（默认 1 小时）。
func (a *app) canRemind(m message) bool {
	return m.Mine && m.ServerID != "" && !m.Read && !m.Failed && !m.Pending &&
		m.Stamp > 0 && time.Since(time.UnixMilli(m.Stamp)) >= remindMinAge &&
		a.currentConv().Kind == "single"
}

// attachMessageMenu 给消息行 / 气泡挂右键菜单。
// 气泡正文是 Selectable 的，自带编辑菜单会抢走最内层点击，
// 所以行和气泡两处都要挂同一个菜单，否则在文字上右键只会出复制/全选。
func (a *app) attachMessageMenu(e ui.Element, m message) {
	if m.Pill != "" || m.Divider {
		return
	}
	build := func(menu *ui.Menu) {
		// 文案对齐 qim-client MessageContextMenu.vue：用「复制」而不是
		// 「复制全文」，和浏览器默认编辑菜单区分靠的是下面的 EditItems。
		if menu.Item("复制").Chosen() {
			a.copyMessageText(m.Text)
		}
		// 复制代码：仅当消息正文含 ``` 围栏时出现，对齐 qim-client
		// MessageContextMenu 的 hasCodeBlock 条件；复制所有围栏的代码正文。
		if hasCodeBlock(m) && menu.Item("复制代码").Chosen() {
			blocks := extractCodeBlocks(m.RawContent)
			if len(blocks) == 0 {
				blocks = extractCodeBlocks(m.Text)
			}
			a.copyMessageText(strings.Join(blocks, "\n\n"))
		}
		if m.Kind != "" && !m.Att.IsZero() {
			if menu.Item("打开 / 下载").Chosen() {
				a.openAttachment(m.Att)
			}
			if menu.Item("另存为…").Chosen() {
				a.saveAttachmentAs(m.Att)
			}
		}
		if menu.Item("引用").Disabled(m.ServerID == "").Chosen() {
			a.setQuote(m)
		}
		if menu.Item("多选").Chosen() {
			a.startSelectMode(m.Key)
		}
		if menu.Item("转发").Disabled(m.ServerID == "" && m.Text == "").Chosen() {
			a.openForwardMsg(m)
		}
		menu.Separator()
		if menu.Item("撤回").Disabled(!a.canRecall(m)).Chosen() {
			a.recallMessage(m)
		}
		if menu.Item("发送提醒").Disabled(!a.canRemind(m)).Chosen() {
			a.remindMessage(m)
		}
		// 已读详情：仅自己发的已送达消息可查（对齐 qim-client 的 isSelf 限制）
		if menu.Item("查看已读").Disabled(!(m.Mine && m.ServerID != "")).Chosen() {
			a.openReadUsers(m)
		}
		if menu.Item("删除").Disabled(m.ServerID == "").Chosen() {
			a.deleteMessages([]string{m.Key})
		}
		menu.Separator()
		// 覆盖掉 Selectable 的默认菜单后把标准编辑命令补回来。
		menu.EditItems()
	}
	e.ContextMenu(build)
}

// recallMessage 调后端撤回，成功后本地立刻换成撤回占位。
// WS 也会推 message_recalled，重复应用是幂等的。
func (a *app) recallMessage(m message) {
	if a.client == nil || m.ServerID == "" {
		return
	}
	client, ctx, id, key := a.client, a.ctx, m.ServerID, m.Key
	go func() {
		recalled, err := client.RecallMessage(ctx, id)
		if err != nil {
			log.Printf("[action] 撤回失败 %s: %v", id, err)
			a.setNotice("撤回失败：" + friendlyError(err))
			return
		}
		a.mutate(func() {
			for i := range a.msgs {
				if a.msgs[i].Key == key {
					a.msgs[i].Text = recalledLabel(true, "")
					a.msgs[i].Recalled = true
					a.msgs[i].OrigText = recalled.RecallOriginal()
					a.msgs[i].Kind = ""
					a.msgs[i].Att = qapi.Attachment{}
					a.msgs[i].LocalPath = ""
					break
				}
			}
			a.notice = ""
		})
	}()
}

// editRecalled 把撤回消息的原文回填进输入框（对齐 qim-client 的
// 「撤回后重新编辑」：解析 extra.original_content，整体替换当前输入）。
func (a *app) editRecalled(m message) {
	if m.OrigText == "" {
		return
	}
	a.mutate(func() {
		a.input = m.OrigText
		a.notice = ""
	})
}

// deleteMessages 批量删除。服务端按用户软删，逐条发请求：
// 一条失败不影响其它条，最后统一把成功的从本地列表摘掉。
func (a *app) deleteMessages(keys []string) {
	if a.client == nil || len(keys) == 0 {
		return
	}
	type target struct{ key, id string }
	targets := make([]target, 0, len(keys))
	for _, key := range keys {
		for i := range a.msgs {
			if a.msgs[i].Key == key && a.msgs[i].ServerID != "" {
				targets = append(targets, target{key: key, id: a.msgs[i].ServerID})
				break
			}
		}
	}
	if len(targets) == 0 {
		a.notice = "选中的消息还不能删除"
		return
	}
	client, ctx := a.client, a.ctx
	go func() {
		removed := make(map[string]bool, len(targets))
		failed := 0
		for _, t := range targets {
			if err := client.DeleteMessage(ctx, t.id); err != nil {
				log.Printf("[action] 删除失败 %s: %v", t.id, err)
				failed++
				continue
			}
			removed[t.key] = true
		}
		a.mutate(func() {
			removeMessageKeys(&a.msgs, removed)
			if failed > 0 {
				a.notice = "部分消息删除失败"
			} else {
				a.notice = ""
			}
			a.exitSelectMode()
		})
	}()
}

// removeMessageKeys 是纯函数：调用方已经在 mutate 里，不能再嵌一层。
func removeMessageKeys(msgs *[]message, keys map[string]bool) {
	if len(keys) == 0 {
		return
	}
	out := (*msgs)[:0]
	for _, m := range *msgs {
		if !keys[m.Key] {
			out = append(out, m)
		}
	}
	*msgs = out
}

// remindMessage 给未读的单聊消息发一次提醒。后端异步推送结果，
// 这里只确认请求被接受。
func (a *app) remindMessage(m message) {
	if a.client == nil || m.ServerID == "" {
		return
	}
	client, ctx, id := a.client, a.ctx, m.ServerID
	go func() {
		if err := client.RemindMessage(ctx, id); err != nil {
			log.Printf("[action] 提醒失败 %s: %v", id, err)
			a.setNotice("提醒失败：" + friendlyError(err))
			return
		}
		a.setNotice("提醒已发送")
	}()
}

func (a *app) startSelectMode(key string) {
	a.selectMode = true
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}
	if key != "" {
		a.selected[key] = true
	}
}

func (a *app) toggleSelected(key string) {
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}
	if a.selected[key] {
		delete(a.selected, key)
	} else {
		a.selected[key] = true
	}
}

func (a *app) exitSelectMode() {
	a.selectMode = false
	a.selected = nil
}

// selectedKeys 返回稳定的选中顺序（map 遍历顺序随机，操作日志会乱）。
func (a *app) selectedKeys() []string {
	keys := make([]string, 0, len(a.selected))
	for k, on := range a.selected {
		if on {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// selectedText 把选中的消息正文拼成可复制的一段，按列表里的顺序。
func (a *app) selectedText() string {
	if len(a.selected) == 0 {
		return ""
	}
	parts := make([]string, 0, len(a.selected))
	for _, m := range a.msgs {
		if a.selected[m.Key] && m.Pill == "" && !m.Divider {
			who := m.Sender
			if m.Mine {
				who = "我"
			}
			parts = append(parts, who+"："+m.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// clearMessageOps 切会话 / 退出登录时把这一组临时状态一次清干净。
func (a *app) clearMessageOps() {
	a.quote = nil
	a.selectMode = false
	a.selected = nil
	a.pendingUnread = 0
	a.mentionPicks = nil
	a.mentionQuery = ""
	a.lastMentionQ = ""
	a.mentionHits = nil
	a.mentionBusy = false
	a.forwardOpen = false
	a.forwardText = ""
	a.forwardTitle = ""
	a.annEditorOpen = false
	a.annDraft = ""
	a.readUsersOpen = false
	a.readUsersBusy = false
	a.readUsersID = ""
	a.readUsersInfo = nil
	// A 组聊天体验：切会话时关掉 emoji 面板与 /quick 面板（输入被草稿替换）
	a.emojiOpen = false
	a.emojiTab = ""
	a.quickOpen = false
	a.quickBusy = false
	a.quickItems = nil
	// C 组杂项：切会话关掉反馈/更新弹层（全局弹层不跨会话保留）
	a.feedbackOpen = false
	a.updOpen = false
}

// openForward 打开转发面板，把单条消息正文作为待转发内容。
func (a *app) openForward(text, title string) {
	a.forwardText = strings.TrimSpace(text)
	a.forwardTitle = title
	a.forwardReq = nil // 纯文本转发（搜索结果等无原始消息的入口）
	a.forwardOpen = true
}

// openForwardMsg 按原始消息构造转发请求（对齐网页端 buildForwardedMessage）：
//   - text        → "[转发] " 前缀的文本消息
//   - image/file  → 原类型原 content 重发（对端直接看到图片/文件卡片）
//   - merged_forward / share / 其他 → share JSON 包一层
//
// 面板里展示的仍是可读摘要，真正发出去的是 forwardReq。
func (a *app) openForwardMsg(m message) {
	a.forwardText = strings.TrimSpace(m.Text)
	a.forwardTitle = "转发消息"
	a.forwardReq = forwardRequestFor(m)
	a.forwardOpen = true
}

// forwardRequestFor 按消息类型挑转发用的 SendRequest。
func forwardRequestFor(m message) *qapi.SendRequest {
	content := strings.TrimSpace(m.RawContent)
	switch {
	case m.MF != nil && content != "":
		req := qapi.SendRequest{Type: "merged_forward", Content: content}
		return &req
	case m.SH != nil && content != "":
		share := map[string]any{
			"type":            "message",
			"id":              m.ServerID,
			"name":            "转发消息",
			"content":         "转发了消息: " + truncateRunes(m.Text, 20),
			"originalContent": content,
		}
		b, _ := json.Marshal(share)
		req := qapi.SendRequest{Type: "share", Content: string(b)}
		return &req
	case m.Kind == "image", m.Kind == "file", m.Kind == "video", m.Kind == "audio":
		if content != "" {
			req := qapi.SendRequest{Type: m.Kind, Content: content}
			return &req
		}
		return nil
	default:
		if m.RawType == "text" && m.Text != "" {
			req := qapi.TextRequest("[转发] " + m.Text)
			return &req
		}
		return nil
	}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// openMergeForward 把当前选中的多条消息打包成 merged_forward 真协议
// （对齐网页端 createMergedForwardPayload：version 1 + title + messages
// [id/type/content/senderName/timestamp]），不再拼纯文本。
func (a *app) openMergeForward() {
	if len(a.selected) == 0 {
		return
	}
	items := make([]qapi.MergedForwardItem, 0, len(a.selected))
	for _, m := range a.msgs {
		if !a.selected[m.Key] || m.Pill != "" || m.Divider {
			continue
		}
		id := m.ServerID
		if id == "" {
			id = m.Key
		}
		typ := m.RawType
		if typ == "" {
			typ = "text"
		}
		content := m.RawContent
		if content == "" {
			content = m.Text
		}
		items = append(items, qapi.MergedForwardItem{
			ID: id, Type: typ, Content: content,
			SenderName: m.Sender, Timestamp: m.Stamp,
		})
	}
	if len(items) == 0 {
		return
	}
	title := "聊天记录"
	if a.active >= 0 && a.active < len(a.convs) {
		title = a.convs[a.active].Name + " 的聊天记录"
	}
	req := qapi.SendRequest{
		Type:    "merged_forward",
		Content: qapi.BuildMergedForwardPayload(title, items),
	}
	a.forwardText = fmt.Sprintf("合并转发 %d 条消息", len(items))
	a.forwardTitle = "合并转发"
	a.forwardReq = &req
	a.forwardOpen = true
	a.exitSelectMode()
}

// closeForward 收起转发面板并清空待发内容。
func (a *app) closeForward() {
	a.forwardOpen = false
	a.forwardText = ""
	a.forwardTitle = ""
}

// doForward 真正发起转发：记录目标会话名用于反馈，关面板，发请求。
func (a *app) doForward(targetID, targetName string) {
	req := a.forwardReq
	text := a.forwardText
	a.closeForward()
	a.sendForward(targetID, req, text)
	a.setNotice("已转发到 " + targetName)
}

// encodeMentions 发送前把输入框里的 "@名字" 换成后端 token。
// 按名字长度降序替换，避免 "@张" 先把 "@张三" 的前缀吃掉。
func (a *app) encodeMentions(text string) string {
	if len(a.mentionPicks) == 0 || text == "" {
		return text
	}
	names := make([]string, 0, len(a.mentionPicks))
	for name := range a.mentionPicks {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, name := range names {
		text = strings.ReplaceAll(text, name, a.mentionPicks[name])
	}
	return text
}

// messageCount 是给菜单/多选条用的「真实消息数」。
func messageCount(msgs []message) int {
	n := 0
	for _, m := range msgs {
		if m.Pill == "" && !m.Divider {
			n++
		}
	}
	return n
}
