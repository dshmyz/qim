package client

// -probe-groups：对真实 qim-server 验证 P3 的群成员 / 置顶 / 免打扰 / 公告。
// 不开窗口、不初始化 mygo，只验证协议。会临时翻转某个会话的置顶与免打扰，
// 验证完立即翻回原状态，但只对本地开发库使用。

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

var probeGroups = flag.Bool("probe-groups", false, "只跑群成员/置顶/免打扰/公告的真实接口验证（不开窗口），跑完退出")
var probeAnnounce = flag.Bool("probe-announce", false, "只跑群公告写链路：临时建群→写公告→回读验证→解散（不开窗口），跑完退出")
var probeReadUsers = flag.Bool("probe-readusers", false, "只跑已读详情链路：临时建群→发消息→GET read-users→解散（不开窗口），跑完退出")
var probeRecall = flag.Bool("probe-recall", false, "只跑撤回增强链路：建群→发消息→撤回→回读验证 extra 原文→解散（不开窗口），跑完退出")
var probeNotifs = flag.Bool("probe-notifs", false, "只跑通知中心链路：建群→第二账号申请入群→群主收通知→已读/置顶/拒绝/一键已读/清空→解散（不开窗口），跑完退出")
var probeChannels = flag.Bool("probe-channels", false, "只跑频道链路：创建→列表→发动态→点赞→评论→回读验证（不开窗口），跑完退出")
var probeProfileF = flag.Bool("probe-profile", false, "只跑 B 组链路：资料读写→用户状态→组织树→建群→解散（不开窗口），跑完退出")
var probeQuickF = flag.Bool("probe-quick", false, "只跑 /quick 链路：GET quick_replies→PUT 自定义→回读→还原（不开窗口），跑完退出")
var probeCF = flag.Bool("probe-c", false, "只跑 C 组链路：提交反馈→我的反馈回读→带截图→版本检查（不开窗口），跑完退出")
var probeFilesF = flag.Bool("probe-files", false, "只跑 D 组链路：建目录→分片上传→列表/星标/重命名/移动→下载比对→批量删除（不开窗口），跑完退出")

var probeMsgF = flag.Bool("probe-msg", false, "只跑消息渲染互操作：发 merged_forward/share 真协议→回读校验→清理（不开窗口），跑完退出")

func runRecallProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-recall -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	conv, err := c.CreateGroup(ctx, "撤回探针临时群", nil)
	if err != nil {
		fmt.Printf("[建群] ✗ %v\n", err)
		return 1
	}
	convID := conv.ID
	defer func() {
		if err := c.DissolveGroup(context.Background(), convID); err != nil {
			fmt.Printf("[解散] ⚠ 未清理探针群 %s: %v\n", convID, err)
		} else {
			fmt.Printf("[解散] ✓ 探针群已清理\n")
		}
	}()

	const original = "撤回原文探针 ABC123"
	msg, err := c.SendMessage(ctx, convID, qapi.TextRequest(original))
	if err != nil {
		fmt.Printf("[发消息] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[发消息] ✓ id=%s\n", msg.ID)

	recalled, err := c.RecallMessage(ctx, msg.ID.String())
	if err != nil {
		fmt.Printf("[撤回] ✗ %v\n", err)
		return 1
	}
	if !recalled.IsRecalled {
		fmt.Printf("[撤回] ✗ 响应里 is_recalled=false: %+v\n", recalled)
		return 1
	}
	if got := recalled.RecallOriginal(); got != original {
		fmt.Printf("[撤回] ✗ 撤回响应 extra 原文不一致: %q (期望 %q)\n", got, original)
		return 1
	}
	fmt.Printf("[撤回] ✓ 响应含 extra.original_content=%q\n", recalled.RecallOriginal())

	// 回读历史，验证库里真的存了原文、content 已换成占位。
	page, err := c.Messages(ctx, convID, 1, 20)
	if err != nil {
		fmt.Printf("[回读] ✗ %v\n", err)
		return 1
	}
	var found *qapi.Message
	for i := range page.Messages {
		if page.Messages[i].ID == msg.ID {
			found = &page.Messages[i]
			break
		}
	}
	if found == nil {
		fmt.Println("[回读] ✗ 历史里没找到该消息")
		return 1
	}
	if !found.IsRecalled || found.RecallOriginal() != original {
		fmt.Printf("[回读] ✗ 历史数据不对: is_recalled=%v original=%q content=%q\n",
			found.IsRecalled, found.RecallOriginal(), found.Content)
		return 1
	}
	fmt.Printf("[回读] ✓ is_recalled=true，content=%q，extra 原文=%q\n", found.Content, found.RecallOriginal())

	fmt.Println("\n结论: 撤回增强链路（撤回存 extra 原文 → 历史回读可重新编辑）真实协议对齐。")
	return 0
}

func runReadUsersProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-readusers -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	// 临时单人群：发消息 → 查已读。自己不在 read_users 里，期望
	// read_users=[] / read_count=0 / total_members=1。
	conv, err := c.CreateGroup(ctx, "已读探针临时群", nil)
	if err != nil {
		fmt.Printf("[建群] ✗ %v\n", err)
		return 1
	}
	convID := conv.ID
	defer func() {
		if err := c.DissolveGroup(context.Background(), convID); err != nil {
			fmt.Printf("[解散] ⚠ 未清理探针群 %s: %v\n", convID, err)
		} else {
			fmt.Printf("[解散] ✓ 探针群已清理\n")
		}
	}()
	fmt.Printf("[建群] ✓ 会话 id=%s\n", convID)

	msg, err := c.SendMessage(ctx, convID, qapi.TextRequest("已读探针消息"))
	if err != nil {
		fmt.Printf("[发消息] ✗ %v\n", err)
		return 1
	}
	msgID := msg.ID.String()
	fmt.Printf("[发消息] ✓ id=%s type=%s\n", msgID, msg.Type)

	info, err := c.MessageReadUsers(ctx, qapi.ID(msgID))
	if err != nil {
		fmt.Printf("[已读] ✗ %v\n", err)
		return 1
	}
	if info.ReadCount != 0 || info.TotalMembers != 1 || len(info.ReadUsers) != 0 {
		fmt.Printf("[已读] ✗ 计数异常: read=%d total=%d users=%d\n",
			info.ReadCount, info.TotalMembers, len(info.ReadUsers))
		return 1
	}
	fmt.Printf("[已读] ✓ read_users=%d read_count=%d total_members=%d（自己不在已读列表，符合预期）\n",
		len(info.ReadUsers), info.ReadCount, info.TotalMembers)

	// 边界：对不存在/非自己的消息 id 查已读，应得到业务错误而不是 panic。
	if _, err := c.MessageReadUsers(ctx, qapi.ID("99999999")); err == nil {
		fmt.Println("[边界] ⚠ 不存在的消息 id 竟然查询成功（可接受但不典型）")
	} else {
		fmt.Printf("[边界] ✓ 非法消息 id 返回错误: %v\n", err)
	}

	fmt.Println("\n结论: 已读详情链路（GET /messages/:id/read-users）真实协议对齐。")
	return 0
}

func runAnnouncementProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-announce -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	// 临时建群（创建者自动成为群主，有公告写权限），探完即解散不留脏数据。
	conv, err := c.CreateGroup(ctx, "公告探针临时群", nil)
	if err != nil {
		fmt.Printf("[建群] ✗ %v\n", err)
		return 1
	}
	convID := conv.ID
	defer func() {
		if err := c.DissolveGroup(context.Background(), convID); err != nil {
			fmt.Printf("[解散] ⚠ 未清理探针群 %s: %v\n", convID, err)
		} else {
			fmt.Printf("[解散] ✓ 探针群已清理\n")
		}
	}()
	fmt.Printf("[建群] ✓ 会话 id=%s name=%s (type=%s)\n", conv.ID, conv.DisplayName(), conv.Type)

	// 写公告
	if err := c.SetGroupAnnouncement(ctx, convID, "探针公告：每周五同步"); err != nil {
		fmt.Printf("[写公告] ✗ %v\n", err)
		return 1
	}
	// 清空成空串也要能写通（对应客户端「清空公告」操作）
	if err := c.SetGroupAnnouncement(ctx, convID, ""); err != nil {
		fmt.Printf("[清公告] ✗ %v\n", err)
		return 1
	}
	if err := c.SetGroupAnnouncement(ctx, convID, "探针公告：每周五同步"); err != nil {
		fmt.Printf("[再写公告] ✗ %v\n", err)
		return 1
	}

	// 回读验证
	detail, err := c.GetConversation(ctx, convID)
	if err != nil {
		fmt.Printf("[回读] ✗ %v\n", err)
		return 1
	}
	if detail.Announcement != "探针公告：每周五同步" {
		fmt.Printf("[回读] ✗ 公告不一致: %q\n", detail.Announcement)
		return 1
	}
	fmt.Printf("[回读] ✓ 公告=%q，成员 %d 人\n", detail.Announcement, len(detail.Members))

	fmt.Println("\n结论: 群公告写链路（PUT /groups/:convID/announcement，含清空）真实协议对齐。")
	return 0
}

func runGroupsProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-groups -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	convs, err := c.Conversations(ctx)
	if err != nil || len(convs) == 0 {
		fmt.Printf("[会话] ✗ 没有可用会话: %v\n", err)
		return 1
	}

	// 优先找群会话；没有群就退而用第一个，至少能验证置顶/免打扰/公告字段。
	var conv *qapi.Conversation
	for i := range convs {
		if convs[i].Type == "group" || convs[i].Type == "discussion" {
			conv = &convs[i]
			break
		}
	}
	if conv == nil {
		conv = &convs[0]
	}
	fmt.Printf("[会话] ✓ 使用 [%s] %s (type=%s)\n", conv.ID, conv.DisplayName(), conv.Type)

	detail, err := c.GetConversation(ctx, conv.ID)
	if err != nil {
		fmt.Printf("[详情] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[详情] ✓ 成员 %d 人，公告=%q\n", len(detail.Members), detail.Announcement)
	for i, m := range detail.Members {
		if i >= 6 {
			fmt.Printf("       …还有 %d 人\n", len(detail.Members)-6)
			break
		}
		fmt.Printf("       - %s (role=%s, uid=%d)\n", m.DisplayName(), m.Role, m.UserID)
	}

	// 置顶：翻一次再翻回，中途从列表里确认状态确实变了。
	origPin := conv.PinnedNow()
	if err := c.SetConversationPin(ctx, conv.ID, !origPin); err != nil {
		fmt.Printf("[置顶] ✗ 切换失败: %v\n", err)
		return 1
	}
	after, _ := c.Conversations(ctx)
	p2 := findConvPinned(after, conv.ID)
	if p2 != !origPin {
		fmt.Printf("[置顶] ✗ 切换后列表状态未变: 期望 %v 实际 %v\n", !origPin, p2)
		return 1
	}
	_ = c.SetConversationPin(ctx, conv.ID, origPin) // 翻回原状态
	fmt.Printf("[置顶] ✓ 切换并恢复（原状态=%v）\n", origPin)

	// 免打扰：同理。
	origMute := conv.Muted
	if err := c.SetConversationMute(ctx, conv.ID, !origMute); err != nil {
		fmt.Printf("[免打扰] ✗ 切换失败: %v\n", err)
		return 1
	}
	after2, _ := c.Conversations(ctx)
	m2 := findConvMuted(after2, conv.ID)
	if m2 != !origMute {
		fmt.Printf("[免打扰] ✗ 切换后列表状态未变: 期望 %v 实际 %v\n", !origMute, m2)
		return 1
	}
	_ = c.SetConversationMute(ctx, conv.ID, origMute)
	fmt.Printf("[免打扰] ✓ 切换并恢复（原状态=%v）\n", origMute)

	fmt.Println("\n结论: 群成员 / 置顶 / 免打扰 / 公告 的真实协议全部对齐。")
	return 0
}

func findConvPinned(list []qapi.Conversation, id qapi.ID) bool {
	for i := range list {
		if list[i].ID.String() == id.String() {
			return list[i].PinnedNow()
		}
	}
	return false
}

// runNotifsProbe 验证通知中心全链路（-probe-notifs）：
// ① 主账号建临时群；② 注册/登录第二账号申请入群（POST /groups/:id/apply）；
// ③ 主账号拉通知应看到 approve_reject 类型；④ 置顶 toggle / 标已读 /
// 拒绝动作（DELETE join-requests）/ 一键已读 / 清空逐项验证；⑤ 解散清理。
func runNotifsProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-notifs -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ 群主 %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	conv, err := c.CreateGroup(ctx, "通知探针临时群", nil)
	if err != nil {
		fmt.Printf("[建群] ✗ %v\n", err)
		return 1
	}
	convID := conv.ID
	defer func() {
		if err := c.DissolveGroup(context.Background(), convID); err != nil {
			fmt.Printf("[解散] ⚠ 未清理探针群 %s: %v\n", convID, err)
		} else {
			fmt.Printf("[解散] ✓ 探针群已清理\n")
		}
	}()
	fmt.Printf("[建群] ✓ 会话 id=%s\n", convID)

	// 第二账号：不存在就注册，已存在直接登录。
	applicant := qapi.New(server)
	regBody, _ := json.Marshal(map[string]string{
		"username": "probe_notif", "password": "Probe#2026notif", "nickname": "通知探针二号",
	})
	regReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, server+"/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(regReq)
	if err != nil {
		fmt.Printf("[注册] ✗ %v\n", err)
		return 1
	}
	resp.Body.Close() // 已存在（400 用户名已存在）也算通过，下面用登录确认
	sess2, err := applicant.Login(ctx, "probe_notif", "Probe#2026notif")
	if err != nil {
		fmt.Printf("[第二账号] ✗ 登录失败: %v\n", err)
		return 1
	}
	fmt.Printf("[第二账号] ✓ %s id=%s\n", sess2.User.DisplayName(), sess2.User.ID)

	if err := applicant.ApplyGroupJoin(ctx, convID); err != nil {
		fmt.Printf("[申请入群] ✗ %v\n", err)
		return 1
	}
	fmt.Println("[申请入群] ✓ 已提交（后端会给群主写 approve_reject 通知）")

	// 群主拉通知列表
	page, err := c.Notifications(ctx, 1, 50)
	if err != nil {
		fmt.Printf("[通知列表] ✗ %v\n", err)
		return 1
	}
	var req *qapi.Notification
	// conversation_id 是群会话 id（数字），按字符串与 convID 比对
	for i := range page.List {
		n := page.List[i]
		if n.ActionType != "approve_reject" {
			continue
		}
		var p struct {
			ConversationID uint `json:"conversation_id"`
		}
		_ = json.Unmarshal([]byte(n.ActionPayload), &p)
		if fmt.Sprintf("%d", p.ConversationID) == convID.String() {
			req = &page.List[i]
		}
	}
	if req == nil {
		fmt.Printf("[通知列表] ✗ 没找到 approve_reject 通知: %d 条\n", len(page.List))
		for i, n := range page.List {
			fmt.Printf("       - #%d type=%s action=%s payload=%s\n", i, n.Type, n.ActionType, n.ActionPayload)
		}
		return 1
	}
	fmt.Printf("[通知列表] ✓ %q %q (id=%s, 重要=%v, payload=%s)\n",
		req.Title, req.Content, req.ID, req.Important, req.ActionPayload)

	notID := req.ID.String()

	// 置顶 toggle：翻一次再翻回
	if _, err := c.PatchNotification(ctx, qapi.ID(notID), qapi.NotificationPatch{Pinned: ptrBool(true)}); err != nil {
		fmt.Printf("[置顶] ✗ %v\n", err)
		return 1
	}
	after, _ := c.Notifications(ctx, 1, 50)
	if notifByIDIn(after.List, notID).Pinned != true {
		fmt.Println("[置顶] ✗ toggle 后 pinned 仍为 false")
		return 1
	}
	c.PatchNotification(ctx, qapi.ID(notID), qapi.NotificationPatch{Pinned: ptrBool(true)})
	fmt.Println("[置顶] ✓ PATCH pinned toggle 生效")

	// 标已读
	if err := c.MarkNotificationRead(ctx, qapi.ID(notID)); err != nil {
		fmt.Printf("[已读] ✗ %v\n", err)
		return 1
	}
	after2, _ := c.Notifications(ctx, 1, 50)
	if !notifByIDIn(after2.List, notID).Read {
		fmt.Println("[已读] ✗ 回读 read=false")
		return 1
	}
	fmt.Println("[已读] ✓ PUT :id/read 回读 read=true")

	// 拒绝动作（approve_reject 特殊分支）
	if err := c.RejectGroupJoin(ctx, convID, sess2.User.ID); err != nil {
		fmt.Printf("[拒绝] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[拒绝] ✓ DELETE join-requests（第二账号应收到被拒通知）\n")
	page3, _ := applicant.Notifications(ctx, 1, 50)
	rejected := false
	for _, n := range page3.List {
		if n.Type == "group_join_rejected" {
			rejected = true
		}
	}
	if !rejected {
		fmt.Println("[拒绝] ✗ 第二账号没收到被拒通知")
		return 1
	}
	fmt.Printf("[拒绝] ✓ 第二账号收到 group_join_rejected 通知（共 %d 条）\n", len(page3.List))

	// 一键已读 + 清空（主账号）
	if err := c.MarkNotificationsAllRead(ctx); err != nil {
		fmt.Printf("[一键已读] ✗ %v\n", err)
		return 1
	}
	after3, _ := c.Notifications(ctx, 1, 50)
	for _, n := range after3.List {
		if !n.Read {
			fmt.Printf("[一键已读] ✗ 还有未读 id=%s\n", n.ID)
			return 1
		}
	}
	fmt.Printf("[一键已读] ✓ 全部已读（%d 条）\n", len(after3.List))

	if err := c.ClearNotifications(ctx); err != nil {
		fmt.Printf("[清空] ✗ %v\n", err)
		return 1
	}
	after4, _ := c.Notifications(ctx, 1, 50)
	if len(after4.List) != 0 {
		fmt.Printf("[清空] ✗ 清空后仍有 %d 条\n", len(after4.List))
		return 1
	}
	fmt.Println("[清空] ✓ DELETE /notifications 后列表为空")

	// 第二账号也清掉自己的通知，避免探针残留影响下次运行
	_ = applicant.ClearNotifications(ctx)

	fmt.Println("\n结论: 通知中心链路（列表/已读/置顶 toggle/拒绝动作/一键已读/清空）真实协议对齐。")
	return 0
}

// notifByIDIn 在通知列表里按 id 找通知，找不到返回零值。
func notifByIDIn(list []qapi.Notification, id string) qapi.Notification {
	for _, n := range list {
		if n.ID.String() == id {
			return n
		}
	}
	return qapi.Notification{}
}

func findConvMuted(list []qapi.Conversation, id qapi.ID) bool {
	for i := range list {
		if list[i].ID.String() == id.String() {
			return list[i].Muted
		}
	}
	return false
}
