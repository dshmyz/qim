package client

// P2：富媒体消息归一化、上传发送闭环、失败重试跳过已上传步骤、
// 以及图片/文件气泡在 headless 渲染里确实画得出来。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func p2ImageContent(name string) string {
	b, _ := json.Marshal(map[string]any{
		"url": "/static/uploads/2026/11/" + name, "id": 109,
		"name": name, "size": 2048, "mimeType": "image/png",
	})
	return string(b)
}

func TestP2MessageFromAPIMedia(t *testing.T) {
	content := p2ImageContent("截图.png")
	raw := fmt.Sprintf(`{"id":960,"conversation_id":106,"sender_id":8,"type":"image",
		"content":%q,"created_at":"2026-11-14T21:00:00+08:00"}`, content)
	var api qapi.Message
	if err := json.Unmarshal([]byte(raw), &api); err != nil {
		t.Fatal(err)
	}
	m := messageFromAPI(api, "7", 0)
	if m.Kind != "image" || m.Att.ID != "109" || m.Att.URL == "" || m.Att.Size != 2048 {
		t.Fatalf("图片消息没解析成附件: %+v", m)
	}
	if !strings.Contains(m.Text, "[图片]") || !strings.Contains(m.Text, "截图.png") || strings.Contains(m.Text, "static/uploads") {
		t.Fatalf("摘要不该露出协议字段: %q", m.Text)
	}

	// 文件类型走同一条解析，但 kind 不同，UI 会画文件卡片
	fileRaw := fmt.Sprintf(`{"id":961,"type":"file","content":%q,"is_recalled":false}`,
		`{"url":"/static/f/需求文档.pdf","id":110,"name":"需求文档.pdf","size":2097152}`)
	var fapi qapi.Message
	_ = json.Unmarshal([]byte(fileRaw), &fapi)
	fm := messageFromAPI(fapi, "7", 1)
	if fm.Kind != "file" || !strings.Contains(fm.Text, "[文件]") || !strings.Contains(fm.Text, "2.0 MB") {
		t.Fatalf("文件消息摘要不对: %+v", fm)
	}

	// 撤回后不能再画图片/文件卡片
	recallRaw := fmt.Sprintf(`{"id":962,"type":"image","content":%q,"is_recalled":true}`, content)
	var rapi qapi.Message
	_ = json.Unmarshal([]byte(recallRaw), &rapi)
	rm := messageFromAPI(rapi, "7", 2)
	if rm.Kind != "" || !rm.Recalled || !strings.Contains(rm.Text, "撤回了一条消息") {
		t.Fatalf("撤回的富媒体没退化成文本: %+v", rm)
	}

	// 引用一条图片消息时，引用条也用摘要而非 content JSON
	quoteRaw := fmt.Sprintf(`{"id":963,"type":"text","content":"你看这张",
		"quoted_message":{"id":960,"type":"image","content":%q}}`, content)
	var qapiMsg qapi.Message
	_ = json.Unmarshal([]byte(quoteRaw), &qapiMsg)
	qm := messageFromAPI(qapiMsg, "7", 3)
	if !strings.Contains(qm.QuoteText, "[图片]") || strings.Contains(qm.QuoteText, "static/uploads") {
		t.Fatalf("引用摘要不对: %q", qm.QuoteText)
	}
}

// mediaTestServer 模拟上传 + 发消息两个端点，顺带记录调用次数与请求体。
func mediaTestServer(t *testing.T, failUpload bool) (*httptest.Server, *mediaProbeState) {
	t.Helper()
	st := &mediaProbeState{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/upload" && r.Method == http.MethodPost:
			st.mu.Lock()
			st.uploads++
			st.mu.Unlock()
			if failUpload {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":1001,"message":"存储不可用"}`))
				return
			}
			p1Reply(w, map[string]any{
				"id": 109, "url": "/static/uploads/pic.png",
				"name": "pic.png", "size": 105,
			})
		case strings.HasSuffix(r.URL.Path, "/messages") && r.Method == http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			st.mu.Lock()
			st.sends++
			st.lastType, _ = body["type"].(string)
			st.lastContent, _ = body["content"].(string)
			st.mu.Unlock()
			p1Reply(w, map[string]any{
				"id": 970, "conversation_id": 101, "sender_id": 7,
				"type": body["type"], "content": body["content"],
				"created_at": "2026-11-14T21:10:00+08:00",
			})
		default:
			p1Reply(w, map[string]any{})
		}
	}))
	return srv, st
}

type mediaProbeState struct {
	mu                    sync.Mutex
	uploads, sends        int
	lastType, lastContent string
}

func (s *mediaProbeState) snapshot() (int, int, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.uploads, s.sends, s.lastType, s.lastContent
}

func p2App(url string) *app {
	a := newTestApp(url)
	a.loggedIn = true
	a.selfID = "7"
	a.userName = "原生客户端测试"
	a.active = 0
	a.convs = []conversation{{ID: "101", Name: "用户一", Kind: "single",
		Color: avatarColor("101用户一")}}
	return a
}

func p2PNG(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(path, probePNG(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestP2SendAttachment(t *testing.T) {
	srv, st := mediaTestServer(t, false)
	defer srv.Close()
	a := p2App(srv.URL)
	path := p2PNG(t)

	a.sendAttachment("image", path, "101", nil, qapi.Attachment{})
	waitFor(t, "富媒体发送完成", func() bool {
		return a.readBool(func() bool {
			for i := range a.msgs {
				if a.msgs[i].Kind == "image" && !a.msgs[i].Pending {
					return true
				}
			}
			return false
		})
	})
	uploads, sends, typ, content := st.snapshot()
	if uploads != 1 || sends != 1 || typ != "image" {
		t.Fatalf("上传/发送调用不对: uploads=%d sends=%d type=%q", uploads, sends, typ)
	}
	if !strings.Contains(content, "/static/uploads/pic.png") || !strings.Contains(content, `"id":109`) {
		t.Fatalf("发送 content 不是附件 JSON: %s", content)
	}
	var sent message
	a.read(func() {
		for _, m := range a.msgs {
			if m.Kind == "image" {
				sent = m
			}
		}
	})
	if sent.Failed || sent.Att.URL != "/static/uploads/pic.png" || sent.LocalPath == "" {
		t.Fatalf("乐观消息没更新成服务端附件: %+v", sent)
	}
	if sent.ServerID != "970" || sent.Key != "m-970" {
		t.Fatalf("换 key 后没补 serverID（撤回/引用会失效）: %+v", sent)
	}
}

func TestP2UploadFailureKeepsRetryableBubble(t *testing.T) {
	srv, st := mediaTestServer(t, true)
	defer srv.Close()
	a := p2App(srv.URL)
	path := p2PNG(t)

	a.sendAttachment("image", path, "101", nil, qapi.Attachment{})
	waitFor(t, "上传失败落到气泡", func() bool {
		return a.readBool(func() bool {
			return len(a.msgs) == 1 && a.msgs[0].Failed && !a.msgs[0].Pending
		})
	})
	uploads, sends, _, _ := st.snapshot()
	if uploads != 1 || sends != 0 {
		t.Fatalf("上传失败后不该发消息: uploads=%d sends=%d", uploads, sends)
	}
	var notice string
	a.read(func() { notice = a.notice })
	if !strings.Contains(notice, "上传失败") {
		t.Fatalf("没给出失败原因: %q", notice)
	}
}

func TestP2ResendSkipsUpload(t *testing.T) {
	srv, st := mediaTestServer(t, false)
	defer srv.Close()
	a := p2App(srv.URL)
	path := p2PNG(t)
	att := qapi.Attachment{
		URL: "/static/uploads/pic.png", ID: "109", Name: "pic.png",
		Size: 105, MimeType: "image/png",
	}
	a.msgs = []message{{
		Key: "local-1", Mine: true, Kind: "image", Att: att, LocalPath: path,
		Text: mediaSummary("image", att), Failed: true,
		At: "21:00", Stamp: time.Now().UnixMilli(),
	}}

	a.retryFailed("local-1")
	waitFor(t, "重试发送完成", func() bool {
		return a.readBool(func() bool {
			return len(a.msgs) == 1 && !a.msgs[0].Pending
		})
	})
	uploads, sends, typ, _ := st.snapshot()
	if uploads != 0 || sends != 1 || typ != "image" {
		t.Fatalf("重试不该重复上传: uploads=%d sends=%d type=%q", uploads, sends, typ)
	}
	var m message
	a.read(func() { m = a.msgs[0] })
	if m.Failed || m.Att.ID != "109" {
		t.Fatalf("重试后附件信息丢了: %+v", m)
	}
}

func TestP2ImageFailureDoesNotRetryEveryFrame(t *testing.T) {
	a := &app{}
	path := "/tmp/qim-image-does-not-exist.png"
	att := qapi.Attachment{Name: "missing.png"}
	key := a.mediaKey(att, path)
	a.media.mu.Lock()
	a.media.ensure()
	a.media.errs[key] = "读取失败"
	a.media.mu.Unlock()

	_, errText, busy := a.imageState(att, path)
	if errText == "" || busy {
		t.Fatalf("失败态应该只显示错误、不再加载: err=%q busy=%v", errText, busy)
	}
	a.media.mu.Lock()
	loading := a.media.loading[key]
	a.media.mu.Unlock()
	if loading {
		t.Fatal("加载失败后又在下一帧偷偷重试，会造成请求风暴")
	}
	// 点占位块重试会清掉错误态，这时才允许再加载一次
	a.invalidateMedia(key)
	a.media.mu.Lock()
	cleared := a.media.errs[key] == ""
	a.media.mu.Unlock()
	if !cleared {
		t.Fatal("invalidateMedia 没清掉错误态")
	}
}

func TestP2RenderShots(t *testing.T) {
	imgAtt := qapi.Attachment{
		URL: "/static/uploads/screenshot.png", ID: "109",
		Name: "界面截图.png", Size: 2048, MimeType: "image/png",
	}
	fileAtt := qapi.Attachment{
		URL: "/static/f/需求文档.pdf", ID: "110",
		Name: "需求文档.pdf", Size: 2 << 20, MimeType: "application/pdf",
	}
	localPath := "/tmp/local-preview.png"
	localAtt := qapi.Attachment{Name: "local-preview.png", MimeType: "image/png", Size: 105}

	now := time.Now().UnixMilli()
	a := &app{
		active: 0, loggedIn: true, userName: "原生客户端测试",
		connStatus: qapi.StatusConnected, selfID: "7",
		convs: []conversation{{ID: "106", Name: "用户一", Kind: "single",
			Preview: "[图片] 界面截图.png", Time: "21:10", Color: avatarColor("106用户一")}},
		msgs: []message{
			{Key: "day", Pill: "2026-11-14"},
			{Key: "m-1", ServerID: "1", Sender: "用户一", Text: "这是刚截的图", At: "21:08", Stamp: now},
			{Key: "m-2", ServerID: "2", Sender: "用户一", Kind: "image", Att: imgAtt,
				Text: mediaSummary("image", imgAtt), At: "21:09", Stamp: now},
			{Key: "m-3", ServerID: "3", Mine: true, Kind: "file", Att: fileAtt,
				Text: mediaSummary("file", fileAtt), At: "21:10", Stamp: now},
			{Key: "local-4", Mine: true, Kind: "image", Att: localAtt, LocalPath: localPath,
				Text: mediaSummary("image", localAtt), At: "21:11", Stamp: now, Pending: true},
		},
	}
	// headless 没有真实窗口，也不该发网络请求：把解码好的位图直接注入缓存，
	// 这样截图里能看到真正的图片气泡而不是占位块。
	src, err := ui.DecodeImage(probePNG())
	if err != nil {
		t.Fatal(err)
	}
	a.media.mu.Lock()
	a.media.ensure()
	a.media.images[a.mediaKey(imgAtt, "")] = src
	a.media.images[a.mediaKey(localAtt, localPath)] = src
	a.media.mu.Unlock()

	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"这是刚截的图", "需求文档.pdf", "2.0 MB", "点击下载"} {
		if !tt.HasText(want) {
			t.Errorf("界面缺 %q, texts=%q", want, tt.Texts())
		}
	}
	for _, bad := range []string{"图片加载中", "图片加载失败", "static/uploads"} {
		if tt.HasText(bad) {
			t.Errorf("界面不该出现 %q, texts=%q", bad, tt.Texts())
		}
	}
	write(t, tt, "/tmp/qim-p2-chat.png")
}

// TestMessageAvatarTopAligned 原版 .message-item 是 align-items:flex-start：
// 头像顶部对齐「昵称 + 气泡」列的顶部，不能垂直居中在高图片气泡旁边。
func TestMessageAvatarTopAligned(t *testing.T) {
	att := qapi.Attachment{URL: "/static/uploads/tall.png", ID: "200",
		Name: "tall.png", Size: 4096, MimeType: "image/png"}
	a := &app{
		active: 0, loggedIn: true, userName: "测试用户",
		connStatus: qapi.StatusConnected, selfID: "7",
		convs: []conversation{{ID: "106", Name: "技术交流群", Kind: "group",
			Preview: "[图片]", Time: "11:40"}},
		msgs: []message{{Key: "m1", ServerID: "1", Sender: "用户一", Kind: "image",
			Att: att, Text: mediaSummary("image", att), At: "11:40"}},
	}
	// 造一张 800×600 的大图，让气泡明显高于头像（居中错位最容易看出来）。
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.RGBA{R: 235, G: 243, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	src, err := ui.DecodeImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	a.media.mu.Lock()
	a.media.ensure()
	a.media.images[a.mediaKey(att, "")] = src
	a.media.mu.Unlock()

	tt := ui.NewTester(a.view, 1050, 703)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("用户一") {
		t.Fatalf("缺发送者昵称, texts=%q", tt.Texts())
	}
	write(t, tt, "/tmp/qim-avatar-align.png")
}
