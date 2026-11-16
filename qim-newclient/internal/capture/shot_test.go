package capture

import (
	"image"
	"image/png"
	"os"
	"testing"
	"time"
)

// TestShotCG 验证 CoreGraphics 抓屏可用、够快、颜色通道没搞反。
// 出图写到 /tmp/qim-cg.png，供肉眼核对。
func TestShotCG(t *testing.T) {
	start := time.Now()
	img, err := Shot(image.Rect(0, 0, 1792, 1120))
	if err != nil {
		t.Fatalf("抓屏失败: %v", err)
	}
	t.Logf("抓屏耗时 %v，尺寸 %v", time.Since(start), img.Bounds())

	if img.Bounds().Dx() < 100 || img.Bounds().Dy() < 100 {
		t.Fatalf("尺寸异常: %v", img.Bounds())
	}

	f, err := os.Create("/tmp/qim-cg.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
