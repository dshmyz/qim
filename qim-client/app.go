package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
	runtime.EventsEmit(a.ctx, "tray-flash", enabled)
}

type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Url       string `json:"url"`
}

func (a *App) CheckForUpdates() *UpdateInfo {
	return &UpdateInfo{Available: false}
}

func (a *App) DownloadUpdate() *UpdateInfo {
	return &UpdateInfo{Available: false}
}

type ScreenSource struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`
}

func (a *App) GetScreenSources() ([]ScreenSource, error) {
	runtime.EventsEmit(a.ctx, "screen-share-requested", true)
	return nil, nil
}

func (a *App) StartScreenshot() {
	runtime.WindowMinimise(a.ctx)
	time.Sleep(300 * time.Millisecond)
	runtime.EventsEmit(a.ctx, "screenshot-requested", true)
}

func (a *App) GetPlatform() string {
	return runtime.Environment(a.ctx).Platform
}
