package client

// -probe-media：对真实 qim-server 验证 P2 富媒体协议闭环。
// 与 -probe-api / -probe-actions 一样不开窗口，只验证协议：
//
//	go run . -probe-media -server http://127.0.0.1:8080 -user native_probe -pass '...'
//
// 会生成一张小 PNG、上传、发 image/file 消息、下载比对字节，最后撤回清理。
// 只应对本地开发库使用，不要指向生产环境。

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

var probeMedia = flag.Bool("probe-media", false, "只跑上传/富媒体消息/下载的真实接口验证（不开窗口），跑完退出")

func probePNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func runMediaProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-media -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())
	fmt.Println("注意: 该探针会上传一张测试图并在第一个会话写入两条消息，仅用于本地开发库。")
	fmt.Println()

	sess, err := c.Login(ctx, user, pass)
	if err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[登录] ✓ %s id=%s\n", sess.User.DisplayName(), sess.User.ID)

	convs, err := c.Conversations(ctx)
	if err != nil || len(convs) == 0 {
		fmt.Printf("[会话] ✗ 没有可用会话: %v\n", err)
		return 1
	}
	conv := convs[0]
	fmt.Printf("[会话] ✓ 使用 [%s] %s\n", conv.ID, conv.DisplayName())

	// [1] 本地造一张真实 PNG（不是随便一段字节），后面下载要按字节比对。
	pngData := probePNG()
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("qim-media-probe-%d.png", time.Now().Unix()))
	if err := os.WriteFile(tmp, pngData, 0o600); err != nil {
		fmt.Printf("[1/6 测试图] ✗ %v\n", err)
		return 1
	}
	defer os.Remove(tmp)
	fmt.Printf("[1/6 测试图] ✓ %d 字节 %s\n", len(pngData), tmp)

	// [2] multipart 上传
	att, err := c.Upload(ctx, tmp)
	if err != nil {
		fmt.Printf("[2/6 上传] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[2/6 上传] ✓ id=%s url=%s name=%s mime=%s size=%d\n",
		att.ID, att.URL, att.Name, att.MimeType, att.Size)
	if att.URL == "" || !att.IsImage() {
		fmt.Println("[2/6 上传] ✗ 返回的附件不是可渲染图片")
		return 1
	}

	// [3] 发 image 消息，并从历史回查（不假设 SendMessage 的响应体形状）
	content := att.Content()
	if _, err := c.SendMessage(ctx, conv.ID, qapi.MediaRequest("image", att, "")); err != nil {
		fmt.Printf("[3/6 image 消息] ✗ %v\n", err)
		return 1
	}
	var imageMsg *qapi.Message
	page, err := c.Messages(ctx, conv.ID, 1, 10)
	if err != nil {
		fmt.Printf("[3/6 image 消息] ✗ 拉历史失败: %v\n", err)
		return 1
	}
	for i := len(page.Messages) - 1; i >= 0; i-- {
		if page.Messages[i].Content == content && page.Messages[i].Type == "image" {
			cp := page.Messages[i]
			imageMsg = &cp
			break
		}
	}
	if imageMsg == nil {
		fmt.Println("[3/6 image 消息] ✗ 历史里找不到刚发的 image 消息")
		return 1
	}
	fmt.Printf("[3/6 image 消息] ✓ id=%s type=%s content=%s\n",
		imageMsg.ID, imageMsg.Type, imageMsg.Content)

	// [4] 从历史消息重新解析附件，再直连 URL 下载比对
	parsed := qapi.ParseAttachment(imageMsg.Content, imageMsg)
	if parsed.URL != att.URL || parsed.ID != att.ID || !parsed.IsImage() {
		fmt.Printf("[4/6 下载] ✗ content 往返解析不一致: %+v\n", parsed)
		return 1
	}
	got, err := c.Download(ctx, parsed)
	if err != nil {
		fmt.Printf("[4/6 下载] ✗ 直连 URL 失败: %v\n", err)
		return 1
	}
	if !bytes.Equal(got, pngData) {
		fmt.Printf("[4/6 下载] ✗ 字节不一致: 原 %d / 下载 %d\n", len(pngData), len(got))
		return 1
	}
	fmt.Printf("[4/6 下载] ✓ 直连 %s 取回 %d 字节，与原图完全一致\n",
		c.ResolveURL(parsed.URL), len(got))

	// [5] 只用 id 走 /api/v1/files/:id/download 回退通道
	got2, err := c.Download(ctx, qapi.Attachment{ID: att.ID, Name: att.Name})
	if err != nil {
		fmt.Printf("[5/6 文件接口下载] ✗ %v\n", err)
		return 1
	}
	if !bytes.Equal(got2, pngData) {
		fmt.Printf("[5/6 文件接口下载] ✗ 字节不一致: %d\n", len(got2))
		return 1
	}
	fmt.Printf("[5/6 文件接口下载] ✓ /files/%s/download 取回 %d 字节\n", att.ID, len(got2))

	// [6] 同一个附件按 file 类型发一条，验证两种 type 都收
	fileAtt := att
	fileAtt.MimeType = "application/octet-stream"
	if _, err := c.SendMessage(ctx, conv.ID, qapi.MediaRequest("file", fileAtt, "")); err != nil {
		fmt.Printf("[6/6 file 消息] ✗ %v\n", err)
		return 1
	}
	page, err = c.Messages(ctx, conv.ID, 1, 10)
	if err != nil {
		fmt.Printf("[6/6 file 消息] ✗ 拉历史失败: %v\n", err)
		return 1
	}
	var fileMsgID string
	for i := len(page.Messages) - 1; i >= 0; i-- {
		if page.Messages[i].Type == "file" && page.Messages[i].Content == fileAtt.Content() {
			fileMsgID = page.Messages[i].ID.String()
			break
		}
	}
	if fileMsgID == "" {
		fmt.Println("[6/6 file 消息] ✗ 历史里找不到刚发的 file 消息")
		return 1
	}
	fmt.Printf("[6/6 file 消息] ✓ id=%s name=%s size=%s\n",
		fileMsgID, qapi.ParseAttachment(fileAtt.Content(), nil).DisplayName(),
		qapi.FormatSize(fileAtt.Size))

	// 清理：两条探针消息撤回掉，别把开发库越跑越脏。
	for _, id := range []string{imageMsg.ID.String(), fileMsgID} {
		if id == "" {
			continue
		}
		if _, err := c.RecallMessage(ctx, id); err != nil {
			fmt.Printf("[清理] ! %s 撤回失败（超过时限可忽略）: %v\n", id, err)
		}
	}
	fmt.Println("[清理] ✓ 两条探针消息已尝试撤回")

	fmt.Println()
	fmt.Println("结论: multipart 上传 → image/file 消息 content → 直连下载 / 文件接口下载 全部对齐。")
	return 0
}
