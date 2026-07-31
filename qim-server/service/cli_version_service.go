package service

import (
	"os"
	"path/filepath"
	"strings"
)

// CLIVersionService 管理 CLI 二进制的版本与校验值读取。
// 数据来源为 data/cli/ 目录下的文件，不依赖数据库。
//
// Deprecated: CLI 版本已迁移到 VersionService (app_type="cli")，通过数据库管理。
// 此服务仅保留用于向后兼容，新部署请使用 VersionService。
type CLIVersionService struct {
	binaryDir string
}

// NewCLIVersionService 创建 CLI 版本服务。
func NewCLIVersionService(binaryDir string) *CLIVersionService {
	return &CLIVersionService{binaryDir: binaryDir}
}

// GetVersion 读取 version.txt 返回版本号，文件不存在时返回 "unknown"。
func (s *CLIVersionService) GetVersion() string {
	b, err := os.ReadFile(filepath.Join(s.binaryDir, "version.txt"))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}

// GetSha256Map 读取 data/cli/ 下所有 qim-{os}-{arch}.sha256 旁路文件。
// 返回 map["qim-darwin-arm64"] = "abc123..."。
// 管理员可通过 `shasum -a 256 qim-darwin-arm64 > qim-darwin-arm64.sha256` 生成。
// 无旁路文件时返回 nil（客户端跳过校验，前向兼容旧部署）。
func (s *CLIVersionService) GetSha256Map() map[string]string {
	entries, err := os.ReadDir(s.binaryDir)
	if err != nil {
		return nil
	}
	m := make(map[string]string)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sha256") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.binaryDir, name))
		if err != nil {
			continue
		}
		// .sha256 文件内容可能是 "hash  filename" 或纯 hash
		line := strings.TrimSpace(string(b))
		if idx := strings.IndexByte(line, ' '); idx > 0 {
			line = line[:idx]
		}
		binaryName := strings.TrimSuffix(name, ".sha256")
		m[binaryName] = line
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// GetBinaryPath 返回指定平台的二进制文件完整路径。
func (s *CLIVersionService) GetBinaryPath(binaryName string) string {
	return filepath.Join(s.binaryDir, binaryName)
}

// BinaryExists 检查二进制文件是否存在。
func (s *CLIVersionService) BinaryExists(binaryName string) bool {
	_, err := os.Stat(s.GetBinaryPath(binaryName))
	return !os.IsNotExist(err)
}
