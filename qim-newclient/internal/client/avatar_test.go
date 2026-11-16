package client

// 成员真实头像的加载链路验证：用 httptest 起一个返回合法 PNG 的本地服务，
// 把 client 指向它，再走 memberAvatar 复用的 imageState → Download → DecodeImage
// 链路，确认头像位图能真正落进缓存（首帧加载中、随后非空），而不是永远回退首字。

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func makeTestPNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{0x33, 0x85, 0xff, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestMemberAvatarLoads(t *testing.T) {
	pngBytes := makeTestPNG(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	a := &app{
		loggedIn: true,
		client:   qapi.New(srv.URL),
		ctx:      context.Background(),
	}
	att := qapi.Attachment{URL: "/av.png"}
	key := "url:" + a.client.ResolveURL(att.URL)

	// 首帧：缓存为空，应进入加载中、返回 nil（渲染此时回退首字）。
	if src, _, loading := a.imageState(att, ""); src != nil || !loading {
		t.Fatalf("首帧状态不符：src=%v loading=%v，期望 src=nil loading=true", src, loading)
	}

	// 等异步加载把位图写进缓存。
	deadline := time.Now().Add(3 * time.Second)
	var got ui.ImageSource
	for time.Now().Before(deadline) {
		a.media.mu.Lock()
		v, errTxt := a.media.images[key], a.media.errs[key]
		a.media.mu.Unlock()
		if errTxt != "" {
			t.Fatalf("头像加载报错: %s", errTxt)
		}
		if v != nil {
			got = v
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got == nil {
		t.Fatal("头像 3s 内未加载完成，缓存里仍是 nil")
	}

	// 第二次取：src 应非空，memberAvatar 因此画图片而非首字。
	if src, _, _ := a.imageState(att, ""); src == nil {
		t.Fatal("加载完成后 imageState 仍返回 nil，memberAvatar 会错误回退首字")
	}
}
