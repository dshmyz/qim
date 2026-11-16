package client

// 会话生命周期：登录 → 拉会话 → 拉历史 → 发消息 → 收推送 → 退出。
//
// 所有网络调用都是同步函数（方便测试直接调），UI 只负责把它们丢进 goroutine；
// 状态一律通过 a.mutate 回到主线程再改，避免 WS 回调和绘制树抢同一份数据。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

// historyPageSize 是第一页和每次「加载更早」共用的页大小。
// 后端上限 100；50 条足够覆盖一屏又不至于让首次加载变慢。
const historyPageSize = 50

// newApp 建立应用状态。token 不放在 app 上，由 qapi.Client 自己持有，
// 刷新后自动写回，不需要界面层参与。
func newApp() *app {
	ctx, cancel := context.WithCancel(context.Background())
	server := strings.TrimSpace(*flagServer)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	a := &app{
		active:      -1,
		client:      qapi.New(server),
		ctx:         ctx,
		cancel:      cancel,
		loginServer: server,
		loginUser:   strings.TrimSpace(*flagUser),
		loginPass:   pass,
		store:       secretStore{},
		hint:        newFileHintStore(),
		notifier:    desktopNotifier{},
		notifyOnNew: true,
		notifsTab:   "all",
	}
	// refresh 会轮换 refresh_token，凭据一变就同步回钥匙串，
	// 否则重启时拿的是旧 refresh token，可能已经失效。
	a.client.OnSessionChange(a.onSessionChange)
	return a
}

// mutate 把状态变更投递到 UI 线程并触发重绘。
// headless 测试里没有窗口，直接执行并加锁，避免网络回调和测试读取撞车。
func (a *app) mutate(fn func()) {
	if a.win == nil {
		a.mu.Lock()
		defer a.mu.Unlock()
		fn()
		return
	}
	a.win.Update(fn)
}

func (a *app) setNotice(msg string) {
	if msg == "" {
		return
	}
	log.Printf("[app] %s", msg)
	a.mutate(func() { a.notice = msg })
}

// submitLogin 由登录按钮 / Enter / 命令行自动登录调用。
// 网络请求放到后台，登录中禁用按钮，错误直接显示在原生登录页上。
func (a *app) submitLogin() {
	if a.loginBusy || a.client == nil {
		return
	}
	server := strings.TrimSpace(a.loginServer)
	user := strings.TrimSpace(a.loginUser)
	pass := a.loginPass
	code := strings.TrimSpace(a.loginCode)
	if server == "" || user == "" {
		a.mutate(func() { a.loginErr = "请填写服务器地址和账号" })
		return
	}
	if pass == "" && !a.need2FA {
		a.mutate(func() { a.loginErr = "请填写密码" })
		return
	}
	if a.need2FA && code == "" {
		a.mutate(func() { a.loginErr = "请输入二次验证的验证码" })
		return
	}
	a.mutate(func() {
		a.loginBusy = true
		a.loginErr = ""
	})

	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()
		ok, err := a.login(ctx, server, user, pass, code)
		if err != nil {
			log.Printf("[app] 登录失败: %v", err)
			a.mutate(func() {
				a.loginBusy = false
				a.loginErr = friendlyError(err)
			})
			return
		}
		if !ok {
			// 已经在 login 里切到验证码步骤，等用户再点一次
			return
		}
		a.afterLogin()
	}()
}

// login 跑一次登录（或二次验证）。返回值 ok 表示已经进入已登录态；
// false 且 err==nil 表示需要用户输入验证码，不算失败。
func (a *app) login(ctx context.Context, server, user, pass, code string) (bool, error) {
	a.client.SetBaseURL(server)

	var sess *qapi.Session
	var err error
	if a.need2FA {
		sess, err = a.client.Verify2FA(ctx, a.twoFASession, code)
	} else {
		sess, err = a.client.Login(ctx, user, pass)
	}
	if err != nil {
		var ae *qapi.APIError
		if errors.As(err, &ae) && ae.IsTwoFactorRequired() {
			a.mutate(func() {
				a.need2FA = true
				a.twoFASession = ae.TwoFactorSession()
				a.loginBusy = false
				a.loginErr = "该账号开启了二次验证，请输入验证码"
			})
			return false, nil
		}
		return false, err
	}

	log.Printf("[app] 登录成功 用户=%q id=%s token=%d 字节",
		sess.User.DisplayName(), sess.User.ID, len(sess.Token))
	a.mutate(func() {
		a.loggedIn = true
		a.userName = sess.User.DisplayName()
		a.selfID = sess.User.ID.String()
		a.loginBusy = false
		a.loginErr = ""
		a.need2FA = false
		a.twoFASession = ""
		a.loginPass = "" // 登录完不留密码
		a.loginCode = ""
		a.convs = nil
		a.msgs = nil
		a.active = -1
		a.input = ""
		a.notice = ""
		a.connStatus = qapi.StatusConnecting
		a.resetNotifs()
		a.resetChannels()
		a.resetProfile()
		a.resetFiles()
		a.clearPendingFiles()
		a.atSeen = map[string]bool{}
		a.tab = ""
		a.orgTabExpanded = map[string]bool{}
		a.membersCollapsed = false
		a.annEditorOpen = false
		a.annDraft = ""
		a.nameEditorOpen = false
		a.nameDraft = ""
		a.dissolveConfirmOpen = false
		a.memberSearchOpen = false
		a.memberSearch = ""
	})
	a.rememberSession(server, sess.User.ID.String(), sess.User.DisplayName())
	return true, nil
}

// restoreSession 用钥匙串里的凭据尝试免登录。
// 先发一次轻量请求校验：token 过期时 qapi 会自动 refresh 并重放；
// 只有明确判定为认证失败才清凭据，断网时保留，免得网络一抖就要重登。
func (a *app) restoreSession() {
	if a.store == nil {
		return
	}
	s, err := a.store.Load()
	if err != nil || !s.valid() {
		return
	}
	// 先填快照再写凭据：SetSession 的回调会顺手持久化，
	// 这时服务器地址和用户资料必须已经就位。
	a.savedMu.Lock()
	a.saved = s
	a.savedMu.Unlock()
	a.client.SetBaseURL(s.ServerURL)
	a.client.SetSession(s.Token, s.RefreshToken)

	go func(s savedSession) {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		if _, err := a.client.Conversations(ctx); err != nil {
			var ae *qapi.APIError
			if errors.As(err, &ae) && (ae.Code == 401 || ae.Code == 1002) {
				log.Printf("[app] 已保存的登录态失效，需要重新登录: %v", err)
				a.forgetSession()
				a.client.SetSession("", "")
				return
			}
			log.Printf("[app] 恢复登录态时网络不可用，保留凭据: %v", err)
			return
		}
		a.mutate(func() {
			a.loggedIn = true
			a.userName = s.UserName
			a.selfID = s.UserID
			a.loginServer = s.ServerURL
			a.connStatus = qapi.StatusConnecting
		})
		log.Printf("[app] 已恢复登录态 用户=%q", s.UserName)
		a.afterLogin()
	}(s)
}

// friendlyError 把网络错误翻译成用户能看懂的一句话。
func friendlyError(err error) string {
	s := err.Error()
	// 服务端已经给了中文原因（「服务端错误 1001: 群主退出前请先转让群主或解散群聊」）：
	// 直接取冒号后的文案，别让下面的错误码映射把它改写成别的话。
	if strings.Contains(s, "服务端错误") {
		if i := strings.Index(s, ": "); i >= 0 {
			if msg := strings.TrimSpace(s[i+2:]); msg != "" {
				return msg
			}
		}
	}
	switch {
	case strings.Contains(s, "connection refused"), strings.Contains(s, "no such host"):
		return "连不上服务器，请检查服务器地址"
	case strings.Contains(s, "context deadline exceeded"):
		return "服务器响应超时"
	case strings.Contains(s, "1001"):
		return "用户名或密码错误"
	default:
		return s
	}
}

// afterLogin 开长连接、拉第一屏会话列表。
func (a *app) afterLogin() {
	ctx := a.ctx
	a.startRealtime()
	go func() {
		if err := a.refreshConversations(ctx); err != nil {
			return
		}
		if *autoConv {
			a.mutate(func() {
				if len(a.convs) > 0 {
					a.selectConversation(0)
				}
			})
		}
	}()
}

func (a *app) startRealtime() {
	if a.rt != nil {
		a.rt.Stop()
	}
	// ctx / selfID 在登录完成后就固定下来，WS 回调只捕获这两个值，
	// 不去读 app 上会被 logout 覆盖的字段。
	ctx, selfID := a.ctx, a.selfID
	rt := qapi.NewRealtime(a.client)
	a.rt = rt
	rt.OnStatus(func(s qapi.Status) {
		a.mutate(func() { a.connStatus = s })
	})
	a.registerRealtimeHandlers(rt, ctx, selfID)
	rt.Start(ctx)
}

// refreshConversations 重新拉会话列表。
// 选中项按服务端 id 还原，避免每次同步都把高亮清零。
func (a *app) refreshConversations(ctx context.Context) error {
	if a.client == nil {
		return nil
	}
	list, err := a.client.Conversations(ctx)
	if err != nil {
		a.setNotice("加载会话失败：" + friendlyError(err))
		return err
	}
	convs := convsFromAPI(list)
	a.mutate(func() {
		prevID := ""
		if a.active >= 0 && a.active < len(a.convs) {
			prevID = a.convs[a.active].ID
		}
		a.convs = convs
		a.active = -1
		if prevID != "" {
			a.active = a.findConv(prevID)
		}
	})
	log.Printf("[app] 会话列表 %d 条", len(convs))
	return nil
}

// selectConversation 由原生侧栏点击调用：存旧草稿 → 清未读 → 拉历史 → 标记已读。
func (a *app) selectConversation(i int) {
	if i < 0 || i >= len(a.convs) {
		return
	}
	// 草稿按会话 id 存，切走时保存、切回来时恢复。
	if a.active >= 0 && a.active < len(a.convs) {
		a.saveDraft(a.convs[a.active].ID, a.input)
	}
	// 未读数马上会被清掉，必须先快照：历史加载完要用它插「以下为新消息」。
	pendingUnread := a.convs[i].Unread
	a.convs[i].Unread = 0
	a.active = i
	a.msgs = nil
	a.clearMessageOps()
	a.pendingUnread = pendingUnread
	a.input = a.draftFor(a.convs[i].ID)
	a.restoreMentionPicks(a.convs[i].ID)
	a.highlight = ""
	a.notice = ""
	// 头部「更多」菜单相关的弹层不能跨会话残留。
	a.annEditorOpen = false
	a.annDraft = ""
	a.nameEditorOpen = false
	a.nameDraft = ""
	a.dissolveConfirmOpen = false
	a.convConfirmKind = ""
	a.convConfirmID = ""
	a.convConfirmName = ""
	a.hasOlder = false
	a.oldestID = ""
	a.loadingOlder = false
	convID := a.convs[i].ID
	if a.client == nil || !a.loggedIn {
		return
	}
	selfID := a.selfID
	ctx := a.ctx
	a.loadingMsgs = true
	go a.loadMessages(ctx, convID, selfID, pendingUnread)
	go func() {
		if err := a.client.MarkRead(ctx, qapi.ID(convID)); err != nil {
			// 已读失败不影响看消息，只记日志，不弹错误条打扰用户
			log.Printf("[app] 标记已读失败 %s: %v", convID, err)
		}
	}()
	// 群聊详情带 members 与 announcement，进会话就拉一次，给 @ 候选和公告条用。
	go a.refreshConversationDetail(convID)
}

// refreshConversationDetail 拉会话详情（成员/公告/groupID）回写当前会话。
// 供进会话时预热，以及公告保存失败后的回滚（以服务端为准，无条件覆盖公告，
// 公告被清空成 "" 也要反映出来）。
func (a *app) refreshConversationDetail(convID string) {
	ctx := a.ctx
	cv, err := a.client.GetConversation(ctx, qapi.ID(convID))
	if err != nil {
		log.Printf("[app] 加载会话详情失败 %s: %v", convID, err)
		return
	}
	a.mutate(func() {
		if a.active < 0 || a.active >= len(a.convs) || a.convs[a.active].ID != convID {
			return
		}
		c := &a.convs[a.active]
		c.Members = membersFromAPI(cv.Members)
		c.Announcement = cv.Announcement
		if cv.GroupID != 0 {
			c.GroupID = cv.GroupID
		}
		// 顺带批量拉成员在线状态（失败静默，资料卡/成员行展示用）
		if len(c.Members) > 0 {
			ids := make([]string, 0, len(c.Members))
			for _, m := range c.Members {
				ids = append(ids, fmt.Sprintf("%d", m.UserID))
			}
			a.fetchMemberStatuses(ids)
		}
	})
}

// togglePin 翻转会话置顶状态：先乐观更新本地，再请求服务端，失败回滚。
func (a *app) togglePin(i int) {
	if i < 0 || i >= len(a.convs) || a.client == nil || !a.loggedIn {
		return
	}
	var convID string
	var want bool
	a.mutate(func() {
		c := &a.convs[i]
		want = !c.Pinned
		c.Pinned = want
		convID = c.ID
	})
	go func() {
		if err := a.client.SetConversationPin(a.ctx, qapi.ID(convID), want); err != nil {
			log.Printf("[app] 置顶切换失败 %s: %v", convID, err)
			a.mutate(func() {
				if i < len(a.convs) && a.convs[i].ID == convID {
					a.convs[i].Pinned = !want
				}
			})
		}
	}()
}

// toggleMute 翻转会话免打扰状态，逻辑同 togglePin。
func (a *app) toggleMute(i int) {
	if i < 0 || i >= len(a.convs) || a.client == nil || !a.loggedIn {
		return
	}
	var convID string
	var want bool
	a.mutate(func() {
		c := &a.convs[i]
		want = !c.Muted
		c.Muted = want
		convID = c.ID
	})
	go func() {
		if err := a.client.SetConversationMute(a.ctx, qapi.ID(convID), want); err != nil {
			log.Printf("[app] 免打扰切换失败 %s: %v", convID, err)
			a.mutate(func() {
				if i < len(a.convs) && a.convs[i].ID == convID {
					a.convs[i].Muted = !want
				}
			})
		}
	}()
}

// askConvAction 打开会话的二次确认弹层。kind 只有 exit（退出群聊）与
// remove（移除会话）两种，都是不可撤销的破坏性动作。
func (a *app) askConvAction(kind, convID, name string) {
	if convID == "" {
		return
	}
	a.mutate(func() {
		a.convConfirmKind = kind
		a.convConfirmID = convID
		a.convConfirmName = name
	})
}

// closeConvConfirm 关掉确认弹层（取消 / 动作完成后都走这里）。
func (a *app) closeConvConfirm() {
	a.mutate(func() {
		a.convConfirmKind = ""
		a.convConfirmID = ""
		a.convConfirmName = ""
	})
}

// confirmConvAction 执行确认过的退出群聊 / 移除会话。
// 本地先摘掉会话（失败再拉一次列表还原），随后后台打接口。
func (a *app) confirmConvAction() {
	kind, convID := a.convConfirmKind, a.convConfirmID
	a.closeConvConfirm()
	if convID == "" || a.client == nil || !a.loggedIn {
		return
	}
	a.dropConvLocal(convID)
	ctx := a.ctx
	go func() {
		var err error
		switch kind {
		case "exit":
			err = a.client.ExitGroup(ctx, qapi.ID(convID))
		default:
			err = a.client.DeleteConversation(ctx, qapi.ID(convID))
		}
		if err != nil {
			log.Printf("[app] 会话操作失败 %s(%s): %v", kind, convID, err)
			a.setNotice(friendlyError(err))
			_ = a.refreshConversations(ctx) // 失败时以服务端列表为准
		}
	}()
}

// dropConvLocal 从侧栏摘掉一个会话，并修正 active 下标：
// 摘掉的是当前会话就回到占位页，摘掉的是它前面的会话则下标左移。
func (a *app) dropConvLocal(convID string) {
	a.mutate(func() {
		idx := a.findConv(convID)
		if idx < 0 {
			return
		}
		a.convs = append(a.convs[:idx], a.convs[idx+1:]...)
		switch {
		case a.active == idx:
			a.active = -1
			a.msgs = nil
			a.clearMessageOps()
		case a.active > idx:
			a.active--
		}
		delete(a.drafts, convID)
		delete(a.mentionDrafts, convID)
	})
}

// openAnnouncementEditor 打开群公告编辑弹层，草稿预填当前公告。
func (a *app) openAnnouncementEditor() {
	a.mutate(func() {
		if a.active >= 0 && a.active < len(a.convs) {
			a.annDraft = a.convs[a.active].Announcement
		}
		a.annEditorOpen = true
	})
}

// closeAnnouncementEditor 关闭编辑弹层并清空草稿。
func (a *app) closeAnnouncementEditor() {
	a.mutate(func() {
		a.annEditorOpen = false
		a.annDraft = ""
	})
}

// openGroupNameEditor / saveGroupName：头部「更多 → 修改群名称」。
// 对齐 qim-client ChatHeaderActions 的 openEditGroupName，走
// PUT /api/v1/groups/:convID {"name": ...}，失败把本地名字还原。
func (a *app) openGroupNameEditor() {
	a.mutate(func() {
		if a.active >= 0 && a.active < len(a.convs) {
			a.nameDraft = a.convs[a.active].Name
		}
		a.nameEditorOpen = true
	})
}

func (a *app) closeGroupNameEditor() {
	a.mutate(func() {
		a.nameEditorOpen = false
		a.nameDraft = ""
	})
}

func (a *app) saveGroupName() {
	if a.client == nil || !a.loggedIn || a.active < 0 || a.active >= len(a.convs) {
		return
	}
	var convID, name, old string
	a.mutate(func() {
		name = strings.TrimSpace(a.nameDraft)
		convID = a.convs[a.active].ID
		old = a.convs[a.active].Name
		a.nameEditorOpen = false
		a.nameDraft = ""
		if name != "" {
			a.convs[a.active].Name = name
		}
	})
	if name == "" {
		return
	}
	go func() {
		if err := a.client.SetGroupName(a.ctx, qapi.ID(convID), name); err != nil {
			log.Printf("[app] 群名称保存失败 %s: %v", convID, err)
			a.setNotice("群名称修改失败：" + friendlyError(err))
			a.mutate(func() {
				if a.active >= 0 && a.active < len(a.convs) && a.convs[a.active].ID == convID {
					a.convs[a.active].Name = old
				}
			})
			return
		}
		a.setNotice("群名称已更新")
	}()
}

// openDissolveConfirm / dissolveGroup：头部「更多 → 解散群聊」。
// 原版有 ConfirmDialog 二次确认，这里同样先弹确认框再调 DELETE。
func (a *app) openDissolveConfirm() {
	a.mutate(func() { a.dissolveConfirmOpen = true })
}

func (a *app) closeDissolveConfirm() {
	a.mutate(func() { a.dissolveConfirmOpen = false })
}

func (a *app) dissolveGroup() {
	if a.client == nil || !a.loggedIn || a.active < 0 || a.active >= len(a.convs) {
		return
	}
	convID := a.convs[a.active].ID
	a.mutate(func() { a.dissolveConfirmOpen = false })
	go func() {
		if err := a.client.DissolveGroup(a.ctx, qapi.ID(convID)); err != nil {
			a.setNotice("解散群聊失败：" + friendlyError(err))
			return
		}
		a.mutate(func() {
			a.msgs = nil
			a.active = -1
		})
		_ = a.refreshConversations(a.ctx)
	}()
}

// saveAnnouncement 保存群公告：乐观更新本地，PUT /groups/:convID/announcement，
// 失败回滚并提示（后端仅允许群主/管理员，权限拒绝会走到这里）。
func (a *app) saveAnnouncement() {
	if a.client == nil || !a.loggedIn || a.active < 0 || a.active >= len(a.convs) {
		return
	}
	var convID, text string
	a.mutate(func() {
		text = strings.TrimSpace(a.annDraft)
		convID = a.convs[a.active].ID
		a.convs[a.active].Announcement = text
		a.annEditorOpen = false
		a.annDraft = ""
	})
	go func() {
		if err := a.client.SetGroupAnnouncement(a.ctx, qapi.ID(convID), text); err != nil {
			log.Printf("[app] 群公告保存失败 %s: %v", convID, err)
			a.setNotice("公告保存失败：" + friendlyError(err))
			// 回滚：以服务端为准重拉详情（无条件覆盖公告），不能在 mutate 里做网络调用。
			a.refreshConversationDetail(convID)
		} else {
			a.setNotice("公告已更新")
		}
	}()
}

// openReadUsers 打开已读详情弹层并异步拉取列表。
// 仅对自己的已送达消息有意义（对齐 qim-client：message.isSelf 才能看已读）。
func (a *app) openReadUsers(m message) {
	if a.client == nil || !a.loggedIn || m.ServerID == "" {
		return
	}
	msgID := m.ServerID
	a.mutate(func() {
		a.readUsersOpen = true
		a.readUsersBusy = true
		a.readUsersID = msgID
		a.readUsersInfo = nil
	})
	go func() {
		info, err := a.client.MessageReadUsers(a.ctx, qapi.ID(msgID))
		a.mutate(func() {
			// 弹层已被关掉或换了消息，丢弃结果
			if !a.readUsersOpen || a.readUsersID != msgID {
				return
			}
			a.readUsersBusy = false
			if err != nil {
				log.Printf("[app] 拉取已读列表失败 %s: %v", msgID, err)
				a.readUsersInfo = nil
				a.notice = "已读列表获取失败：" + friendlyError(err)
				return
			}
			a.readUsersInfo = info
		})
	}()
}

// closeReadUsers 关闭已读详情弹层并清理状态。
func (a *app) closeReadUsers() {
	a.mutate(func() {
		a.readUsersOpen = false
		a.readUsersBusy = false
		a.readUsersID = ""
		a.readUsersInfo = nil
	})
}

// notifsPageSize 是通知中心每页条数，对齐 qim-client（50）。
const notifsPageSize = 50

// openNotifs 打开通知中心并拉第一页。面板开着时不重复拉（点铃铛是幂等的）。
func (a *app) openNotifs() {
	a.mutate(func() {
		a.notifsOpen = true
		if a.notifsPage == 0 { // 首次打开才置 busy，翻页保留旧列表
			a.notifsBusy = true
		}
	})
	go a.loadNotifsPage(1)
}

// closeNotifs 关闭通知中心面板；列表保留，下次打开直接展示再刷新。
func (a *app) closeNotifs() {
	a.mutate(func() {
		a.notifsOpen = false
		a.notifsBusy = false // 加载中途关闭：别把 busy 残留给下次打开
	})
}

// loadNotifsPage 拉一页通知。page==1 整表替换，否则按 id 去重后追加。
// 未读徽标从列表实时算，保证与面板内容一致（qim-client 同样从列表 filter）。
func (a *app) loadNotifsPage(page int) {
	if a.client == nil || !a.loggedIn {
		a.mutate(func() { a.notifsBusy = false })
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	p, err := a.client.Notifications(ctx, page, notifsPageSize)
	a.mutate(func() {
		if !a.notifsOpen {
			return // 面板已关，丢弃这次结果
		}
		a.notifsBusy = false
		if err != nil {
			log.Printf("[app] 加载通知失败: %v", err)
			a.notice = "加载通知失败：" + friendlyError(err)
			return
		}
		if page <= 1 {
			a.notifs = p.List
		} else {
			seen := make(map[string]bool, len(a.notifs))
			for _, n := range a.notifs {
				seen[n.ID.String()] = true
			}
			for _, n := range p.List {
				if !seen[n.ID.String()] {
					a.notifs = append(a.notifs, n)
				}
			}
		}
		a.notifsPage = page
		a.notifsHasMore = len(p.List) >= notifsPageSize
		a.unreadNotifs = countUnreadNotifs(a.notifs)
	})
}

// countUnreadNotifs 统计列表里的未读数。
func countUnreadNotifs(list []qapi.Notification) int {
	n := 0
	for _, it := range list {
		if !it.Read {
			n++
		}
	}
	return n
}

// setNotifsTab 切换通知过滤 tab（all / unread / important）。
func (a *app) setNotifsTab(tab string) {
	a.mutate(func() { a.notifsTab = tab })
}

// notifByID 在列表里找一条通知的下标。
func (a *app) notifByID(id string) int {
	for i := range a.notifs {
		if a.notifs[i].ID.String() == id {
			return i
		}
	}
	return -1
}

// clickNotif 点击通知条目：未读就标已读（乐观更新，失败回滚）。
func (a *app) clickNotif(id string) {
	i := a.notifByID(id)
	if i < 0 || a.client == nil || !a.loggedIn || a.notifs[i].Read {
		return
	}
	a.mutate(func() { a.notifs[i].Read = true; a.unreadNotifs = countUnreadNotifs(a.notifs) })
	go func() {
		if err := a.client.MarkNotificationRead(a.ctx, qapi.ID(id)); err != nil {
			log.Printf("[app] 通知标已读失败 %s: %v", id, err)
			a.mutate(func() {
				if j := a.notifByID(id); j >= 0 {
					a.notifs[j].Read = false
					a.unreadNotifs = countUnreadNotifs(a.notifs)
				}
			})
		}
	}()
}

// markAllNotifsRead 一键已读：先快照旧未读集合，PUT read-all 成功则全部置已读，
// 失败按快照回滚。
func (a *app) markAllNotifsRead() {
	if a.client == nil || !a.loggedIn {
		return
	}
	wasUnread := make(map[string]bool)
	a.mutate(func() {
		for _, n := range a.notifs {
			if !n.Read {
				wasUnread[n.ID.String()] = true
			}
		}
		for i := range a.notifs {
			a.notifs[i].Read = true
		}
		a.unreadNotifs = 0
	})
	go func() {
		if err := a.client.MarkNotificationsAllRead(a.ctx); err != nil {
			log.Printf("[app] 一键已读失败: %v", err)
			a.setNotice("一键已读失败：" + friendlyError(err))
			a.mutate(func() {
				for i := range a.notifs {
					if wasUnread[a.notifs[i].ID.String()] {
						a.notifs[i].Read = false
					}
				}
				a.unreadNotifs = countUnreadNotifs(a.notifs)
			})
		}
	}()
}

// clearNotifs 清空全部通知（乐观清列表，失败回滚整表）。
func (a *app) clearNotifs() {
	if a.client == nil || !a.loggedIn {
		return
	}
	var backup []qapi.Notification
	a.mutate(func() {
		backup = append(backup, a.notifs...)
		a.notifs = nil
		a.notifsPage = 0
		a.notifsHasMore = false
		a.unreadNotifs = 0
	})
	go func() {
		if err := a.client.ClearNotifications(a.ctx); err != nil {
			log.Printf("[app] 清空通知失败: %v", err)
			a.setNotice("清空通知失败：" + friendlyError(err))
			a.mutate(func() {
				a.notifs = backup
				a.unreadNotifs = countUnreadNotifs(a.notifs)
			})
		}
	}()
}

// deleteNotif 删除一条通知（乐观移除，失败插回原位置）。
func (a *app) deleteNotif(id string) {
	if a.client == nil || !a.loggedIn {
		return
	}
	var backup qapi.Notification
	var pos int
	a.mutate(func() {
		i := a.notifByID(id)
		if i < 0 {
			return
		}
		backup = a.notifs[i]
		pos = i
		a.notifs = append(a.notifs[:i], a.notifs[i+1:]...)
		a.unreadNotifs = countUnreadNotifs(a.notifs)
	})
	if backup.ID.Empty() {
		return
	}
	go func() {
		if err := a.client.DeleteNotification(a.ctx, qapi.ID(id)); err != nil {
			log.Printf("[app] 删除通知失败 %s: %v", id, err)
			a.mutate(func() {
				if a.notifByID(id) >= 0 {
					return // 已经重新加载过，别再插回去造成重复
				}
				if pos > len(a.notifs) {
					pos = len(a.notifs)
				}
				a.notifs = append(a.notifs[:pos], append([]qapi.Notification{backup}, a.notifs[pos:]...)...)
				a.unreadNotifs = countUnreadNotifs(a.notifs)
			})
		}
	}()
}

// toggleNotifPin 置顶/取消置顶一条通知。后端 PATCH pinned 是 toggle 语义，
// 直接本地翻转，失败再翻回来。
func (a *app) toggleNotifPin(id string) {
	a.toggleNotifFlag(id, func(p *qapi.NotificationPatch) { p.Pinned = ptrBool(true) })
}

// toggleNotifImportant 标记/取消重要，语义同 toggleNotifPin。
func (a *app) toggleNotifImportant(id string) {
	a.toggleNotifFlag(id, func(p *qapi.NotificationPatch) { p.Important = ptrBool(true) })
}

// toggleNotifFlag 是 pinned/important 共用的 toggle 骨架：乐观翻转 + 失败回滚。
func (a *app) toggleNotifFlag(id string, fill func(*qapi.NotificationPatch)) {
	if a.client == nil || !a.loggedIn {
		return
	}
	var patch qapi.NotificationPatch
	fill(&patch)
	field := "important"
	if patch.Pinned != nil {
		field = "pinned"
	}
	a.mutate(func() {
		if i := a.notifByID(id); i >= 0 {
			if field == "pinned" {
				a.notifs[i].Pinned = !a.notifs[i].Pinned
			} else {
				a.notifs[i].Important = !a.notifs[i].Important
			}
		}
	})
	go func() {
		if _, err := a.client.PatchNotification(a.ctx, qapi.ID(id), patch); err != nil {
			log.Printf("[app] 通知 %s 切换失败 %s: %v", field, id, err)
			a.mutate(func() {
				if i := a.notifByID(id); i >= 0 {
					if field == "pinned" {
						a.notifs[i].Pinned = !a.notifs[i].Pinned
					} else {
						a.notifs[i].Important = !a.notifs[i].Important
					}
				}
			})
		}
	}()
}

// handleNotifAction 处理通知条目上的动作按钮（接受/忽略/确认/延期/同意/拒绝…）。
// approve_reject 是特例：不走 PATCH action 分发，而是直接调群成员接口
// （对齐 qim-client handleAction 的特殊分支）。成功后条目标 handled=true+已读；
// 失败不置 handled，按钮保持可点（后端语义：用户可重试）。
func (a *app) handleNotifAction(id, action string) {
	if a.client == nil || !a.loggedIn {
		return
	}
	i := a.notifByID(id)
	if i < 0 {
		return
	}
	n := a.notifs[i]

	if n.ActionType == "approve_reject" && (action == "approve" || action == "reject") {
		// payload: {conversation_id, user_id}
		var payload struct {
			ConversationID qapi.ID `json:"conversation_id"`
			UserID         qapi.ID `json:"user_id"`
		}
		if err := json.Unmarshal([]byte(n.ActionPayload), &payload); err != nil || payload.ConversationID.Empty() {
			a.setNotice("无法处理该申请：通知数据不完整")
			return
		}
		go func() {
			var err error
			if action == "approve" {
				err = a.client.ApproveGroupJoin(a.ctx, payload.ConversationID, payload.UserID)
			} else {
				err = a.client.RejectGroupJoin(a.ctx, payload.ConversationID, payload.UserID)
			}
			if err != nil {
				log.Printf("[app] 处理入群申请失败 %s %s: %v", id, action, err)
				a.setNotice("处理失败：" + friendlyError(err))
				return
			}
			a.mutate(func() {
				if j := a.notifByID(id); j >= 0 {
					a.notifs[j].Handled = true
					a.notifs[j].Read = true
				}
				a.unreadNotifs = countUnreadNotifs(a.notifs)
			})
			// 成员变动会影响会话列表，刷新一下
			go a.refreshConversations(a.ctx)
		}()
		return
	}

	// 通用动作：PATCH {action} 分发到后端 handler。
	go func() {
		_, err := a.client.PatchNotification(a.ctx, qapi.ID(id), qapi.NotificationPatch{Action: action})
		if err != nil {
			log.Printf("[app] 通知动作失败 %s %s: %v", id, action, err)
			a.setNotice("操作失败：" + friendlyError(err))
			return // 失败不置 handled，按钮保持可点
		}
		a.mutate(func() {
			if j := a.notifByID(id); j >= 0 {
				a.notifs[j].Handled = true
				a.notifs[j].Read = true
			}
			a.unreadNotifs = countUnreadNotifs(a.notifs)
		})
	}()
}

// applyNewNotification 处理 WS 推来的 new_notification：按 id 去重后插到列表头，
// 未读数重算。面板没开时只累计徽标（列表打开时才会拉全量）。
func (a *app) applyNewNotification(n qapi.Notification) {
	a.mutate(func() {
		if a.notifByID(n.ID.String()) >= 0 {
			return
		}
		// 列表已加载过才插队；没加载过（首次打开前）等 openNotifs 拉全量
		if a.notifsPage > 0 || a.notifsOpen {
			a.notifs = append([]qapi.Notification{n}, a.notifs...)
		}
		a.unreadNotifs = countUnreadNotifs(a.notifs)
	})
}

func ptrBool(v bool) *bool { return &v }

// loadMessages 拉第一页历史消息；会话中途切走了就丢弃这次结果。
// 后端返回「最新 N 条、时间正序」，所以只要这一页满了就说明
// 可能还有更早的，翻页用 oldestID 作 before_id 游标。
func (a *app) loadMessages(ctx context.Context, convID, selfID string, unread int) error {
	page, err := a.client.Messages(ctx, qapi.ID(convID), 1, historyPageSize)
	if err != nil {
		a.mutate(func() { a.loadingMsgs = false })
		a.setNotice("加载消息失败：" + friendlyError(err))
		return err
	}
	msgs := insertUnreadDivider(msgsFromAPI(page.Messages, selfID), unread)
	a.mutate(func() {
		if a.active < 0 || a.active >= len(a.convs) || a.convs[a.active].ID != convID {
			return
		}
		a.msgs = msgs
		a.loadingMsgs = false
		a.oldestID = oldestServerID(msgs)
		a.hasOlder = len(page.Messages) >= historyPageSize
	})
	return nil
}

// loadOlder 由消息列表顶部的按钮触发，按 before_id 再取一页更早的消息。
// 用游标而不是 page+offset：会话里随时会插入新消息，offset 会漂移。
func (a *app) loadOlder() {
	if a.loadingOlder || !a.hasOlder || a.active < 0 || a.active >= len(a.convs) || a.client == nil {
		return
	}
	convID, beforeID, selfID := a.convs[a.active].ID, a.oldestID, a.selfID
	if beforeID == "" {
		a.mutate(func() { a.hasOlder = false })
		return
	}
	a.loadingOlder = true
	ctx := a.ctx
	go func() {
		page, err := a.client.MessagesBefore(ctx, qapi.ID(convID), beforeID, historyPageSize)
		if err != nil {
			a.mutate(func() { a.loadingOlder = false })
			a.setNotice("加载更早消息失败：" + friendlyError(err))
			return
		}
		older := msgsFromAPI(page.Messages, selfID)
		a.mutate(func() {
			defer func() { a.loadingOlder = false }()
			if a.active < 0 || a.active >= len(a.convs) || a.convs[a.active].ID != convID {
				return // 用户已经切到别的会话
			}
			a.msgs = prependOlder(a.msgs, older)
			if id := oldestServerID(older); id != "" {
				a.oldestID = id
			}
			// 返回不足一页 = 已经翻到头；空页也一样收尾。
			a.hasOlder = len(page.Messages) >= historyPageSize
		})
	}()
}

// sendText 是输入框 Enter / 发送按钮共用的入口：取走当前引用再发。
func (a *app) sendText(text string) {
	var q *messageQuote
	if a.quote != nil {
		cp := *a.quote
		q = &cp
		a.quote = nil
	}
	a.sendMessage(text, q)
}

// sendMessage 先乐观上屏，再走 REST；成功换上服务端 id，失败标红。
// 引用与 @ 提及都在这里落到请求里：界面显示的是可读文本，线上格式
// 各走一条转换，互不污染。
func (a *app) sendMessage(text string, quote *messageQuote) {
	displayText := strings.TrimSpace(text)
	if displayText == "" || a.active < 0 || a.active >= len(a.convs) {
		return
	}
	wireText := a.encodeMentions(displayText)
	convID := a.convs[a.active].ID
	a.nextID++
	key := fmt.Sprintf("local-%d", a.nextID)
	now := time.Now()
	msg := message{
		Key: key, Mine: true, Text: displayText,
		At: now.Format("15:04"), Stamp: now.UnixMilli(), Pending: true,
	}
	if quote != nil {
		msg.QuoteID = quote.ID
		msg.QuoteSender = quote.Name
		msg.QuoteText = quote.Text
	}
	a.mutate(func() {
		a.msgs = append(a.msgs, msg)
		a.input = ""
		delete(a.drafts, convID) // 草稿已发出，切走再回来不该重现
		delete(a.mentionDrafts, convID)
		a.mentionPicks = nil
		if a.active >= 0 && a.active < len(a.convs) {
			a.convs[a.active].Preview = displayText
			a.convs[a.active].Time = now.Format("15:04")
			a.convs[a.active].Stamp = now.UnixMilli()
		}
	})
	if a.client == nil || !a.loggedIn {
		// headless 测试：没有服务端时把这条当作本地已送达，别一直转圈
		a.mutate(func() { a.updateLocal(key, false, false) })
		return
	}
	ctx := a.ctx
	quotedID := ""
	if quote != nil {
		quotedID = quote.ID
	}
	go a.submitMessage(ctx, convID, key, wireText, quotedID)
}

// sendForward 转发到指定会话。req 非 nil 时按原类型发（image/file/
// merged_forward/share 真协议），nil 退化为文本转发。
// 不改当前活跃会话、也不乐观上屏——目标会话未必是当前窗口，且 qim-client
// 转发也是「发到目标会话」语义，回当前会话的反馈用一个轻量 toast 即可。
func (a *app) sendForward(targetConvID string, req *qapi.SendRequest, text string) {
	if a.client == nil || !a.loggedIn {
		return
	}
	if req == nil {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		req = &qapi.SendRequest{Type: "text", Content: text}
	}
	convID := qapi.ID(targetConvID)
	// 超时 ctx 必须在 goroutine 内创建：外层 WithTimeout+defer cancel 会在
	// 函数返回时立刻取消，而本函数发完就返回、goroutine 还没跑完请求，会
	// 拿到 "context canceled"。
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()
		if _, err := a.client.SendMessage(ctx, convID, *req); err != nil {
			log.Printf("[forward] 转发到 %s 失败: %v", targetConvID, err)
			a.setNotice("转发失败：" + friendlyError(err))
			return
		}
		a.setNotice("已转发")
	}()
}

func (a *app) submitMessage(parent context.Context, convID, key, text, quotedID string) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	m, err := a.client.SendMessage(ctx, qapi.ID(convID), qapi.TextRequestWithQuote(text, quotedID))
	if err != nil {
		a.mutate(func() {
			a.updateLocal(key, false, true)
			a.notice = "发送失败：" + friendlyError(err)
		})
		return
	}
	a.mutate(func() {
		a.updateLocal(key, false, false)
		if m != nil && m.ID.String() != "" {
			a.rekeyLocal(key, messageKey(m.ID.String(), 0), m.Timestamp)
		}
	})
}

// saveDraft / draftFor 按会话保存没发出去的输入。
// 空白不算草稿，免得每个点过的会话都留一条空记录。
func (a *app) saveDraft(convID, text string) {
	if convID == "" {
		return
	}
	if a.drafts == nil {
		a.drafts = make(map[string]string)
	}
	if strings.TrimSpace(text) == "" {
		delete(a.drafts, convID)
		delete(a.mentionDrafts, convID)
		return
	}
	a.drafts[convID] = text
	// @ 提及的选择也随草稿存：草稿里是 "@名字"，没有 picks 就发不出通知。
	if a.mentionDrafts == nil {
		a.mentionDrafts = make(map[string]map[string]string)
	}
	if len(a.mentionPicks) > 0 {
		cp := make(map[string]string, len(a.mentionPicks))
		for k, v := range a.mentionPicks {
			cp[k] = v
		}
		a.mentionDrafts[convID] = cp
	} else {
		delete(a.mentionDrafts, convID)
	}
}

func (a *app) draftFor(convID string) string {
	return a.drafts[convID]
}

// restoreMentionPicks 切回会话时恢复草稿里的 @ 选择（拷贝，避免共享 map）。
func (a *app) restoreMentionPicks(convID string) {
	a.mentionPicks = nil
	src := a.mentionDrafts[convID]
	if len(src) == 0 {
		return
	}
	cp := make(map[string]string, len(src))
	for k, v := range src {
		cp[k] = v
	}
	a.mentionPicks = cp
}

// retryFailed 重发一条失败的消息：摘掉旧气泡，再走一遍正常发送路径。
// 由气泡上的「重试」按钮在 UI 线程直接调用；摘除必须同步完成，
// 不能塞进 mutate（真窗口下 mutate 是投递到下一轮循环，异步的），
// 否则紧接着的 sendText 会把旧气泡留在屏幕上。
func (a *app) retryFailed(key string) {
	var found *message
	a.notice = ""
	for i := range a.msgs {
		if a.msgs[i].Key == key && a.msgs[i].Failed {
			cp := a.msgs[i]
			found = &cp
			a.msgs = append(a.msgs[:i], a.msgs[i+1:]...)
			break
		}
	}
	if found == nil {
		return
	}
	// 富媒体：保留本地预览和已上传的附件，只重走失败的那一步。
	if found.Kind != "" {
		a.resendAttachment(*found)
		return
	}
	var q *messageQuote
	if found.QuoteID != "" {
		q = &messageQuote{
			ID: found.QuoteID, Name: found.QuoteSender, Text: found.QuoteText,
		}
	}
	if found.Text != "" {
		// 直接走 sendMessage：重试不该再吃掉用户后来新设的引用。
		a.sendMessage(found.Text, q)
	}
}

// updateLocal 改一条乐观消息的状态。
func (a *app) updateLocal(key string, pending, failed bool) {
	for i := range a.msgs {
		if a.msgs[i].Key == key {
			a.msgs[i].Pending = pending
			a.msgs[i].Failed = failed
			return
		}
	}
}

// rekeyLocal 把本地临时 id 换成服务端 id；若 WS 已经先把这条推过来了，
// 就删掉本地那条，避免同一条消息显示两遍。
func (a *app) rekeyLocal(key, newKey string, stamp int64) {
	idx := -1
	for i := range a.msgs {
		if a.msgs[i].Key == key {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	if a.hasMessage(newKey) {
		a.msgs = append(a.msgs[:idx], a.msgs[idx+1:]...)
		return
	}
	a.msgs[idx].Key = newKey
	// messageKey 是 "m-"+服务端 id。乐观消息换 key 后必须同步补上
	// serverID，否则刚发出去的消息右键里的撤回/引用一直是禁用态。
	if strings.HasPrefix(newKey, "m-") {
		a.msgs[idx].ServerID = strings.TrimPrefix(newKey, "m-")
	}
	if stamp > 0 {
		a.msgs[idx].Stamp = stamp
		a.msgs[idx].At = stampLabel(stamp)
	}
}

// applyIncoming 处理 WS 推来的一条新消息：更新侧栏预览/未读，
// 是当前会话就追加到消息列表。流式消息（type=streaming）只是中间帧，
// 等它 finish 成正式消息再显示，否则会刷出一串半截气泡。
func (a *app) applyIncoming(m qapi.Message, selfID string) {
	if m.Type == "streaming" {
		return
	}
	if m.Timestamp == 0 {
		m.Timestamp = time.Now().UnixMilli()
	}
	convID := m.ConversationID.String()
	incoming := messageFromAPI(m, selfID, 0)
	var notifyTitle string
	a.mutate(func() {
		i := a.findConv(convID)
		if i < 0 {
			// 新会话：列表里没有，重拉一次（此处在 mutate 内，读 a.ctx 是安全的）
			go a.refreshConversations(a.ctx)
		} else {
			a.convs[i].Preview = m.Preview()
			a.convs[i].Time = convTimeLabel(m.Timestamp)
			a.convs[i].Stamp = m.Timestamp
			if i != a.active && !m.IsMine(selfID) {
				a.convs[i].Unread++
				// 收集标题，真正的弹窗放到 mutate 外做，避免阻塞状态变更或被 a.mu 嵌套。
				if a.notifyOnNew {
					notifyTitle = a.convs[i].Name
				}
			}
		}
		if i >= 0 && i == a.active && !a.hasMessage(incoming.Key) {
			a.msgs = append(a.msgs, incoming)
		}
	})
	if notifyTitle != "" && a.notifier != nil {
		_ = a.notifier.Notify(notifyTitle, m.Preview())
	}
}

// logout 断开长连接、通知服务端、清凭据、回到原生登录页。
func (a *app) logout() {
	// 服务端登出用 token 快照 + 独立 ctx：本地 ctx 马上会被 cancel、
	// 本地凭据马上会被清空，两者都不能依赖。尽力而为，失败不阻塞退出。
	if a.client != nil && a.client.HasToken() {
		token, _ := a.client.Session()
		go func(c *qapi.Client, tok string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := c.Logout(ctx, tok); err != nil {
				log.Printf("[app] 服务端登出失败（本地已退出）: %v", err)
			}
		}(a.client, token)
	}
	a.forgetSession()

	if a.cancel != nil {
		a.cancel()
	}
	if a.rt != nil {
		a.rt.Stop()
		a.rt = nil
	}
	if a.client != nil {
		a.client.SetSession("", "")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mutate(func() {
		a.ctx, a.cancel = ctx, cancel
		a.loggedIn = false
		a.userName = ""
		a.selfID = ""
		a.connStatus = qapi.StatusDisconnected
		a.convs = nil
		a.msgs = nil
		a.active = -1
		a.input = ""
		a.search = ""
		a.lastSearch = ""
		a.searchHits = qapi.SearchHits{}
		a.searchBusy = false
		a.searchErr = ""
		a.highlight = ""
		a.drafts = nil
		a.clearMessageOps()
		a.mentionDrafts = nil
		a.hasOlder = false
		a.oldestID = ""
		a.loadingOlder = false
		a.notice = ""
		a.loginErr = ""
		a.loginCode = ""
		a.loginBusy = false
		a.need2FA = false
		a.twoFASession = ""
		a.loadingMsgs = false
		a.resetNotifs()
		a.resetChannels()
		a.resetProfile()
		a.resetFiles()
		a.clearPendingFiles()
		a.atSeen = map[string]bool{}
		a.tab = ""
		a.orgTabExpanded = map[string]bool{}
		a.membersCollapsed = false
		a.annEditorOpen = false
		a.annDraft = ""
		a.nameEditorOpen = false
		a.nameDraft = ""
		a.dissolveConfirmOpen = false
		a.memberSearchOpen = false
		a.memberSearch = ""
	})
	log.Printf("[app] 已退出登录")
}

// resetNotifs 清空通知中心全部状态（登录/登出切换账号时用）。
func (a *app) resetNotifs() {
	a.notifsOpen = false
	a.notifsBusy = false
	a.notifs = nil
	a.notifsTab = "all"
	a.notifsPage = 0
	a.notifsHasMore = false
	a.unreadNotifs = 0
}
