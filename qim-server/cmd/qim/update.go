package main

import (
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var insecure bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "从服务器下载最新版 CLI 并替换",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := loadConfig()
			if err != nil || cfg.ServerURL == "" {
				die("请先配置: qim config set --server URL --token T")
			}

			// 1. 查询最新版本
			verBody, err := httpGetRaw(fmt.Sprintf("%s/api/v1/cli/version?os=%s&arch=%s", cfg.ServerURL, runtime.GOOS, runtime.GOARCH))
			if err != nil {
				die("查询版本失败: %v", err)
			}
			var verResp struct {
				Data struct {
					Version string            `json:"version"`
					Sha256  map[string]string `json:"sha256"`
				} `json:"data"`
			}
			if err := json.Unmarshal(verBody, &verResp); err != nil {
				die("解析版本响应失败: %v", err)
			}
			latest := verResp.Data.Version
			if latest == "" {
				die("服务端返回的版本号为空")
			}

			if latest == version && version != "dev" {
				fmt.Printf("已是最新版: %s\n", version)
				return
			}
			if latest == "unknown" {
				die("服务端未配置 CLI 二进制，请联系管理员上传到 data/cli/ 目录")
			}

			// 2. 下载新二进制（大文件用无超时 client，避免 30s 限制）
			binaryName := fmt.Sprintf("qim-%s-%s", runtime.GOOS, runtime.GOARCH)
			if runtime.GOOS == "windows" {
				binaryName += ".exe"
			}
			fmt.Printf("当前: %s → 最新: %s\n正在下载...\n", version, latest)
			dlURL := fmt.Sprintf("%s/api/v1/cli/download?os=%s&arch=%s", cfg.ServerURL, runtime.GOOS, runtime.GOARCH)
			binary, err := downloadBinary(dlURL)
			if err != nil {
				die("下载失败: %v", err)
			}

			// 3. SHA256 校验（fail-closed：无 hash 则拒绝更新，除非 --insecure）
			expectedHash, ok := verResp.Data.Sha256[binaryName]
			if !ok || expectedHash == "" {
				if insecure {
					fmt.Fprintln(os.Stderr, "⚠️  服务端未提供 SHA256，已通过 --insecure 跳过校验（不推荐）")
				} else {
					die("服务端未提供 SHA256 校验值，拒绝更新（如确认安全可加 --insecure 跳过）")
				}
			} else {
				actual := sha256.Sum256(binary)
				actualHex := hex.EncodeToString(actual[:])
				if !strings.EqualFold(actualHex, expectedHash) {
					die("SHA256 校验失败: 期望 %s 实际 %s（文件可能被篡改）", expectedHash, actualHex)
				}
				fmt.Fprintln(os.Stderr, "🔒 SHA256 校验通过")
			}

			// 4. 找到当前可执行文件路径
			selfPath, err := os.Executable()
			if err != nil {
				die("获取当前程序路径失败: %v", err)
			}
			selfPath, _ = filepath.EvalSymlinks(selfPath)

			// 5. 写入临时文件
			tmpPath := selfPath + ".tmp"
			if err := os.WriteFile(tmpPath, binary, 0o755); err != nil {
				die("写入临时文件失败: %v", err)
			}

			// 6. 替换并验证（失败自动回滚到原二进制）
			rolledBack, err := replaceBinary(selfPath, tmpPath, func(p string) error {
				out, runErr := exec.Command(p, "version").Output()
				if runErr != nil {
					return fmt.Errorf("二进制无法执行: %w", runErr)
				}
				fmt.Printf("✅ 已更新: %s\n", strings.TrimSpace(string(out)))
				return nil
			})
			if err != nil {
				if !rolledBack {
					os.Remove(tmpPath)
				}
				die("%v", err)
			}
		},
	}
	cmd.Flags().BoolVar(&insecure, "insecure", false, "跳过 SHA256 校验（不推荐）")
	return cmd
}

// maxBinarySize 限制下载二进制最大 200MB，防止异常响应耗尽内存。
const maxBinarySize = 200 * 1024 * 1024

// downloadBinary 用无超时 client 下载二进制（大文件可能超过 30s）。
func downloadBinary(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBinarySize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBinarySize {
		return nil, fmt.Errorf("二进制超过 %dMB 上限，下载中止", maxBinarySize/(1024*1024))
	}
	return data, nil
}

// replaceBinary 用 tmpPath 原子替换 selfPath，verify 失败时自动回滚到原文件。
// 流程：备份原文件 → 替换 → 验证 → 失败回滚 / 成功清理备份。
// Windows 上正在运行的 exe 无法直接覆盖，但可以先 rename 再写入新文件。
// 返回 rolledBack=true 表示 verify 失败已恢复原文件。
func replaceBinary(selfPath, tmpPath string, verify func(string) error) (bool, error) {
	backupPath := selfPath + ".bak"
	// 清理上次中断遗留的备份
	_ = os.Remove(backupPath)

	// 1. 备份原二进制（rename 在 Unix/Windows 上对运行中的 exe 均可执行）
	if err := os.Rename(selfPath, backupPath); err != nil {
		return false, fmt.Errorf("备份原二进制失败: %w", err)
	}
	// 2. 替换为新二进制
	if err := os.Rename(tmpPath, selfPath); err != nil {
		_ = os.Rename(backupPath, selfPath) // 回滚
		return false, fmt.Errorf("替换二进制失败: %w（Windows 需关闭所有 qim 进程后重试）", err)
	}
	// 3. 验证新二进制可用
	if verify != nil {
		if err := verify(selfPath); err != nil {
			// 验证失败：移除损坏的新文件，恢复备份
			_ = os.Remove(selfPath)
			_ = os.Rename(backupPath, selfPath)
			return true, fmt.Errorf("更新后验证失败，已回滚到原版本: %w", err)
		}
	}
	// 4. 成功，清理备份
	_ = os.Remove(backupPath)
	return false, nil
}
