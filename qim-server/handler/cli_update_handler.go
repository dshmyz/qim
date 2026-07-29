package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// cliBinaryDir 存放预编译 CLI 二进制的目录。
// 文件命名: qim-{os}-{arch}[.exe]，如 qim-darwin-arm64、qim-linux-amd64.exe
// 可选: version.txt 存放当前版本号。
const cliBinaryDir = "data/cli"

// CLIVersion 返回 CLI 最新版本号和平台信息。
// GET /api/v1/cli/version
func CLIVersion(c *gin.Context) {
	version := readCLIVersion()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"version": version,
		},
	})
}

// CLIDownload 下载预编译 CLI 二进制。
// GET /api/v1/cli/download[?os=darwin&arch=arm64]
// 不传 os/arch 则根据请求自动检测。
func CLIDownload(c *gin.Context) {
	goos := c.Query("os")
	goarch := c.Query("arch")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	// 安全校验：只允许合法的 os/arch 值
	if !isPlatformAllowed(goos, goarch) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "不支持的平台"})
		return
	}

	binaryName := fmt.Sprintf("qim-%s-%s", goos, goarch)
	if goos == "windows" {
		binaryName += ".exe"
	}

	binaryPath := filepath.Join(cliBinaryDir, binaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"code": 1004, "message": "当前平台无可用二进制", "data": gin.H{
			"os": goos, "arch": goarch, "expected": binaryName,
		}})
		return
	}

	version := readCLIVersion()
	c.Header("X-Qim-Version", version)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, binaryName))
	c.File(binaryPath)
}

// readCLIVersion 从 data/cli/version.txt 读取版本号。
func readCLIVersion() string {
	b, err := os.ReadFile(filepath.Join(cliBinaryDir, "version.txt"))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}

// isPlatformAllowed 校验平台参数（防止路径穿越）。
func isPlatformAllowed(goos, goarch string) bool {
	allowed := map[string][]string{
		"darwin":  {"amd64", "arm64"},
		"linux":   {"amd64", "arm64"},
		"windows": {"amd64", "arm64"},
	}
	archs, ok := allowed[goos]
	if !ok {
		return false
	}
	for _, a := range archs {
		if a == goarch {
			return true
		}
	}
	return false
}
