package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type ScreenshotManager struct{}

func NewScreenshotManager() *ScreenshotManager {
	return &ScreenshotManager{}
}

type DisplayInfo struct {
	Id     int `json:"id"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (s *ScreenshotManager) GetDisplayInfo() *DisplayInfo {
	if runtime.GOOS == "darwin" {
		output, err := exec.Command("osascript", "-e",
			`tell application "System Events" to get {word 1 of (do shell script "system_profiler SPDisplaysDataType | grep Resolution"), word 3 of (do shell script "system_profiler SPDisplaysDataType | grep Resolution")}`).CombinedOutput()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(output)), ",")
			if len(parts) == 2 {
				w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					return &DisplayInfo{Id: 0, Width: w, Height: h}
				}
			}
		}
	}

	return &DisplayInfo{
		Id:     0,
		Width:  1920,
		Height: 1080,
	}
}

func (s *ScreenshotManager) CaptureFullScreen() (string, *DisplayInfo, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "qim-screenshot-full.png")

	display := s.GetDisplayInfo()

	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("screencapture", "-x", tmpFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", nil, fmt.Errorf("screencapture failed: %v, output: %s", err, string(output))
		}

		data, err := os.ReadFile(tmpFile)
		if err != nil {
			return "", nil, fmt.Errorf("read screenshot file failed: %v", err)
		}
		os.Remove(tmpFile)

		if len(data) == 0 {
			return "", nil, fmt.Errorf("screenshot data is empty")
		}

		base64Str := base64.StdEncoding.EncodeToString(data)
		return fmt.Sprintf("data:image/png;base64,%s", base64Str), display, nil

	default:
		return "", nil, fmt.Errorf("unsupported platform for full screen capture: %s", runtime.GOOS)
	}
}