package client

// 核心聊天补齐的单元测试：待发送文件队列（粘贴/拖拽）、@ 我横幅计数。

import (
	"os"
	"path/filepath"
	"testing"

	"qim-newclient/internal/qapi"
)

func TestPendingKind(t *testing.T) {
	cases := map[string]string{
		"a.png": "image", "b.JPG": "image", "c.webp": "image",
		"d.pdf": "file", "e.zip": "file", "f": "file",
	}
	for name, want := range cases {
		if got := pendingKind(name); got != want {
			t.Fatalf("%s → %s, want %s", name, got, want)
		}
	}
}

func TestPendingQueue(t *testing.T) {
	a := &app{active: 0, loggedIn: true}
	tmp := t.TempDir()
	p1 := filepath.Join(tmp, "qim-paste-1.png")
	p2 := filepath.Join(tmp, "doc.pdf")
	_ = os.WriteFile(p1, []byte("png"), 0o644)
	_ = os.WriteFile(p2, []byte("pdf"), 0o644)

	a.pendingFiles = []string{p1, p2}
	a.removePendingFile(0)
	if len(a.pendingFiles) != 1 || a.pendingFiles[0] != p2 {
		t.Fatalf("after remove: %v", a.pendingFiles)
	}
	if _, err := os.Stat(p1); !os.IsNotExist(err) {
		t.Fatal("paste temp file should be removed")
	}
	a.clearPendingFiles()
	if len(a.pendingFiles) != 0 {
		t.Fatal("queue should be empty")
	}
	// 普通（非临时）文件不被删除
	if _, err := os.Stat(p2); err != nil {
		t.Fatal("user file should survive clear")
	}
}

func TestTakePasteNoClipboard(t *testing.T) {
	a := &app{loggedIn: true} // svc 为零值：headless 下 ReadClipboardData 返回空
	if a.takePaste("hello") {
		t.Fatal("empty clipboard must not take the paste")
	}
	if len(a.pendingFiles) != 0 {
		t.Fatal("queue must stay empty")
	}
}

func TestAtBannerCount(t *testing.T) {
	a := &app{active: 0, loggedIn: true, atSeen: map[string]bool{}}
	mk := func(id, senderID string, atMe, mine bool) message {
		m := messageFromAPI(mustMsg(
			`{"id":"`+id+`","type":"text","content":"@{mention:7|我} 看这条","sender":{"id":"`+senderID+`","name":"x"},"timestamp":1700000000000}`), "7", 0)
		m.AtMe = atMe
		m.Mine = mine
		return m
	}
	a.msgs = []message{
		{Key: "pill", Pill: "11-15"},
		mk("41", "8", true, false),
		mk("42", "8", true, false),
		mk("43", "8", false, false), // 没 @ 我
		mk("44", "7", true, true),   // 自己发的
	}
	if n := a.unreadAtCount(); n != 2 {
		t.Fatalf("count = %d, want 2", n)
	}
	idx, first, ok := a.firstUnreadAt()
	if !ok || idx != 1 || first.Key != "m-41" {
		t.Fatalf("first = idx %d key %s ok %v", idx, first.Key, ok)
	}
	// 消费第一条后计数 -1，再点定位到下一条
	a.atSeen[first.Key] = true
	if n := a.unreadAtCount(); n != 1 {
		t.Fatalf("after consume count = %d", n)
	}
	_, second, ok := a.firstUnreadAt()
	if !ok || second.Key != "m-42" {
		t.Fatalf("second = %s ok %v", second.Key, ok)
	}
}

func TestStructuredCardsParse(t *testing.T) {
	card := messageFromAPI(mustMsg(`{"id":"51","type":"card","content":"{\"title\":\"下单\",\"text\":\"确认\",\"buttons\":[{\"id\":\"ok\",\"text\":\"确认\"}]}","sender":{"id":"8","name":"bot"},"timestamp":1700000000000}`), "7", 0)
	if card.CD == nil || card.CD.Title != "下单" || len(card.CD.Buttons) != 1 {
		t.Fatalf("card %+v", card.CD)
	}
	news := messageFromAPI(mustMsg(`{"id":"52","type":"news","content":"{\"title\":\"发布\",\"url\":\"https://a.com\"}","sender":{"id":"8","name":"bot"},"timestamp":1700000001000}`), "7", 0)
	if news.NW == nil || news.NW.Title != "发布" {
		t.Fatalf("news %+v", news.NW)
	}
	mini := messageFromAPI(mustMsg(`{"id":"53","type":"miniApp","content":"{\"name\":\"便签\",\"description\":\"记一笔\"}","sender":{"id":"8","name":"bot"},"timestamp":1700000002000}`), "7", 0)
	if mini.MI == nil || mini.MI.Label() != "便签" {
		t.Fatalf("mini %+v", mini.MI)
	}
	// 坏 payload 退化普通气泡：三个指针全空，text 是可读短标签
	bad := messageFromAPI(mustMsg(`{"id":"54","type":"card","content":"not-json","sender":{"id":"8","name":"bot"},"timestamp":1700000003000}`), "7", 0)
	if bad.CD != nil || bad.Text != "[卡片]" {
		t.Fatalf("bad card: cd=%v text=%q", bad.CD, bad.Text)
	}
	_ = qapi.CardInfo{} // 保持 qapi 引用（struct 字段已覆盖）
}
