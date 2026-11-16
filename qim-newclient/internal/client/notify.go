package client

// P4 系统集成：新消息桌面通知。
//
// 收到非当前会话的消息时，把「会话名 + 消息预览」交给操作系统弹通知。
// 走各平台自带命令，零第三方依赖，真机直接可用；测试用 fakeNotifier 注入，
// 只验证「在正确时机被调用、内容正确」这一层逻辑，不真的弹窗。

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// notifier 把一条通知交给操作系统。做成接口是为了替换成测试替身。
type notifier interface {
	Notify(title, body string) error
}

// desktopNotifier 用平台原生命令弹系统通知，不需要任何外部库。
type desktopNotifier struct{}

func (desktopNotifier) Notify(title, body string) error {
	title = sanitize(title)
	body = sanitize(body)
	switch runtime.GOOS {
	case "darwin":
		// osascript 的 display notification 把正文放前面、标题放 with title。
		return run("osascript", "-e",
			fmt.Sprintf(`display notification %q with title %q`, body, title))
	case "linux":
		return run("notify-send", title, body)
	case "windows":
		// msg 是 Windows 自带、无需桌面体验包，* 表示发给所有会话。
		return run("cmd", "/c", "msg", "*", title+": "+body)
	default:
		return fmt.Errorf("unsupported OS %s", runtime.GOOS)
	}
}

// fakeNotifier 记录所有调用，供单测断言。
type fakeNotifier struct {
	calls []notifyCall
}

type notifyCall struct {
	title string
	body  string
}

func (f *fakeNotifier) Notify(title, body string) error {
	f.calls = append(f.calls, notifyCall{title: title, body: body})
	return nil
}

// sanitize 去掉会让 shell / AppleScript 串行的换行，并截断超长文本，
// 避免一条长消息把通知撑爆或破坏命令结构。
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 200 {
		s = s[:197] + "..."
	}
	return s
}

func run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}
