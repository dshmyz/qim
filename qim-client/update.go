package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type UpdateManager struct {
	currentVersion string
	updateServer   string
}

type UpdateResponse struct {
	Version string `json:"version"`
	Url     string `json:"url"`
	Name    string `json:"name"`
	Notes   string `json:"notes"`
}

func NewUpdateManager() *UpdateManager {
	return &UpdateManager{
		currentVersion: "1.0.0",
		updateServer:   "http://localhost:8080/api/v1/updates",
	}
}

func (u *UpdateManager) getPlatformUpdateURL() string {
	switch runtime.GOOS {
	case "windows":
		return u.getWindowsUpdateURL()
	case "darwin":
		return fmt.Sprintf("%s/darwin/", u.updateServer)
	default:
		return fmt.Sprintf("%s/%s/", u.updateServer, runtime.GOOS)
	}
}

func (u *UpdateManager) getWindowsUpdateURL() string {
	major, _, _ := u.getWindowsVersion()
	if major < 10 {
		return fmt.Sprintf("%s/win7/", u.updateServer)
	}
	return fmt.Sprintf("%s/win10/", u.updateServer)
}

func (u *UpdateManager) getWindowsVersion() (int, int, int) {
	if runtime.GOOS != "windows" {
		return 0, 0, 0
	}

	cmd := exec.Command("cmd", "/C", "ver")
	output, err := cmd.Output()
	if err != nil {
		return 10, 0, 0
	}

	verStr := string(output)
	if strings.Contains(verStr, "Version 10") {
		return 10, 0, 0
	} else if strings.Contains(verStr, "Version 6.1") {
		return 6, 1, 0
	} else if strings.Contains(verStr, "Version 6.3") {
		return 6, 3, 0
	}

	return 10, 0, 0
}

func (u *UpdateManager) CheckForUpdates() (*UpdateInfo, error) {
	updateURL := u.getPlatformUpdateURL()
	if updateURL == "" {
		return &UpdateInfo{Available: false}, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(updateURL)
	if err != nil {
		return nil, fmt.Errorf("check update failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &UpdateInfo{Available: false}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read update response failed: %w", err)
	}

	var updateResp UpdateResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return nil, fmt.Errorf("parse update response failed: %w", err)
	}

	if updateResp.Version == "" || updateResp.Version == u.currentVersion {
		return &UpdateInfo{Available: false}, nil
	}

	return &UpdateInfo{
		Available: true,
		Version:   updateResp.Version,
		Url:       updateResp.Url,
	}, nil
}

func (u *UpdateManager) DownloadUpdate(url string, progressCallback func(progress int)) (string, error) {
	if url == "" {
		return "", fmt.Errorf("update URL is empty")
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download update failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	tmpDir := os.TempDir()
	var fileName string
	if runtime.GOOS == "windows" {
		fileName = "qim-update.exe"
	} else if runtime.GOOS == "darwin" {
		fileName = "qim-update.dmg"
	} else {
		fileName = "qim-update.tar.gz"
	}

	tmpFile := filepath.Join(tmpDir, fileName)
	out, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("create temp file failed: %w", err)
	}
	defer out.Close()

	totalSize := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return "", writeErr
			}
			downloaded += int64(n)

			if totalSize > 0 && progressCallback != nil {
				progress := int(float64(downloaded) / float64(totalSize) * 100)
				progressCallback(progress)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return tmpFile, nil
}

func (u *UpdateManager) InstallUpdate(updateFile string) error {
	if _, err := os.Stat(updateFile); err != nil {
		return fmt.Errorf("update file not found: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command(updateFile)
		cmd.Start()
		os.Exit(0)
	case "darwin":
		cmd := exec.Command("open", updateFile)
		cmd.Start()
		os.Exit(0)
	default:
		cmd := exec.Command("tar", "-xzf", updateFile)
		cmd.Start()
		os.Exit(0)
	}

	return nil
}
