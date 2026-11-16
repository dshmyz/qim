package client

// P0 事件测试：撤回/删除/更新对气泡的影响、真 WS 上的事件分发、
// sync_hint 的增量补偿与 acknowledge_sync。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"qim-newclient/internal/qapi"
)

func mustMessage(t *testing.T, raw string) qapi.Message {
	t.Helper()
	var m qapi.Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMessagePatchHandlers(t *testing.T) {
	a := &app{
		active: 0,
		convs:  []conversation{{ID: "101", Preview: "原文"}},
		msgs: []message{
			{Key: "m-1", Text: "原文", Sender: "张三"},
			{Key: "m-2", Text: "第二条"},
			{Key: "m-3", Text: "第三条"},
		},
	}

	a.applyRecalled(mustMessage(t, `{"id":1,"conversation_id":101,"content":"[消息已撤回]","is_recalled":true}`))
	// 对齐 qim-client：占位是「{发送者} 撤回了一条消息」，sender 不再清空
	if a.msgs[0].Text != "未知用户 撤回了一条消息" || a.msgs[0].Sender != "张三" {
		t.Errorf("撤回没改气泡: %+v", a.msgs[0])
	}
	if a.convs[0].Preview != "未知用户 撤回了一条消息" {
		t.Errorf("侧栏预览没同步: %q", a.convs[0].Preview)
	}

	a.removeMessage("2")
	if len(a.msgs) != 2 || a.msgs[1].Key != "m-3" {
		t.Errorf("删除结果不对: %+v", a.msgs)
	}

	a.applyUpdated(mustMessage(t, `{"id":3,"conversation_id":101,"content":"改过的正文","type":"text"}`), "7")
	if a.msgs[1].Text != "改过的正文" {
		t.Errorf("正文更新没生效: %q", a.msgs[1].Text)
	}
	// 流式中间帧不能覆盖已显示的正文
	a.applyUpdated(mustMessage(t, `{"id":3,"conversation_id":101,"content":"半截","type":"streaming"}`), "7")
	if a.msgs[1].Text != "改过的正文" {
		t.Errorf("流式帧覆盖了正文: %q", a.msgs[1].Text)
	}
}

func wsUpgrade(w http.ResponseWriter, r *http.Request) *websocket.Conn {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return nil
	}
	return conn
}

func TestRealtimeEventHandlers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ws" {
			jsonReply(w, 0, map[string]any{})
			return
		}
		conn := wsUpgrade(w, r)
		if conn == nil {
			return
		}
		defer conn.Close()
		var auth struct {
			Type string `json:"type"`
		}
		if err := conn.ReadJSON(&auth); err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{"type": "auth_success"})
		_ = conn.WriteJSON(map[string]any{"type": "message_recalled", "data": map[string]any{
			"id": 1, "conversation_id": 101, "content": "[消息已撤回]", "is_recalled": true,
		}})
		_ = conn.WriteJSON(map[string]any{"type": "message_deleted", "data": map[string]any{
			"message_id": 2, "conversation_id": 101,
		}})
		_ = conn.WriteJSON(map[string]any{"type": "message_updated", "data": map[string]any{
			"id": 3, "conversation_id": 101, "content": "改过的", "type": "text",
		}})
		time.Sleep(800 * time.Millisecond)
	}))
	defer srv.Close()

	a := newTestApp(srv.URL)
	a.client.SetSession("tkn", "rtk")
	a.mutate(func() {
		a.active = 0
		a.convs = []conversation{{ID: "101"}}
		a.msgs = []message{
			{Key: "m-1", Text: "原文"}, {Key: "m-2", Text: "第二条"}, {Key: "m-3", Text: "第三条"},
		}
	})
	a.startRealtime()
	defer func() {
		a.rt.Stop()
		a.cancel()
	}()

	waitFor(t, "撤回/删除/更新三个事件处理完", func() bool {
		return a.readBool(func() bool {
			return len(a.msgs) == 2 && a.msgs[0].Recalled && a.msgs[1].Text == "改过的"
		})
	})
}

func TestSyncHintRecoversMissedMessages(t *testing.T) {
	var afterID atomic.Value
	ack := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/ws":
			conn := wsUpgrade(w, r)
			if conn == nil {
				return
			}
			defer conn.Close()
			var auth struct {
				Type string `json:"type"`
			}
			if err := conn.ReadJSON(&auth); err != nil {
				return
			}
			_ = conn.WriteJSON(map[string]any{"type": "auth_success"})
			_ = conn.WriteJSON(map[string]any{"type": "sync_hint", "data": map[string]any{"reason": "buffer_overflow"}})
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			for {
				var m struct {
					Type string `json:"type"`
				}
				if err := conn.ReadJSON(&m); err != nil {
					return
				}
				if m.Type == "acknowledge_sync" {
					ack <- struct{}{}
					return
				}
			}

		case r.URL.Path == "/api/v1/conversations":
			jsonReply(w, 0, map[string]any{"list": []any{
				map[string]any{"id": 101, "name": "技术交流群"},
			}})

		case strings.HasSuffix(r.URL.Path, "/messages") && r.Method == http.MethodGet:
			afterID.Store(r.URL.Query().Get("after_id"))
			jsonReply(w, 0, map[string]any{"messages": []any{
				map[string]any{
					"id": 4, "conversation_id": 101, "sender_id": 8, "type": "text",
					"content": "补回来的", "created_at": "2026-11-14T10:07:00+08:00",
				},
			}})

		default:
			jsonReply(w, 0, map[string]any{})
		}
	}))
	defer srv.Close()

	a := newTestApp(srv.URL)
	a.client.SetSession("tkn", "rtk")
	a.mutate(func() {
		a.active = 0
		a.convs = []conversation{{ID: "101"}}
		a.msgs = []message{
			{Key: "day-2026-11-14", Pill: "2026-11-14"},
			{Key: "m-2", Text: "最后一条"},
		}
	})
	a.startRealtime()
	defer func() {
		a.rt.Stop()
		a.cancel()
	}()

	select {
	case <-ack:
	case <-time.After(3 * time.Second):
		t.Fatal("补偿成功后没有发 acknowledge_sync")
	}
	if got, _ := afterID.Load().(string); got != "2" {
		t.Errorf("after_id 应为本地最大消息 id 2，实际 %q", got)
	}
	waitFor(t, "补回的消息上屏", func() bool {
		return a.readBool(func() bool {
			for _, m := range a.msgs {
				if m.Text == "补回来的" {
					return true
				}
			}
			return false
		})
	})
}
