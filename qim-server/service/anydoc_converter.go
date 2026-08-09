package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

// anydocBinaryEnv 显式指定 anydoc 可执行文件路径的环境变量。
// 不设置则回退到 PATH 探测（exec.LookPath("anydoc")）。
const anydocBinaryEnv = "QIM_ANYDOC_BINARY"

// anydocCallTimeout 单次 anydoc 调用的超时上限，防止机器上二进制挂起拖住解析。
const anydocCallTimeout = 30 * time.Second

// errAnydocUnsupported 表示 anydoc 明确报告「无法转换该文档」
// （扫描型/图片型 PDF、加密文档、损坏文件等）。调用方据此区分
// 「不可转换」与「工具本身故障」，决定是降级回退还是原样报错。
var errAnydocUnsupported = errors.New("anydoc 无法转换该文档")

// AnydocConverter 封装对 anydoc CLI 的进程外调用。
//
// anydoc（https://github.com/firecrawl/anydoc）是 Firecrawl 提供的自包含本地
// 文档转换器：把 Word/PowerPoint/Excel/OpenDocument/RTF/EPUB/CSV/PDF 等 14 种
// 格式统一转成 GFM Markdown。它没有 Go binding，这里通过 exec 调其预编译二进制，
// 不做任何外部服务/网络依赖，也不需要 LibreOffice。设计对齐项目「Linux 兼容 +
// 不引入 LibreOffice/中文字体重依赖」的硬约束。
//
// 本转换器是可选的增强后端：二进制缺失或调用失败时上层应回退到既有解析器，
// 绝不因 anydoc 不可用而拖垮主路径。
type AnydocConverter struct {
	// bin 探测到的 anydoc 可执行文件路径；空表示未找到（不可用）。
	bin string

	// once 保证 lookPath 只探测一次，避免每次解析都走一次 PATH 查找。
	once sync.Once
}

// NewAnydocConverter 构造转换器并探测二进制。探测逻辑幂等且只跑一次：
// 显式环境变量 anydocBinaryEnv 优先，否则 PATH 里的 anydoc。找不到时
// Available() 返回 false，Convert() 直接返回错误（上层应降级）。
func NewAnydocConverter() *AnydocConverter {
	c := &AnydocConverter{}
	c.once.Do(c.detect)
	return c
}

// detect 惰性探测一次二进制路径。
func (c *AnydocConverter) detect() {
	if p := os.Getenv(anydocBinaryEnv); p != "" {
		c.bin = p
		return
	}
	if p, err := exec.LookPath("anydoc"); err == nil {
		c.bin = p
	}
}

// binPath 惰性确保已探测，返回二进制路径（可能为空）。幂等。
func (c *AnydocConverter) binPath() string {
	c.once.Do(c.detect)
	return c.bin
}

// Available 报告 anydoc 二进制是否可用。仅当二进制存在时后续才尝试 anydoc。
func (c *AnydocConverter) Available() bool {
	return c.binPath() != ""
}

// Convert 调用 anydoc 把 filePath 指向的文档转成 Markdown。
// 错误语义：
//   - 二进制缺失 / 非退出码 0 → 返回普通错误；
//   - anydoc 自身报告无法转换（退出码 1，如扫描 PDF/加密/损坏）→ 返回
//     errAnydocUnsupported，调用方据此降级回退原生解析或维持「不支持」。
func (c *AnydocConverter) Convert(filePath string) (string, error) {
	bin := c.binPath()
	if bin == "" {
		return "", errors.New("anydoc 不可用：未找到二进制（PATH 或 " + anydocBinaryEnv + "）")
	}

	ctx, cancel := context.WithTimeout(context.Background(), anydocCallTimeout)
	defer cancel()

	// 只捕获 stdout（Markdown）；stderr 承载诊断信息，仅在失败时记日志。
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, filePath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitErr := new(exec.ExitError)
	switch {
	case err == nil:
		// success
		return stdout.String(), nil
	case errors.As(err, &exitErr) && exitErr.ExitCode() == 1:
		// anydoc: 无可读可写的 Markdown 输出（扫描/加密/损坏等）
		logger.WithModule("AnydocConverter").Debug("anydoc 无法转换文档",
			"file", filePath, "stderr", stderr.String())
		return "", errAnydocUnsupported
	default:
		logger.WithModule("AnydocConverter").Warn("anydoc 调用失败",
			"file", filePath, "error", err, "stderr", stderr.String())
		return "", err
	}
}
