package client

import (
	"context"
	"strconv"
	"strings"

	"qim-newclient/internal/qapi"
)

// aiTurn 是 AI 侧边栏里的一条气泡。Role 只有 "user" / "assistant"；
// Streaming 为真时气泡尾部画打字机光标，并在输入框显示「停止生成」。
type aiTurn struct {
	Role      string
	Text      string
	Streaming bool
}

// aiOpenSuggestions 是空态的 4 张推荐卡片，文案与 qim-client
// AISidebarPanel.vue 的 suggestions 一致（左侧是展示标题，右侧是实际发送的提示词）。
var aiOpenSuggestions = [][2]string{
	{"总结讨论", "帮我总结一下刚才的讨论要点"},
	{"设置提醒", "三点钟提醒我开会"},
	{"搜索信息", "帮我搜索一下相关的历史消息"},
	{"查看待办", "我有哪些待办任务？"},
}

func (a *app) aiToggle() {
	opening := !a.aiOpen
	a.mutate(func() { a.aiOpen = opening })
	// 首次打开时后台拉一次推荐提示；拉不到就静默，不弹错误。
	if opening && len(a.aiPrompts) == 0 {
		go a.aiLoadPrompts()
	}
}

func (a *app) aiClose() {
	a.aiStop()
	a.mutate(func() { a.aiOpen = false })
}

func (a *app) aiStop() {
	if a.aiCancel != nil {
		a.aiCancel()
		a.aiCancel = nil
	}
}

func (a *app) aiClear() {
	a.aiStop()
	a.mutate(func() {
		a.aiTurns = nil
		a.aiBusy = false
	})
}

func (a *app) aiToggleScope() {
	a.mutate(func() { a.aiScopeCross = !a.aiScopeCross })
}

// aiContextName 是上下文条的会话名；自由对话或没选会话时返回空。
func (a *app) aiContextName() string {
	if a.aiScopeCross {
		return ""
	}
	conv := a.currentConv()
	return conv.Name
}

// aiLoadPrompts 拉输入框上方的推荐提示。拉不到就保持空，界面不显示提示条——
// 管理员没配置时后端返回空数组，这是正常状态，不要报错打扰用户。
func (a *app) aiLoadPrompts() {
	// headless 测试或未登录时可能没有 client；静默跳过，别把后台 goroutine 打挂。
	if a.client == nil {
		return
	}
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	prompts, err := a.client.AISuggestedPrompts(parent)
	if err != nil || len(prompts) == 0 {
		return
	}
	a.mutate(func() { a.aiPrompts = prompts })
}

// aiSend 发起一次流式对话。text 是已经准备好的提示词（输入框或空态卡片）。
// 请求在 goroutine 里发，每帧用 mutate 投递回 UI 线程追加到助手气泡。
func (a *app) aiSend(text string) {
	text = strings.TrimSpace(text)
	if text == "" || a.aiBusy || a.client == nil {
		return
	}

	// scope=current 时带上当前会话 id，让 AI 能看到会话上下文。
	// 注意这里在 UI 线程直接读状态并快照：mutate 在真窗口下是异步投递，
	// 不能指望它返回后 a.active 已经更新。
	req := qapi.AISidebarRequest{Message: text, Scope: "current"}
	if a.aiScopeCross {
		req.Scope = "cross"
	} else if conv := a.currentConv(); conv.ID != "" {
		if n, err := strconv.ParseUint(conv.ID, 10, 64); err == nil {
			req.ConversationID = uint(n)
		}
	}

	a.mutate(func() {
		a.aiInput = ""
		a.aiTurns = append(a.aiTurns, aiTurn{Role: "user", Text: text})
		a.aiTurns = append(a.aiTurns, aiTurn{Role: "assistant", Streaming: true})
		a.aiBusy = true
	})

	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.aiCancel = cancel

	go func() {
		defer cancel()
		var acc strings.Builder
		var frameErr string
		err := a.client.AIStreamSidebar(ctx, req, func(ch qapi.AIStreamChunk) {
			acc.WriteString(ch.Content)
			snapshot := acc.String()
			if ch.Error != nil {
				frameErr = *ch.Error
			}
			a.mutate(func() {
				if n := len(a.aiTurns); n > 0 {
					a.aiTurns[n-1].Text = snapshot
				}
			})
		})
		a.mutate(func() {
			a.aiBusy = false
			a.aiCancel = nil
			if n := len(a.aiTurns); n > 0 {
				a.aiTurns[n-1].Streaming = false
			}
			switch {
			case frameErr != "":
				a.notice = frameErr
			case err != nil && ctx.Err() == nil:
				a.notice = "AI 请求失败：" + err.Error()
			}
		})
	}()
}
