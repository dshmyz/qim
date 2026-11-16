package qapi

import "testing"

func TestParseCardInfo(t *testing.T) {
	ci := ParseCardInfo(`{"title":"确认下单","text":"订单 #123 等待确认","buttons":[{"id":"ok","text":"确认","style":"primary"},{"id":"no","text":"取消"}]}`)
	if ci == nil {
		t.Fatal("parse failed")
	}
	if ci.Title != "确认下单" || ci.Text != "订单 #123 等待确认" || len(ci.Buttons) != 2 {
		t.Fatalf("got %+v", ci)
	}
	if ParseCardInfo("") != nil {
		t.Fatal("empty content → nil")
	}
	if ParseCardInfo("not json") != nil {
		t.Fatal("bad json → nil")
	}
	if ParseCardInfo(`{"foo":1}`) != nil {
		t.Fatal("no title/text → nil")
	}
}

func TestParseNewsInfo(t *testing.T) {
	ni := ParseNewsInfo(`{"title":"版本发布","summary":"v2.0 上线","url":"https://example.com","image":"/s/a.png"}`)
	if ni == nil || ni.Title != "版本发布" || ni.URL != "https://example.com" {
		t.Fatalf("got %+v", ni)
	}
	if ParseNewsInfo(`{"foo":1}`) != nil {
		t.Fatal("empty payload → nil")
	}
}

func TestParseMiniAppInfo(t *testing.T) {
	mi := ParseMiniAppInfo(`{"id":"wx1","display_name":"青雀助手","description":"快速记录","path":"pages/index"}`)
	if mi == nil || mi.Label() != "青雀助手" || mi.Desc != "快速记录" {
		t.Fatalf("got %+v", mi)
	}
	// display_name 缺失时退 name，再退 title
	mi2 := ParseMiniAppInfo(`{"name":"工具","title":"工具箱"}`)
	if mi2 == nil || mi2.Label() != "工具" {
		t.Fatalf("got %+v", mi2)
	}
	mi3 := ParseMiniAppInfo(`{"title":"仅标题"}`)
	if mi3 == nil || mi3.Label() != "仅标题" {
		t.Fatalf("got %+v", mi3)
	}
}

func TestPreviewStructuredTypes(t *testing.T) {
	m := Message{Type: "card", Content: `{"title":"审批","text":"请确认"}`}
	if got := m.Preview(); got != "[卡片] 审批" {
		t.Fatalf("card preview %q", got)
	}
	m2 := Message{Type: "news", Content: `{"title":"公告"}`}
	if got := m2.Preview(); got != "[资讯] 公告" {
		t.Fatalf("news preview %q", got)
	}
	m3 := Message{Type: "miniApp", Content: `{"name":"便签"}`}
	if got := m3.Preview(); got != "[小程序] 便签" {
		t.Fatalf("miniApp preview %q", got)
	}
	// 解析失败退化短标签，不露 JSON
	m4 := Message{Type: "card", Content: "garbage"}
	if got := m4.Preview(); got != "[卡片]" {
		t.Fatalf("bad card preview %q", got)
	}
}
