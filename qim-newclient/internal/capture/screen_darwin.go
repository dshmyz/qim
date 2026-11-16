//go:build darwin

// 用 CoreGraphics 直接取屏幕像素，替代 `screencapture` 子进程。
//
// 为什么值得：调 screencapture 抓全屏要 0.6~0.8 秒（进程启动 + 写文件 +
// PNG 编码），按下快捷键后要等这么久才看到覆盖层，完全不像微信那种
// 一按就有的手感；CGDisplayCreateImage 是进程内调用 + 内存位图，
// 量级在几十毫秒。
//
// 走 purego 的 dlopen/dlsym，不引入 cgo，保持 mygo 的"纯 Go、免 cgo"前提。
package capture

import (
	"fmt"
	"image"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var cgFns struct {
	once sync.Once
	err  error

	mainDisplayID        func() uint32
	displayCreateImage   func(display uint32) uintptr
	imageGetWidth        func(img uintptr) uintptr
	imageGetHeight       func(img uintptr) uintptr
	imageGetRowBytes     func(img uintptr) uintptr
	imageGetBitsPerPixel func(img uintptr) uintptr
	imageGetDataProvider func(img uintptr) uintptr
	dataProviderCopyData func(provider uintptr) uintptr
	dataGetBytePtr       func(data uintptr) unsafe.Pointer
	dataGetLength        func(data uintptr) int64
	release              func(cf uintptr)
}

func loadCG() error {
	cgFns.once.Do(func() {
		cg, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
			purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			cgFns.err = fmt.Errorf("加载 CoreGraphics: %w", err)
			return
		}
		cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
			purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			cgFns.err = fmt.Errorf("加载 CoreFoundation: %w", err)
			return
		}
		purego.RegisterLibFunc(&cgFns.mainDisplayID, cg, "CGMainDisplayID")
		purego.RegisterLibFunc(&cgFns.displayCreateImage, cg, "CGDisplayCreateImage")
		purego.RegisterLibFunc(&cgFns.imageGetWidth, cg, "CGImageGetWidth")
		purego.RegisterLibFunc(&cgFns.imageGetHeight, cg, "CGImageGetHeight")
		purego.RegisterLibFunc(&cgFns.imageGetRowBytes, cg, "CGImageGetBytesPerRow")
		purego.RegisterLibFunc(&cgFns.imageGetBitsPerPixel, cg, "CGImageGetBitsPerPixel")
		purego.RegisterLibFunc(&cgFns.imageGetDataProvider, cg, "CGImageGetDataProvider")
		purego.RegisterLibFunc(&cgFns.dataProviderCopyData, cg, "CGDataProviderCopyData")
		purego.RegisterLibFunc(&cgFns.dataGetBytePtr, cf, "CFDataGetBytePtr")
		purego.RegisterLibFunc(&cgFns.dataGetLength, cf, "CFDataGetLength")
		purego.RegisterLibFunc(&cgFns.release, cf, "CFRelease")

		// 预热：purego 的首次调用要解析符号、走一遍 lazy binding，
		// 单独测过 CGMainDisplayID 首调就有 ~200ms，摊到第一次截图里
		// 会变成"按了快捷键半天没反应"。
		_ = cgFns.mainDisplayID()
	})
	return cgFns.err
}

// shotCG 抓主显示器，返回 RGBA 图像。坐标系原点在左上，和 DIP 坐标一致。
func shotCG() (image.Image, error) {
	if err := loadCG(); err != nil {
		return nil, err
	}
	cgImg := cgFns.displayCreateImage(cgFns.mainDisplayID())
	if cgImg == 0 {
		// 十有八九是没给「屏幕录制」权限
		return nil, fmt.Errorf("CGDisplayCreateImage 返回空（可能缺少屏幕录制权限）")
	}
	defer cgFns.release(cgImg)

	w := int(cgFns.imageGetWidth(cgImg))
	h := int(cgFns.imageGetHeight(cgImg))
	rowBytes := int(cgFns.imageGetRowBytes(cgImg))
	bpp := int(cgFns.imageGetBitsPerPixel(cgImg))
	if w == 0 || h == 0 || bpp != 32 {
		return nil, fmt.Errorf("不支持的位图格式: %dx%d %dbpp", w, h, bpp)
	}

	provider := cgFns.imageGetDataProvider(cgImg)
	if provider == 0 {
		return nil, fmt.Errorf("CGImageGetDataProvider 返回空")
	}
	cfData := cgFns.dataProviderCopyData(provider)
	if cfData == 0 {
		return nil, fmt.Errorf("CGDataProviderCopyData 返回空")
	}
	defer cgFns.release(cfData)

	n := int(cgFns.dataGetLength(cfData))
	ptr := cgFns.dataGetBytePtr(cfData)
	if ptr == nil || n < rowBytes*h {
		return nil, fmt.Errorf("像素数据不完整: %d < %d", n, rowBytes*h)
	}
	src := unsafe.Slice((*byte)(ptr), rowBytes*h)

	// CGDisplayCreateImage 给的是 kCGImageAlphaPremultipliedFirst +
	// kCGBitmapByteOrder32Little，内存里就是 BGRA，逐行换字节序。
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		line := src[y*rowBytes : y*rowBytes+w*4]
		o := dst.Pix[y*dst.Stride:]
		for x := 0; x < w; x++ {
			i := x * 4
			o[i] = line[i+2]   // R
			o[i+1] = line[i+1] // G
			o[i+2] = line[i]   // B
			o[i+3] = 0xff      // 屏幕截图没有透明，直接不透明
		}
	}
	return dst, nil
}
