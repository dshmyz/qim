package client

// 输入区对齐专项：斜杠首次引导的记忆与「试一下」、工具栏/发送按钮状态、
// Enter 与按钮共用的提交路径，以及三张渲染截图。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// composerApp 造一个能渲染出聊天输入区的最小 app（无后端，headless 安全）。
func composerApp() *app {
	return &app{
		active: 0, loggedIn: true, selfID: "7", userName: "我",
		convs: []conversation{{ID: "101", Name: "项目群", Kind: "group"}},
		msgs:  []message{{Key: "m1", Sender: "张三", Text: "收到", At: "17:00"}},
	}
}

// TestSlashHintVisibility：引导条默认展示，关闭后立刻消失且被记住。
func TestSlashHintVisibility(t *testing.T) {
	a := composerApp()
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("试一下") || !tt.HasText("可快速插入文件、笔记、任务、短语") {
		t.Fatalf("首次引导条没渲染: texts=%q", tt.Texts())
	}
	a.dismissSlashHint()
	if !a.hints().Dismissed() {
		t.Fatal("关闭后存储没记住")
	}
	tt.Frame()
	if tt.HasText("试一下") {
		t.Errorf("关闭后引导条仍在: texts=%q", tt.Texts())
	}
}

// TestSlashHintClick：真点「试一下」整条链路——关引导条、记住、插入 /、
// 命令面板上屏（不是只测内部函数）。
func TestSlashHintClick(t *testing.T) {
	a := composerApp()
	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	if err := tt.Click("试一下"); err != nil {
		t.Fatalf("点不到「试一下」: %v", err)
	}
	tt.Frame()
	tt.Frame()
	if !a.hints().Dismissed() || a.input != "/" {
		t.Errorf("点击没生效: dismissed=%v input=%q", a.hints().Dismissed(), a.input)
	}
	if tt.HasText("试一下") {
		t.Error("点击后引导条没消失")
	}
	if !tt.HasText("快速回复（/ 或 /quick）") {
		t.Errorf("点击后没弹命令面板: texts=%q", tt.Texts())
	}
}

// TestSlashHintTry：「试一下」记住关闭状态，插入 / 并触发命令面板；
// 已有草稿时不丢内容，草稿退化为过滤词。
func TestSlashHintTry(t *testing.T) {
	a := composerApp()
	a.trySlashHint()
	if a.input != "/" {
		t.Fatalf("空输入时没插入 /: %q", a.input)
	}
	if !a.hints().Dismissed() {
		t.Error("试一下没记住关闭状态")
	}
	a.maybeTriggerSlash()
	if !a.quickOpen {
		t.Error("插入 / 后命令面板没打开")
	}

	b := composerApp()
	b.input = "  晚上评审  "
	b.trySlashHint()
	if b.input != "/ 晚上评审" {
		t.Fatalf("已有草稿时内容处理不对: %q", b.input)
	}
	if q, ok := quickTriggerText(b.input); !ok || q != "晚上评审" {
		t.Errorf("草稿没被当作过滤词: q=%q ok=%v", q, ok)
	}
}

// TestFileHintStorePersists：偏好落盘后重开进程仍能读到（对齐 localStorage）。
func TestFileHintStorePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", slashHintPrefFile)
	s1 := &fileHintStore{path: path}
	if s1.Dismissed() {
		t.Fatal("首次运行不该是已关闭")
	}
	s1.Dismiss()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("偏好文件没写出: %v", err)
	}
	var p hintPrefs
	if err := json.Unmarshal(b, &p); err != nil || !p.SlashHintDismissed {
		t.Fatalf("偏好文件内容不对: %q err=%v", b, err)
	}

	s2 := &fileHintStore{path: path}
	s2.load()
	if !s2.Dismissed() {
		t.Error("重启后没恢复「已关闭」")
	}
}

// TestCanSendComposer：对齐原版 send-btn 的 disabled 条件。
func TestCanSendComposer(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		pending int
		want    bool
	}{
		{"空输入", "", 0, false},
		{"纯空白", "   \n ", 0, false},
		{"有文字", " 你好 ", 0, true},
		{"只有待发附件", "", 1, true},
		{"文字加附件", "看图", 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{input: tc.input, pendingFiles: make([]string, tc.pending)}
			if got := a.canSendComposer(); got != tc.want {
				t.Errorf("canSendComposer()=%v, 期望 %v", got, tc.want)
			}
		})
	}
}

// TestSubmitComposer：Enter 与发送按钮共用的路径会发一条去空白的消息。
func TestSubmitComposer(t *testing.T) {
	a := &app{active: 0, convs: []conversation{{ID: "101"}}}
	a.submitComposer("  晚上一起评审  ")
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.msgs) != 1 || a.msgs[0].Text != "晚上一起评审" || !a.msgs[0].Mine {
		t.Fatalf("提交没上屏: msgs=%+v", a.msgs)
	}
}

// TestShotComposer：默认 / 有输入 / 命令面板三态截图，肉眼看工具栏顺序、
// 引导条、白底输入框、发送按钮位置与禁用态。
func TestShotComposer(t *testing.T) {
	cases := []struct {
		file  string
		setup func(*app)
	}{
		{"/tmp/qim-composer-default.png", func(a *app) {}},
		{"/tmp/qim-composer-typing.png", func(a *app) {
			a.dismissSlashHint()
			a.input = "晚上一起评审一下方案"
		}},
		{"/tmp/qim-composer-slash.png", func(a *app) { a.trySlashHint() }},
	}
	for _, tc := range cases {
		a := composerApp()
		tc.setup(a)
		tt := ui.NewTester(a.view, 1050, 703)
		tt.Frame()
		tt.Frame()
		if !tt.HasText("发送") || !tt.HasText("按 Enter 发送，Shift+Enter 换行") {
			t.Errorf("%s 输入区缺底部操作行: texts=%q", tc.file, tt.Texts())
		}
		write(t, tt, tc.file)
	}
}
