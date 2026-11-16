//go:build !darwin

package capture

import (
	"errors"
	"image"
)

// shotCG 只在 macOS 上有实现，其它平台一律回退 screencapture/子进程路径。
func shotCG() (image.Image, error) {
	return nil, errors.New("CoreGraphics 抓屏仅 macOS 可用")
}
