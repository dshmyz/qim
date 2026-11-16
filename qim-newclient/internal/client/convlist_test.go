package client

// 左侧会话列表（ConversationList.vue）对齐验证：
// 相对时间文案、预览文案（群聊带发送者名 / 无消息兜底）、成员数、
// 草稿预览、右键破坏性动作的真实 HTTP 契约、确认弹层与截图。

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

// TestConvTimeLabel 对齐 ConversationList.vue 的 formatTime：按时间差给
// 刚刚 / N分钟前 / N小时前 / 昨天 / N天前，超过一周才用 MM-DD。
func TestConvTimeLabel(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"刚刚", 20 * time.Second, "刚刚"},
		{"分钟", 5 * time.Minute, "5分钟前"},
		{"小时", 3 * time.Hour, "3小时前"},
		{"23 小时仍是小时", 23 * time.Hour, "23小时前"},
		{"昨天", 30 * time.Hour, "昨天"},
		{"多天", 3 * 24 * time.Hour, "3天前"},
	}
	for _, c := range cases {
		if got := convTimeLabel(now.Add(-c.ago).UnixMilli()); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
	// 超过一周退回 MM-DD（用固定的过去时间，避免跨月边界抖动）
	old := time.Now().Add(-20 * 24 * time.Hour)
	if got := convTimeLabel(old.UnixMilli()); got != old.Format("01-02") {
		t.Errorf("超一周应显示 MM-DD: got %q want %q", got, old.Format("01-02"))
	}
	if got := convTimeLabel(0); got != "" {
		t.Errorf("无时间戳应为空: %q", got)
	}
}

// TestConvsFromAPIPreview 对齐 formatMessagePreview：群聊在正文前拼发送者名，
// 单聊不拼；没有最后一条消息时给「暂无消息」而不是空白行。
func TestConvsFromAPIPreview(t *testing.T) {
	list := []qapi.Conversation{
		{ID: qapi.ID("1"), Type: "group", Name: "项目群",
			LastMessage: &qapi.Message{Type: "text", Content: "收到", Sender: &qapi.Sender{Name: "张三"}}},
		{ID: qapi.ID("2"), Type: "single", Name: "李四",
			LastMessage: &qapi.Message{Type: "text", Content: "好的", Sender: &qapi.Sender{Name: "李四"}}},
		{ID: qapi.ID("3"), Type: "bot", Name: "AI 助手"},
		// 有消息但没带 sender：不拼兜底名（否则会出现「未知用户: 早」）
		{ID: qapi.ID("4"), Type: "group", Name: "无发送者",
			LastMessage: &qapi.Message{Type: "text", Content: "早"}},
	}
	got := convsFromAPI(list)
	want := []string{"张三: 收到", "好的", "暂无消息", "早"}
	for i, w := range want {
		if got[i].Preview != w {
			t.Errorf("第 %d 条预览: got %q want %q", i, got[i].Preview, w)
		}
	}
}

// TestConvSubLabel 成员数只在群聊且成员已加载时显示（原版
// `type === 'group' && members`），讨论组/单聊不显示。
func TestConvSubLabel(t *testing.T) {
	group := conversation{Kind: "group", Members: []member{{UserID: 1}, {UserID: 2}}}
	if got := convSubLabel(group); got != "(2人)" {
		t.Errorf("群聊成员数: got %q", got)
	}
	if got := convSubLabel(conversation{Kind: "discussion", Members: []member{{UserID: 1}}}); got != "" {
		t.Errorf("讨论组不该显示成员数: %q", got)
	}
	if got := convSubLabel(conversation{Kind: "single", Members: []member{{UserID: 1}}}); got != "" {
		t.Errorf("单聊不该显示成员数: %q", got)
	}
	// 显式 sub 优先（测试夹具与接口直出的补充文案都走这里）
	if got := convSubLabel(conversation{Kind: "group", Sub: "(9人)"}); got != "(9人)" {
		t.Errorf("sub 应优先: %q", got)
	}
}

// TestConvDraft 当前会话读输入框实时内容，其他会话读切走时存下的草稿。
func TestConvDraft(t *testing.T) {
	a := &app{
		active: 0,
		input:  "正在输入 A",
		convs:  []conversation{{ID: "a"}, {ID: "b"}},
		drafts: map[string]string{"b": "B 的草稿\n第二行"},
	}
	if got := a.convDraft("a"); got != "正在输入 A" {
		t.Errorf("当前会话草稿: %q", got)
	}
	if got := oneLine(a.convDraft("b")); got != "B 的草稿 第二行" {
		t.Errorf("其他会话草稿应压成一行: %q", got)
	}
	if got := a.convDraft("c"); got != "" {
		t.Errorf("没有草稿应为空: %q", got)
	}
}

// TestAskConvAction 右键「退出群聊 / 移除会话」只负责打开确认弹层，
// 弹层没确认前不能动会话列表。
func TestAskConvAction(t *testing.T) {
	a := &app{convs: []conversation{{ID: "106", Name: "项目群", Kind: "group"}}}
	a.askConvAction("exit", "106", "项目群")
	if a.convConfirmKind != "exit" || a.convConfirmID != "106" || a.convConfirmName != "项目群" {
		t.Fatalf("确认弹层状态不对: %+v", a)
	}
	if len(a.convs) != 1 {
		t.Error("没确认就改了会话列表")
	}
	a.closeConvConfirm()
	if a.convConfirmKind != "" || a.convConfirmID != "" {
		t.Errorf("取消后没清干净: %+v", a)
	}
}

// fakeConvServer 只服务会话删除 / 退群两个端点，其余 404。
func fakeConvServer(t *testing.T, path, method string, hit *bool, mu *sync.Mutex) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method && r.URL.Path == path {
			mu.Lock()
			*hit = true
			mu.Unlock()
			io.WriteString(w, `{"code":0,"message":"ok"}`)
			return
		}
		w.WriteHeader(404)
	}))
}

// TestConfirmConvActionRemove 确认移除会话：DELETE /conversations/:id，
// 本地先摘掉行，当前会话被摘掉时回到占位页。
func TestConfirmConvActionRemove(t *testing.T) {
	var mu sync.Mutex
	hit := false
	srv := fakeConvServer(t, "/api/v1/conversations/2", http.MethodDelete, &hit, &mu)
	defer srv.Close()

	a := &app{
		loggedIn: true, client: qapi.New(srv.URL), ctx: context.Background(),
		active: 1,
		convs:  []conversation{{ID: "1", Name: "项目群"}, {ID: "2", Name: "李四"}},
		drafts: map[string]string{"2": "没发出去的草稿"},
	}
	a.askConvAction("remove", "2", "李四")
	a.confirmConvAction()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := hit
		mu.Unlock()
		a.mu.RLock()
		n, active := len(a.convs), a.active
		a.mu.RUnlock()
		if ok && n == 1 && active == -1 {
			if _, left := a.drafts["2"]; left {
				t.Error("移除会话后草稿没清")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.RLock()
	n, active := len(a.convs), a.active
	a.mu.RUnlock()
	t.Fatalf("移除会话链路未完成: hit=%v convs=%d active=%d", hit, n, active)
}

// TestConfirmConvActionExit 确认退出群聊：POST /groups/:id/exit，
// 退出后会话从列表消失，active 归位到 -1。
func TestConfirmConvActionExit(t *testing.T) {
	var mu sync.Mutex
	hit := false
	srv := fakeConvServer(t, "/api/v1/groups/106/exit", http.MethodPost, &hit, &mu)
	defer srv.Close()

	a := &app{
		loggedIn: true, client: qapi.New(srv.URL), ctx: context.Background(),
		active: 0,
		convs:  []conversation{{ID: "106", Name: "项目群", Kind: "group"}},
	}
	a.askConvAction("exit", "106", "项目群")
	a.confirmConvAction()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := hit
		mu.Unlock()
		a.mu.RLock()
		n, active := len(a.convs), a.active
		a.mu.RUnlock()
		if ok && n == 0 && active == -1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.RLock()
	n, active := len(a.convs), a.active
	a.mu.RUnlock()
	t.Fatalf("退群链路未完成: hit=%v convs=%d active=%d", hit, n, active)
}

// TestConvConfirmRender 确认弹层的标题/正文/按钮齐全（退出与移除各一套）。
func TestConvConfirmRender(t *testing.T) {
	cases := []struct {
		kind  string
		name  string
		wants []string
	}{
		{"exit", "项目群", []string{"确认退出群聊", "项目群", "取消", "退出"}},
		{"remove", "李四", []string{"移除会话", "李四", "取消", "移除"}},
	}
	for _, c := range cases {
		a := &app{
			active:          0,
			loggedIn:        true,
			connStatus:      qapi.StatusConnected,
			convConfirmKind: c.kind,
			convConfirmID:   "106",
			convConfirmName: c.name,
			convs:           []conversation{{ID: "106", Name: c.name, Kind: "group"}},
		}
		tt := ui.NewTester(a.view, 1050, 703)
		tt.Frame()
		tt.Frame()
		for _, w := range c.wants {
			if !tt.HasText(w) {
				t.Errorf("%s 弹层缺 %q, texts=%q", c.kind, w, tt.Texts())
			}
		}
		if c.kind == "exit" {
			write(t, tt, "/tmp/qim-conv-confirm.png")
		}
	}
}

// TestConvListScroll 侧栏会话列表必须能滚：会话多到一屏放不下时，
// 滚轮/触控板能滚到后面的会话。原来直接 for 循环往 Column 里画，
// 超出的行被裁掉且没有任何滚动容器，列表等于「卡死」。
func TestConvListScroll(t *testing.T) {
	a := &app{loggedIn: true, userName: "管理员", connStatus: qapi.StatusConnected, selfID: "7", active: -1}
	for i := range 30 {
		a.convs = append(a.convs, conversation{
			ID: fmt.Sprintf("c%d", i), Name: fmt.Sprintf("会话%02d", i),
			Kind: "single", Preview: "预览", Time: "刚刚", Color: "#3385ff",
		})
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("会话00") {
		t.Fatalf("首屏应看到第一条会话, texts=%q", tt.Texts())
	}
	if tt.HasText("会话29") {
		t.Fatalf("30 条会话一屏放不下，末尾不该直接可见, texts=%q", tt.Texts())
	}
	// 指针放在侧栏列表区域内往下滚（选区避开顶栏和输入区）。
	tt.Scroll(150, 420, 0, 4000)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("会话29") {
		t.Fatalf("滚到底后应看到最后一条会话, texts=%q", tt.Texts())
	}
	if tt.HasText("会话00") {
		t.Fatalf("滚到底后第一条应滚出视口, texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-convlist-scroll.png")
}

// TestShotConvList 会话列表四种状态：群聊（成员数 + 未读）、单聊（免打扰）、
// 讨论组（草稿预览）、长名称（省略号）。
func TestShotConvList(t *testing.T) {
	a := &app{
		active:     0,
		loggedIn:   true,
		userName:   "管理员",
		connStatus: qapi.StatusConnected,
		selfID:     "7",
		convs: []conversation{
			{ID: "106", Name: "项目群", Kind: "group", Preview: "张三: 收到，我下午先整理一版",
				Time: "刚刚", Unread: 12, Color: "#3385ff", Pinned: true,
				Members:      []member{{UserID: 8, Name: "张三"}, {UserID: 9, Name: "李四"}, {UserID: 7, Name: "我"}},
				Announcement: "本周五下午 3 点项目评审会，请提前准备材料"},
			{ID: "2", Name: "李四", Kind: "single", Preview: "好的，明天见",
				Time: "2小时前", Unread: 120, Muted: true, Color: "#26b361",
				IP: "10.0.1.23", Signature: "专注技术，热爱开源", Status: "online"},
			{ID: "3", Name: "产品讨论组", Kind: "discussion", Preview: "聊天记录",
				Time: "昨天", Color: "#7c5cff"},
			{ID: "4", Name: "一个名字很长很长很长很长很长很长很长的群", Kind: "group",
				Preview: "李四: 这条消息也很长很长很长很长很长很长很长很长很长",
				Time:    "3天前", Color: "#f59e0b"},
		},
		drafts: map[string]string{"3": "还没写完的一段讨论组草稿"},
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"项目群", "(3人)", "12", "99+", "[草稿] 还没写完的一段讨论组草稿", "3天前", "管理员"} {
		if !tt.HasText(want) {
			t.Errorf("会话列表缺 %q, texts=%q", want, tt.Texts())
		}
	}
	// 侧栏头部对齐原版：没有「退出」按钮；顶栏没有连接状态/截图入口。
	for _, bad := range []string{"退出", "已连接", "📷 截图"} {
		if tt.HasText(bad) {
			t.Errorf("侧栏顶部/顶栏不该有 %q, texts=%q", bad, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-convlist.png")
}

// TestConvRowLongNameMemberCount 长群名不能把「(2人)」挤成两行
// （原实现两个 Text 都不设伸缩，长名称会把成员数折行成「(2」「人)」）。
func TestConvRowLongNameMemberCount(t *testing.T) {
	a := &app{
		active: -1, loggedIn: true, userName: "管理员",
		connStatus: qapi.StatusConnected, selfID: "7",
		convs: []conversation{{
			ID: "10", Name: "[已解散] 资料探针测试群名称很长很长很长",
			Kind: "group", Preview: "暂无消息", Time: "9小时前", Color: "#f59e0b",
			Members: []member{{UserID: 1}, {UserID: 2}},
		}},
	}
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("(2人)") {
		t.Fatalf("成员数应完整单行显示, texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-convrow-long.png")
}
