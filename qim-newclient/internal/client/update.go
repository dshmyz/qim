package client

// 检查更新（对齐 qim-client Main.vue 的 checkUpdateViaAPI / api/versions.ts）。
//
// 设置页「关于」组点击「检查更新」→ 弹窗 → GET /client/versions?platform=X&pageSize=1
// （公开接口）。有新版本显示版本号 + 更新说明 + 下载入口（系统浏览器打开
// downloadUrl）；无记录或同版本显示「当前已是最新版本」。
// 版本号比较用点分段数字比较，与 qim-client 的 isNewerVersion 一致；
// 基准版本用 clientVersion（对齐 Electron 青雀 2.0.39），不是设置页展示的
// appVersion——后端 client_versions 记的是 Electron 客户端的版本线。

import (
	"context"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// detectPlatform 与 qapi 上报的 X-App-Platform 同源。
func detectPlatform() string {
	return qapi.PlatformOf(runtime.GOOS)
}

// isNewerVersion 点分段版本号比较：任何一段更大即更新。
// 非数字段按 0 处理，长度不等按位数补齐。
func isNewerVersion(newVer, curVer string) bool {
	np := strings.Split(strings.TrimPrefix(newVer, "v"), ".")
	cp := strings.Split(strings.TrimPrefix(curVer, "v"), ".")
	for i := 0; i < len(np) || i < len(cp); i++ {
		n, c := 0, 0
		if i < len(np) {
			n, _ = strconv.Atoi(strings.TrimSpace(np[i]))
		}
		if i < len(cp) {
			c, _ = strconv.Atoi(strings.TrimSpace(cp[i]))
		}
		if n > c {
			return true
		}
		if n < c {
			return false
		}
	}
	return false
}

// openURL 用系统默认浏览器打开链接。仅放行 http/https，防命令注入。
func openURL(rawURL string) error {
	u := strings.TrimSpace(rawURL)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return nil // 非网页链接直接忽略，不执行
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}

// checkUpdate 打开检查更新弹窗并异步请求最新版本。
func (a *app) checkUpdate() {
	if a.updChecking {
		return
	}
	a.updOpen = true
	a.updChecking = true
	a.updHasNew = false
	a.updInfo = nil
	a.updResult = "正在检查更新..."
	if a.client == nil || !a.loggedIn {
		// headless / 未登录：直接按"已是最新"收尾，别一直转圈
		a.updChecking = false
		a.updResult = "当前已是最新版本"
		return
	}
	platform := detectPlatform()
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	go func() {
		defer cancel()
		v, err := a.client.LatestVersion(ctx, platform)
		a.mutate(func() {
			a.updChecking = false
			if err != nil {
				log.Printf("[update] 检查更新失败: %v", err)
				a.updResult = "检查更新失败：" + friendlyError(err)
				return
			}
			if v == nil || !isNewerVersion(v.Version, qapi.ElectronClientVersion) {
				a.updResult = "当前已是最新版本"
				return
			}
			a.updHasNew = true
			a.updResult = "发现新版本 v" + v.Version
			a.updInfo = v
		})
	}()
}

func (a *app) closeUpdate() {
	a.updOpen = false
}

// openUpdateDownload 用系统浏览器打开下载页（对齐非 Electron 环境的降级行为）。
func (a *app) openUpdateDownload() {
	if a.updInfo == nil || a.updInfo.DownloadURL == "" {
		return
	}
	if err := openURL(a.updInfo.DownloadURL); err != nil {
		a.setNotice("打开下载链接失败：" + err.Error())
	}
}

// updateDialog 检查更新弹窗（对齐 qim-client 的 UpdateDialog：状态文案 + 新版本信息）。
func (a *app) updateDialog(c *ui.Context) {
	if !a.updOpen {
		return
	}
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(420).Radius(12).Background(theme.White).Padding(24, 24).Gap(12).
				AlignItems(ui.Center).Children(func() {
				ui.Text(c, "检查更新").FontSize(16).FontWeight(600).TextColor(theme.Gray800)
				ui.Text(c, a.updResult).FontSize(13).TextColor(theme.Gray600)
				if a.updChecking {
					ui.Text(c, "…").FontSize(18).TextColor(theme.Gray500)
				}
				if a.updHasNew && a.updInfo != nil {
					info := a.updInfo
					if info.ReleaseDate != "" {
						ui.Text(c, "发布时间："+info.ReleaseDate).FontSize(12).TextColor(theme.Gray500)
					}
					if notes := strings.TrimSpace(info.UpdateNotes); notes != "" {
						ui.Box(c).FillWidth().MaxHeight(160).Radius(8).Background(theme.Gray50).
							Padding(10, 12).Children(func() {
							ui.Text(c, notes).FontSize(12).TextColor(theme.Gray800)
						})
					}
					if info.DownloadURL != "" {
						dl := ui.Box(c).Height(34).Radius(8).Padding(0, 16).
							Background(theme.Primary).Center()
						if dl.Clicked() {
							a.openUpdateDownload()
						}
						dl.Children(func() {
							ui.Text(c, "前往下载").FontSize(13).TextColor(theme.White)
						})
					}
				}
				cancel := ui.Box(c).Height(32).Radius(8).Padding(0, 16).
					Background(theme.Gray100).Center()
				if cancel.Clicked() {
					a.closeUpdate()
				}
				cancel.Children(func() {
					ui.Text(c, "关闭").FontSize(13).TextColor(theme.Gray600)
				})
			})
		})
	})
}
