package client

// 斜杠命令首次引导条，对齐 qim-client 的 SlashCommandHint.vue：
// 只展示一次，点「试一下」或 ✕ 后本地记住，重启不再出现；「试一下」
// 会在输入框插入 / 并复用 / 命令面板链路，不是死按钮。
//
// 原版用 localStorage（key=qim:slash_hint_dismissed）；本端没有 WebView，
// 用 UserConfigDir 下的一个 JSON 文件存 UI 偏好（非敏感信息，不进钥匙串）。
// 读失败降级为「每次展示」，写失败仍保证本次会话内不再出现。

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"qim-newclient/internal/theme"
	"strings"

	"github.com/egoist/mygo/ui"
)

const slashHintPrefFile = "ui-prefs.json"

// hintStore 是 UI 偏好的存储抽象；headless 测试注入 memHintStore，
// 避免真的往用户配置目录写文件。
type hintStore interface {
	Dismissed() bool
	Dismiss()
}

type hintPrefs struct {
	SlashHintDismissed bool `json:"slash_hint_dismissed"`
}

// memHintStore 只在进程内记住，给测试和文件不可用时的降级路径用。
type memHintStore struct{ dismissed bool }

func (m *memHintStore) Dismissed() bool { return m.dismissed }
func (m *memHintStore) Dismiss()        { m.dismissed = true }

// fileHintStore 把偏好写到 ~/Library/Application Support/qim-newclient
// （Windows/Linux 由 os.UserConfigDir 决定），读一次缓存在内存里。
type fileHintStore struct {
	mem  memHintStore
	path string
}

func newFileHintStore() hintStore {
	s := &fileHintStore{}
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("[hint] 取不到用户配置目录，引导条只在本次会话内记忆: %v", err)
		return s
	}
	s.path = filepath.Join(dir, "qim-newclient", slashHintPrefFile)
	s.load()
	return s
}

// load 从磁盘恢复偏好；首次运行或文件损坏都按「未关闭」处理。
func (s *fileHintStore) load() {
	if s.path == "" {
		return
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var p hintPrefs
	if err := json.Unmarshal(b, &p); err != nil {
		log.Printf("[hint] 偏好文件解析失败，按未关闭处理: %v", err)
		return
	}
	s.mem.dismissed = p.SlashHintDismissed
}

func (s *fileHintStore) Dismissed() bool { return s.mem.Dismissed() }

func (s *fileHintStore) Dismiss() {
	s.mem.Dismiss()
	if s.path == "" {
		return
	}
	b, err := json.Marshal(hintPrefs{SlashHintDismissed: true})
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		log.Printf("[hint] 写偏好失败: %v", err)
		return
	}
	if err := os.WriteFile(s.path, b, 0o600); err != nil {
		log.Printf("[hint] 写偏好失败: %v", err)
	}
}

// hints 惰性取存储；测试里直接 &app{} 时用内存实现，渲染照常展示引导条。
func (a *app) hints() hintStore {
	if a.hint == nil {
		a.hint = &memHintStore{}
	}
	return a.hint
}

// dismissSlashHint 关闭引导条并持久化（对齐 handleClose）。
func (a *app) dismissSlashHint() { a.hints().Dismiss() }

// trySlashHint 对齐 handleSlashHintTry：插入 / 后由 view() 每帧调用的
// maybeTriggerSlash 自然弹出命令面板。本端命令面板按输入框开头识别命令
// （见 quickTriggerText），已有草稿时把 / 放在开头、草稿留作过滤词，
// 既不丢用户输入，也不会出现「点了没反应」。
func (a *app) trySlashHint() {
	a.dismissSlashHint()
	draft := strings.TrimSpace(a.input)
	if draft == "" {
		a.input = "/"
		return
	}
	a.input = "/ " + draft
}

// slashHint 渲染首次引导条。样式对齐 .slash-hint：
// padding 7/12、margin-bottom 6、主色 8% 浅底 + 20% 边框、radius 8、
// 12px 次要色文案、白底小键帽 /、主色「试一下」、20x20 关闭钮。
func (a *app) slashHint(c *ui.Context) {
	if a.hint != nil && a.hint.Dismissed() {
		return
	}
	ui.Row(c).FillWidth().Padding(0, 15, 6, 15).Children(func() {
		bar := ui.Row(c).FillWidth().Padding(7, 12, 7, 12).Gap(10).AlignItems(ui.Center).
			Radius(8).Background(theme.PrimarySoft).
			BorderWidth(1).BorderColor(ui.Hex("#bfdbfe"))
		bar.Children(func() {
			ui.Icon(c, iconZap).FontSize(12).TextColor(theme.Primary)
			ui.Text(c, "在输入框输入").FontSize(12).TextColor(theme.Gray500)
			ui.Box(c).Radius(4).Padding(1, 6).Background(theme.White).
				BorderWidth(1).BorderColor(theme.Gray200).Children(func() {
				ui.Text(c, "/").FontSize(12).FontWeight(600).TextColor(theme.Primary)
			})
			ui.Text(c, "可快速插入文件、笔记、任务、短语").FontSize(12).TextColor(theme.Gray500)
			ui.Spacer(c)

			// 显式宽度：Box 在 Row 里不设宽会被收缩，12px 文字会被挤成重影。
			try := ui.Box(c).Width(58).Height(22).Radius(4).Background(theme.Primary).Center()
			if try.Hovered() {
				try = try.Background(theme.PrimaryStrong)
			}
			if try.Clicked() {
				a.trySlashHint()
			}
			try.Children(func() {
				ui.Text(c, "试一下").FontSize(12).TextColor(theme.White)
			})

			closeBtn := ui.Box(c).Width(20).Height(20).Radius(4).Center()
			if closeBtn.Hovered() {
				closeBtn = closeBtn.Background(ui.Hex("#dbeafe"))
			}
			if closeBtn.Clicked() {
				a.dismissSlashHint()
			}
			closeBtn.Children(func() {
				ui.Icon(c, iconClose).FontSize(12).TextColor(theme.Gray500)
			})
		})
	})
}
