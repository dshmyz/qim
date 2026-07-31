package handler

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/gin-gonic/gin"
)

var cliFileIDRe = regexp.MustCompile(`/files/(\d+)/download`)

// CLIVersion 返回 CLI 最新版本号和各平台 SHA256 校验值。
// @Summary CLI 版本查询
// @Tags CLI
// @Produce json
// @Success 200 {object} response.Response{data=object}
// @Router /api/v1/cli/version [get]
func CLIVersion(c *gin.Context) {
	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())
	goos := c.Query("os")
	goarch := c.Query("arch")
	if goos != "" || goarch != "" {
		if goos == "" {
			goos = runtime.GOOS
		}
		if goarch == "" {
			goarch = runtime.GOARCH
		}
		if !isPlatformAllowed(goos, goarch) {
			response.BadRequest(c, "不支持的平台")
			return
		}
		v, err := svc.GetLatestCLI(goos, goarch)
		if err != nil {
			response.Success(c, gin.H{"version": "unknown", "sha256": nil})
			return
		}
		response.Success(c, gin.H{"version": v.Version, "sha256": map[string]string{cliBinaryName(goos, goarch): v.Sha256}})
		return
	}

	ver, sha256Map, err := svc.GetLatestCLIVersion()
	if err != nil {
		response.InternalServerError(c, "查询版本失败")
		return
	}
	if ver == "" {
		response.Success(c, gin.H{"version": "unknown", "sha256": nil})
		return
	}
	response.Success(c, gin.H{"version": ver, "sha256": sha256Map})
}

func cliBinaryName(goos, goarch string) string {
	name := fmt.Sprintf("qim-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// CLIDownload 下载预编译 CLI 二进制。
// @Summary CLI 二进制下载
// @Tags CLI
// @Produce octet-stream
// @Param os query string false "目标操作系统（默认按请求检测）"
// @Param arch query string false "目标架构（默认按请求检测）"
// @Success 200 {string} binary
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/cli/download [get]
func CLIDownload(c *gin.Context) {
	svc := service.NewVersionService(database.GetDB(), versionStorageAccessor())

	goos := c.Query("os")
	goarch := c.Query("arch")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	// 安全校验：只允许合法的 os/arch 值（防止路径穿越）
	if !isPlatformAllowed(goos, goarch) {
		response.BadRequest(c, "不支持的平台")
		return
	}

	v, err := svc.GetLatestCLI(goos, goarch)
	if err != nil {
		response.NotFound(c, "当前平台无可用二进制")
		return
	}

	binaryName := cliBinaryName(goos, goarch)

	c.Header("X-Qim-Version", v.Version)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, binaryName))

	// 从 DownloadURL 中提取文件 ID，直接用存储层流式返回
	if matches := cliFileIDRe.FindStringSubmatch(v.DownloadURL); len(matches) == 2 {
		fileID, _ := strconv.ParseUint(matches[1], 10, 32)
		serveFileByID(c, uint(fileID))
		return
	}

	// 兜底：外部 URL 走302 重定向
	c.Redirect(302, v.DownloadURL)
}

// serveFileByID 从存储层直接流式返回文件。
// Content-Type 固定为 application/octet-stream（二进制下载，不希望浏览器尝试预览）。
// Content-Disposition 由调用方提前设置。
func serveFileByID(c *gin.Context, fileID uint) {
	db := database.GetDB()
	var file model.File
	if err := db.First(&file, fileID).Error; err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	// CLI 二进制较大，用120秒超时
	streamFileFromStorage(c, file.StoragePath, file.Size, 120*time.Second)
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
