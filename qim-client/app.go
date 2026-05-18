package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	screenshot *ScreenshotManager
	updater    *UpdateManager
	hasUnread  bool
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	a.screenshot = NewScreenshotManager()
	a.updater = NewUpdateManager()

	go func() {
		runtime.EventsOn(ctx, "tray-show-window", func(optional ...interface{}) {
			runtime.WindowShow(ctx)
			runtime.WindowSetAlwaysOnTop(ctx, true)
			runtime.WindowSetAlwaysOnTop(ctx, false)
		})

		runtime.EventsOn(ctx, "tray-quit-app", func(optional ...interface{}) {
			runtime.Quit(ctx)
		})

		runtime.EventsOn(ctx, "hotkey-screenshot", func(optional ...interface{}) {
			a.StartScreenshot()
		})
	}()
}

func (a *App) shutdown(ctx context.Context) {
}

func (a *App) MinimizeWindow() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) MaximizeWindow() {
	if runtime.WindowIsMaximised(a.ctx) {
		runtime.WindowUnmaximise(a.ctx)
	} else {
		runtime.WindowMaximise(a.ctx)
	}
}

func (a *App) CloseWindow() {
	runtime.WindowHide(a.ctx)
}

func (a *App) IsMaximized() bool {
	return runtime.WindowIsMaximised(a.ctx)
}

func (a *App) OpenExternal(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

type FileDialogOptions struct {
	Title      string   `json:"title"`
	DefaultDir string   `json:"defaultDir"`
	Filters    []string `json:"filters"`
}

type FileDialogResult struct {
	Canceled bool   `json:"canceled"`
	FilePath string `json:"filePath"`
}

func (a *App) OpenFileDialog(opts FileDialogOptions) (*FileDialogResult, error) {
	filters := []runtime.FileFilter{}
	for _, f := range opts.Filters {
		filters = append(filters, runtime.FileFilter{
			DisplayName: f,
			Pattern:     "*.*",
		})
	}
	if len(filters) == 0 {
		filters = []runtime.FileFilter{
			{DisplayName: "All Files", Pattern: "*.*"},
		}
	}

	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                opts.Title,
		DefaultDirectory:     opts.DefaultDir,
		CanCreateDirectories: true,
	})
	if err != nil {
		return &FileDialogResult{Canceled: true}, nil
	}
	if path == "" {
		return &FileDialogResult{Canceled: true}, nil
	}
	return &FileDialogResult{Canceled: false, FilePath: path}, nil
}

func (a *App) SaveFileAs(fileName string, data []byte) (*FileDialogResult, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存文件",
		DefaultFilename: fileName,
	})
	if err != nil {
		return &FileDialogResult{Canceled: true}, nil
	}
	if path == "" {
		return &FileDialogResult{Canceled: true}, nil
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return &FileDialogResult{Canceled: false, FilePath: path}, fmt.Errorf("write file failed: %w", err)
	}

	return &FileDialogResult{Canceled: false, FilePath: path}, nil
}

func (a *App) DownloadFile(fileName string, data []byte, saveDir string) (*FileDialogResult, error) {
	targetDir := saveDir
	if targetDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		targetDir = filepath.Join(home, "Downloads")
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(targetDir, fileName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, err
	}

	return &FileDialogResult{Canceled: false, FilePath: filePath}, nil
}

type AppInfo struct {
	Version     string `json:"version"`
	Platform    string `json:"platform"`
	UserDataDir string `json:"userDataDir"`
}

func (a *App) GetAppInfo() *AppInfo {
	return &AppInfo{
		Version:     "1.0.0",
		Platform:    runtime.Environment(a.ctx).Platform,
		UserDataDir: a.getUserDataDir(),
	}
}

func (a *App) getUserDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".qim", "app")
	os.MkdirAll(dir, 0755)
	return dir
}

func (a *App) getCacheDir() string {
	return filepath.Join(a.getUserDataDir(), "avatar-cache")
}

func (a *App) CacheAvatar(avatarUrl string) (string, error) {
	cacheDir := a.getCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return avatarUrl, err
	}

	hash := md5.Sum([]byte(avatarUrl))
	ext := filepath.Ext(avatarUrl)
	if len(ext) > 10 || ext == "" {
		ext = ".png"
	}

	cachePath := filepath.Join(cacheDir, hex.EncodeToString(hash[:])+ext)

	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(avatarUrl)
	if err != nil {
		return avatarUrl, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return avatarUrl, fmt.Errorf("fetch avatar failed: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return avatarUrl, err
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return avatarUrl, err
	}

	return cachePath, nil
}

func (a *App) CleanupAvatarCache(maxAgeDays int) error {
	cacheDir := a.getCacheDir()
	if maxAgeDays <= 0 {
		maxAgeDays = 7
	}
	maxAge := time.Duration(maxAgeDays) * 24 * time.Hour
	now := time.Now()

	return filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if now.Sub(info.ModTime()) > maxAge {
			os.Remove(path)
		}
		return nil
	})
}

func (a *App) FlashTray(enabled bool) {
	a.hasUnread = enabled
	runtime.EventsEmit(a.ctx, "tray-flash", enabled)
}

type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Url       string `json:"url"`
}

func (a *App) CheckForUpdates() *UpdateInfo {
	if a.updater != nil {
		info, err := a.updater.CheckForUpdates()
		if err != nil {
			fmt.Printf("[update] Check failed: %v\n", err)
			return &UpdateInfo{Available: false}
		}
		return info
	}
	return &UpdateInfo{Available: false}
}

func (a *App) DownloadUpdate() *UpdateInfo {
	if a.updater != nil {
		info, err := a.updater.CheckForUpdates()
		if err != nil || !info.Available {
			return &UpdateInfo{Available: false}
		}

		updateFile, err := a.updater.DownloadUpdate(info.Url, func(progress int) {
			runtime.EventsEmit(a.ctx, "update-progress", progress)
		})

		if err != nil {
			fmt.Printf("[update] Download failed: %v\n", err)
			return &UpdateInfo{Available: false}
		}

		runtime.EventsEmit(a.ctx, "update-downloaded", updateFile)

		return &UpdateInfo{
			Available: true,
			Version:   info.Version,
			Url:       updateFile,
		}
	}
	return &UpdateInfo{Available: false}
}

func (a *App) InstallUpdate(updateFile string) error {
	if a.updater != nil {
		return a.updater.InstallUpdate(updateFile)
	}
	return fmt.Errorf("updater not initialized")
}

type ScreenSource struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`
}

func (a *App) GetScreenSources() ([]ScreenSource, error) {
	return []ScreenSource{
		{Id: "screen:0", Name: "整个屏幕"},
	}, nil
}

func (a *App) StartScreenshot() {
	go func() {
		runtime.WindowMinimise(a.ctx)
		time.Sleep(500 * time.Millisecond)

		if a.screenshot != nil {
			dataUrl, display, err := a.screenshot.CaptureFullScreen()
			if err != nil {
				fmt.Printf("[screenshot] Capture failed: %v\n", err)
				runtime.WindowShow(a.ctx)
				return
			}

			displayJson, _ := json.Marshal(display)
			runtime.WindowShow(a.ctx)
			time.Sleep(100 * time.Millisecond)
			runtime.EventsEmit(a.ctx, "screenshot-data", dataUrl, string(displayJson))
		}
	}()
}

func (a *App) CancelScreenshot() {
	runtime.EventsEmit(a.ctx, "screenshot-cancel")
	runtime.WindowShow(a.ctx)
}

func (a *App) CompleteScreenshot(dataUrl string, boundsJson string) {
	runtime.EventsEmit(a.ctx, "screenshot-taken", dataUrl, boundsJson)
	runtime.WindowShow(a.ctx)
}

func (a *App) SaveScreenshot(data []byte, boundsJson string) *FileDialogResult {
	home, _ := os.UserHomeDir()
	fileName := fmt.Sprintf("screenshot_%s.png", time.Now().Format("20060102_150405"))
	targetDir := filepath.Join(home, "Downloads")
	os.MkdirAll(targetDir, 0755)
	filePath := filepath.Join(targetDir, fileName)
	os.WriteFile(filePath, data, 0644)
	return &FileDialogResult{Canceled: false, FilePath: filePath}
}

func (a *App) GetPlatform() string {
	return runtime.Environment(a.ctx).Platform
}
