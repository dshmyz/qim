package ws

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
)

// TriggerMessageSent 是消息发送后回调的统一派发入口（HTTP / WebSocket 两条发送路径共用）：
//   - system 发送者：不派发（避免一次群发给 N 个用户派生 N 个 AI 触发 goroutine）
//   - @all 广播：只走 OnAllMentionedSent（群待办提取）——智能回复与群 bot 转发对 @all
//     不触发是既有设计（避免噪音），保留在 OnMessageSent 侧
//   - 其余：OnMessageSent（智能回复 / 群 bot 转发 / 待办提取）

func newDispatchTestHub() *Hub {
	return &Hub{}
}

func waitSignal(t *testing.T, flag *atomic.Bool, name string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("等待 %s 超时", name)
		default:
			if flag.Load() {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// TestTriggerMessageSent_AllMentionRoutesOnlyToDedicatedCallback 问题④取证：
// @all 广播此前被三处 IsAllMentioned 一刀切拦掉，待办提取对「@所有人 明天交全员周报」
// 这类最典型的任务分派场景完全不生效。分流后 @all 只进专用回调。
func TestTriggerMessageSent_AllMentionRoutesOnlyToDedicatedCallback(t *testing.T) {
	h := newDispatchTestHub()
	var allCalled, normalCalled atomic.Bool
	h.OnAllMentionedSent = func(msg *model.Message) { allCalled.Store(true) }
	h.OnMessageSent = func(msg *model.Message, ids []uint) { normalCalled.Store(true) }

	msg := &model.Message{
		Content: mention.EncodeAll() + " 明天交全员周报",
		Sender:  model.User{Type: "user"},
	}
	h.TriggerMessageSent(msg, nil)

	waitSignal(t, &allCalled, "OnAllMentionedSent")
	time.Sleep(50 * time.Millisecond)
	assertNeverNormal(t, &normalCalled, "@all 不应触发 OnMessageSent（智能回复仍被跳过）")
}

// TestTriggerMessageSent_RegularMessageRoutesToOnMessageSent 普通消息走既有回调。
func TestTriggerMessageSent_RegularMessageRoutesToOnMessageSent(t *testing.T) {
	h := newDispatchTestHub()
	var allCalled, normalCalled atomic.Bool
	h.OnAllMentionedSent = func(msg *model.Message) { allCalled.Store(true) }
	h.OnMessageSent = func(msg *model.Message, ids []uint) { normalCalled.Store(true) }

	msg := &model.Message{
		Content: "明天交报告",
		Sender:  model.User{Type: "user"},
	}
	h.TriggerMessageSent(msg, []uint{7})

	waitSignal(t, &normalCalled, "OnMessageSent")
	time.Sleep(50 * time.Millisecond)
	assertNeverNormal(t, &allCalled, "普通消息不应走 @all 专用回调")
}

// TestTriggerMessageSent_SystemSenderSkipped system 发送者两条回调都不触发。
func TestTriggerMessageSent_SystemSenderSkipped(t *testing.T) {
	h := newDispatchTestHub()
	var allCalled, normalCalled atomic.Bool
	h.OnAllMentionedSent = func(msg *model.Message) { allCalled.Store(true) }
	h.OnMessageSent = func(msg *model.Message, ids []uint) { normalCalled.Store(true) }

	msg := &model.Message{
		Content: "群发通知",
		Sender:  model.User{Type: "system"},
	}
	h.TriggerMessageSent(msg, nil)

	time.Sleep(80 * time.Millisecond)
	assertNeverNormal(t, &normalCalled, "system 发送者不应触发 OnMessageSent")
	assertNeverNormal(t, &allCalled, "system 发送者不应触发 OnAllMentionedSent")
}

func assertNeverNormal(t *testing.T, flag *atomic.Bool, msg string) {
	t.Helper()
	if flag.Load() {
		t.Fatal(msg)
	}
}
