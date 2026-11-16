// Package capture 提供"微信式截图"的底层能力：
// 抓取屏幕像素，按用户拖出的选区裁剪，产出可进剪贴板的 PNG。
//
// macOS 上抓屏走系统 screencapture 命令（-x 静音，-R 指定区域），
// 不用 cgo。系统会要求"屏幕录制"权限（系统设置 → 隐私与安全性 →
// 屏幕录制），微信/QQ/微信输入法做截图同样绕不开它。
package capture

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
)

// shotToFile 调 screencapture 抓区域到临时文件，返回 PNG 字节。
// screencapture 不支持写 stdout（实测输出 0 字节），只能落盘再读。
func shotToFile(bounds image.Rectangle) ([]byte, error) {
	tmp, err := os.CreateTemp("", "qim-shot-*.png")
	if err != nil {
		return nil, err
	}
	path := tmp.Name()
	tmp.Close()
	defer os.Remove(path)

	cmd := exec.Command("screencapture", "-x",
		fmt.Sprintf("-R%d,%d,%d,%d", bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()),
		"-t", "png", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("screencapture: %w: %s", err, out)
	}
	return os.ReadFile(path)
}

// Shot 全屏截取指定显示器，返回其像素图像。
// bounds 是显示器在全局坐标系里的位置（mygo Display.Bounds）。
//
// 主屏（坐标原点）优先走 CoreGraphics 的进程内抓屏，几十毫秒；
// 副屏或指定区域回退到 screencapture 子进程（约 0.7 秒）。
func Shot(bounds image.Rectangle) (image.Image, error) {
	if bounds.Min.X == 0 && bounds.Min.Y == 0 {
		if img, err := shotCG(); err == nil {
			return img, nil
		}
	}
	out, err := shotToFile(bounds)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("解码截图: %w", err)
	}
	return img, nil
}

// ShotPNG 直接返回全屏 PNG 字节，给覆盖层窗口当底图用。
func ShotPNG(bounds image.Rectangle) ([]byte, error) {
	return shotToFile(bounds)
}

// Crop 从全屏截图里裁出选区。sel 用 DIP 坐标（和覆盖层 UI 一致），
// screenDIPs 是该显示器的逻辑尺寸——macOS 截图是 2x Retina，
// 按 DIP/像素比例缩放后再裁，保证裁出来的就是用户看到的那块。
func Crop(full image.Image, sel image.Rectangle, screenDIPs image.Rectangle) image.Image {
	sx := float64(full.Bounds().Dx()) / float64(screenDIPs.Dx())
	sy := float64(full.Bounds().Dy()) / float64(screenDIPs.Dy())
	r := image.Rect(
		int(float64(sel.Min.X-screenDIPs.Min.X)*sx),
		int(float64(sel.Min.Y-screenDIPs.Min.Y)*sy),
		int(float64(sel.Max.X-screenDIPs.Min.X)*sx),
		int(float64(sel.Max.Y-screenDIPs.Min.Y)*sy),
	)
	r = r.Intersect(full.Bounds())
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(dst, dst.Bounds(), full, r.Min, draw.Src)
	return dst
}

// PNG 把图像编码成 PNG 字节，进剪贴板 / 存盘都用它。
func PNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
