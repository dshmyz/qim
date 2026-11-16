package client

// 富媒体缓存：mygo 的 ui.Image 只吃已经解码好的 Bitmap，不会自己去网络上
// 取图片。所以聊天里的每张图都要由 Go 侧下载 → 解码 → 缓存，再交给 UI。
//
// 三层：内存 Bitmap（每帧直接用）→ 磁盘缓存（重启后不用重下）→
// 真实 HTTP。单飞去重由 loading map 负责，避免 50 条消息里同一个头像
// 触发 50 个请求。所有网络都在 goroutine 里，结果通过 a.mutate 回 UI 线程。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

type mediaCache struct {
	mu          sync.Mutex
	images      map[string]ui.ImageSource // key → 解码后的位图
	errs        map[string]string         // key → 最近一次加载错误
	loading     map[string]bool           // key → 正在加载（单飞）
	localFiles  map[string]string         // key → 已落地文件路径
	downloading map[string]bool           // key → 正在下载/另存
	dir         string                    // 磁盘缓存根目录
	seq         uint64                    // 每次结果落盘自增，触发重绘
}

func (m *mediaCache) ensure() {
	if m.images != nil {
		return
	}
	m.images = make(map[string]ui.ImageSource)
	m.errs = make(map[string]string)
	m.loading = make(map[string]bool)
	m.localFiles = make(map[string]string)
	m.downloading = make(map[string]bool)
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	m.dir = filepath.Join(base, "qim-newclient", "media")
	_ = os.MkdirAll(m.dir, 0o700)
}

// mediaKey 是附件的稳定缓存键。本地待上传文件用「local:路径」，
// 已上传的用绝对 URL，只有 id 的用「id:xxx」。
func (a *app) mediaKey(att qapi.Attachment, localPath string) string {
	if localPath != "" {
		return "local:" + localPath
	}
	if att.URL != "" {
		if a.client != nil {
			return "url:" + a.client.ResolveURL(att.URL)
		}
		return "url:" + att.URL
	}
	return "id:" + att.ID
}

func cacheFileName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:32]
}

func (m *mediaCache) diskPath(key string) string {
	m.ensure()
	return filepath.Join(m.dir, cacheFileName(key))
}

// imageState 给渲染层用：返回位图 / 错误文案 / 是否加载中。
// 命中不到就顺手发起一次异步加载（单飞），下一帧自然就有图了。
func (a *app) imageState(att qapi.Attachment, localPath string) (ui.ImageSource, string, bool) {
	key := a.mediaKey(att, localPath)
	m := &a.media
	m.mu.Lock()
	m.ensure()
	src := m.images[key]
	errText := m.errs[key]
	busy := m.loading[key]
	// 失败后不能每帧重试：错误留在 errs 里，直到用户点占位块调用
	// invalidateMedia 清掉，才允许再发一次请求。
	canLoad := (localPath != "" || a.client != nil) && !busy && src == nil && errText == ""
	if canLoad {
		m.loading[key] = true
	}
	m.mu.Unlock()
	if canLoad {
		go a.fetchImage(a.client, key, att, localPath)
	}
	return src, errText, busy || canLoad
}

func (a *app) fetchImage(client *qapi.Client, key string, att qapi.Attachment, localPath string) {
	data, err := a.mediaBytes(client, key, att, localPath)
	var src ui.ImageSource
	if err == nil {
		src, err = ui.DecodeImage(data)
	}
	m := &a.media
	m.mu.Lock()
	m.ensure()
	delete(m.loading, key)
	if err != nil {
		m.errs[key] = friendlyError(err)
	} else {
		m.images[key] = src
		delete(m.errs, key)
	}
	m.seq++
	m.mu.Unlock()
	// 真窗口下 Update 会把状态变更排进下一帧；headless 下直接生效。
	a.mutate(func() {})
}

// mediaBytes 依次尝试：本地文件 → 磁盘缓存 → 服务端下载。
// 下载成功后写磁盘缓存（先写 .tmp 再 rename，避免半截文件被下次命中）。
func (a *app) mediaBytes(client *qapi.Client, key string, att qapi.Attachment, localPath string) ([]byte, error) {
	if localPath != "" {
		return os.ReadFile(localPath)
	}
	path := a.media.diskPath(key)
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return b, nil
	}
	if client == nil {
		return nil, fmt.Errorf("未登录，无法加载图片")
	}
	b, err := client.Download(a.ctx, att)
	if err != nil {
		return nil, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err == nil {
		_ = os.Rename(tmp, path)
	}
	return b, nil
}

// invalidateMedia 让某个 key 的下次加载重新走网络（加载失败后点重试）。
func (a *app) invalidateMedia(key string) {
	m := &a.media
	m.mu.Lock()
	m.ensure()
	delete(m.images, key)
	delete(m.errs, key)
	delete(m.loading, key)
	m.mu.Unlock()
	a.mutate(func() {})
}

// downloadState 给文件卡片显示「下载中 / 打开」。
func (a *app) downloadState(att qapi.Attachment, localPath string) (string, bool) {
	key := a.mediaKey(att, localPath)
	m := &a.media
	m.mu.Lock()
	m.ensure()
	path, busy := m.localFiles[key], m.downloading[key]
	m.mu.Unlock()
	return path, busy
}

// openAttachment 点击文件卡片：已下载就直接打开，否则下载到缓存目录再打开。
func (a *app) openAttachment(att qapi.Attachment) {
	key := a.mediaKey(att, "")
	m := &a.media
	m.mu.Lock()
	m.ensure()
	path := m.localFiles[key]
	busy := m.downloading[key]
	if !busy {
		m.downloading[key] = true
	}
	m.mu.Unlock()
	if path != "" {
		a.finishDownload(key, path, nil)
		return
	}
	if busy {
		return
	}
	client, ctx := a.client, a.ctx
	go func() {
		if client == nil {
			a.finishDownload(key, "", fmt.Errorf("未登录，无法下载附件"))
			return
		}
		data, err := client.Download(ctx, att)
		if err != nil {
			a.finishDownload(key, "", err)
			return
		}
		name := sanitizeFileName(att.DisplayName())
		dest := filepath.Join(a.media.diskPath(key)+"-files", name)
		_ = os.MkdirAll(filepath.Dir(dest), 0o700)
		tmp := dest + ".tmp"
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			a.finishDownload(key, "", err)
			return
		}
		if err := os.Rename(tmp, dest); err != nil {
			a.finishDownload(key, "", err)
			return
		}
		a.finishDownload(key, dest, nil)
	}()
}

// saveAttachmentAs 右键「另存为」：原生 Save 对话框在 goroutine 里调用，
// 不能放在 UI 构建回调里（会阻塞主线程等自己投递的任务）。
func (a *app) saveAttachmentAs(att qapi.Attachment) {
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       "保存附件",
			ButtonLabel: "保存",
			DefaultPath: sanitizeFileName(att.DisplayName()),
			Parent:      a.win,
		})
		if err != nil || path == "" {
			return
		}
		client, ctx := a.client, a.ctx
		if client == nil {
			a.setNotice("未登录，无法下载附件")
			return
		}
		data, err := client.Download(ctx, att)
		if err != nil {
			a.setNotice("下载失败：" + friendlyError(err))
			return
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			a.setNotice("保存失败：" + friendlyError(err))
			return
		}
		a.setNotice("已保存到 " + path)
	}()
}

func (a *app) finishDownload(key, path string, err error) {
	m := &a.media
	m.mu.Lock()
	m.ensure()
	delete(m.downloading, key)
	if err == nil && path != "" {
		m.localFiles[key] = path
	}
	m.seq++
	m.mu.Unlock()
	if err != nil {
		a.setNotice("下载失败：" + friendlyError(err))
	} else if path != "" {
		if oerr := openLocalFile(path); oerr != nil {
			log.Printf("[media] 打开文件失败 %s: %v", path, oerr)
			a.setNotice("已保存到 " + path)
		} else {
			a.setNotice("已保存到 " + path + " 并打开")
		}
	}
	a.mutate(func() {})
}

// openLocalFile 用系统默认程序打开本地文件；三平台分别对应
// open / xdg-open / rundll32，避免引入额外 GUI 依赖。
func openLocalFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		return fmt.Errorf("不支持的平台 %s", runtime.GOOS)
	}
	return cmd.Start()
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" || name == "." || name == ".." {
		return "attachment"
	}
	return name
}
