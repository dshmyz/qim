package service

import (
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"gorm.io/gorm"
)

var hashHTTPClient = &http.Client{Timeout: 10 * time.Minute}

// ComputeFileHash 根据下载 URL 计算 SHA512 和文件大小
// 支持三种来源：公开文件 API、HTTP/HTTPS 远程（含本地缓存）、本地路径
// 失败时返回具体 error，不再静默返回空值
func ComputeFileHash(db *gorm.DB, downloadURL, platform string) (sha512Hex string, size int64, err error) {
	// 1. 公开文件下载链接：从 DB 读 StoragePath
	if strings.Contains(downloadURL, "/api/v1/public/files/") && strings.HasSuffix(downloadURL, "/download") {
		return computeFromPublicFile(db, downloadURL)
	}
	// 2. HTTP/HTTPS：先尝试本地缓存，再走 HTTP 下载
	if strings.HasPrefix(downloadURL, "http://") || strings.HasPrefix(downloadURL, "https://") {
		if cached, ok := tryLocalCache(downloadURL, platform); ok {
			return cached.hash, cached.size, nil
		}
		return computeFromHTTP(downloadURL)
	}
	// 3. 本地路径
	return computeFromFile(downloadURL)
}

type hashResult struct {
	hash string
	size int64
}

// tryLocalCache 尝试在 ./uploads/updates/<platform>/ 目录找缓存文件
func tryLocalCache(downloadURL, platform string) (hashResult, bool) {
	filename := filepath.Base(downloadURL)
	localPath := filepath.Join("./uploads/updates", platform, filename)
	if _, err := os.Stat(localPath); err != nil {
		return hashResult{}, false
	}
	h, s, err := computeFromFile(localPath)
	if err != nil {
		return hashResult{}, false
	}
	return hashResult{h, s}, true
}

// computeFromHTTP 通过 HTTP 下载文件并计算哈希
func computeFromHTTP(url string) (string, int64, error) {
	resp, err := hashHTTPClient.Get(url)
	if err != nil {
		return "", 0, fmt.Errorf("下载安装包失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("下载安装包失败: HTTP %d", resp.StatusCode)
	}
	hash := sha512.New()
	size, err := io.Copy(hash, resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("计算哈希失败: %w", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

// computeFromPublicFile 从公开文件 API 的 URL 中提取文件 ID，读取 StoragePath 计算哈希
func computeFromPublicFile(db *gorm.DB, downloadURL string) (string, int64, error) {
	parts := strings.Split(downloadURL, "/")
	for i, part := range parts {
		if part == "files" && i+1 < len(parts) {
			fileID, err := strconv.ParseUint(parts[i+1], 10, 32)
			if err != nil {
				return "", 0, fmt.Errorf("无效的文件 ID: %s", parts[i+1])
			}
			var file model.File
			if err := db.First(&file, uint(fileID)).Error; err != nil {
				return "", 0, fmt.Errorf("文件记录不存在 (ID=%d): %w", fileID, err)
			}
			storagePath := file.StoragePath
			if strings.HasPrefix(storagePath, "/uploads/") {
				storagePath = "." + storagePath
			}
			return computeFromFile(storagePath)
		}
	}
	return "", 0, fmt.Errorf("无法从 URL 解析文件 ID: %s", downloadURL)
}

// computeFromFile 计算本地文件的 SHA512 和大小
func computeFromFile(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("打开文件失败 (%s): %w", filePath, err)
	}
	defer file.Close()

	hash := sha512.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("计算哈希失败 (%s): %w", filePath, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

// MustComputeFileHash 计算哈希，失败时记录日志并返回 error
// 用于 CreateVersion 等场景，调用方应根据 error 返回具体错误给用户
func MustComputeFileHash(db *gorm.DB, downloadURL, platform string) (string, int64, error) {
	hash, size, err := ComputeFileHash(db, downloadURL, platform)
	if err != nil {
		logger.WithModule("Version").Error("计算安装包哈希失败",
			"download_url", downloadURL,
			"platform", platform,
			"error", err,
		)
		return "", 0, err
	}
	return hash, size, nil
}
