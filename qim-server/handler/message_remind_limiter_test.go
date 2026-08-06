package handler

import (
	"testing"
	"time"
)

func TestReminderLimiterAllowsRetryAfterFailure(t *testing.T) {
	limiter := newReminderLimiter()
	messageID := uint(42)

	if !limiter.start(messageID, time.Now()) {
		t.Fatal("expected first reminder attempt to start")
	}
	limiter.finish(messageID, false, time.Now())

	if !limiter.start(messageID, time.Now()) {
		t.Fatal("expected failed reminder attempt to be retryable immediately")
	}
}

func TestReminderLimiterBlocksConcurrentAndRecentSuccess(t *testing.T) {
	limiter := newReminderLimiter()
	messageID := uint(42)
	now := time.Now()

	if !limiter.start(messageID, now) {
		t.Fatal("expected first reminder attempt to start")
	}
	if limiter.start(messageID, now) {
		t.Fatal("expected concurrent reminder attempt to be blocked")
	}

	limiter.finish(messageID, true, now)
	if limiter.start(messageID, now.Add(30*time.Minute)) {
		t.Fatal("expected reminder to be blocked for one hour after success")
	}
	if !limiter.start(messageID, now.Add(time.Hour+time.Second)) {
		t.Fatal("expected reminder to be allowed after one hour")
	}
}

func TestReminderLimiterRemovesExpiredSuccessfulEntries(t *testing.T) {
	limiter := newReminderLimiter()
	now := time.Now()

	if !limiter.start(42, now) {
		t.Fatal("expected first reminder attempt to start")
	}
	limiter.finish(42, true, now)

	if !limiter.start(99, now.Add(time.Hour+time.Second)) {
		t.Fatal("expected unrelated reminder attempt to start")
	}

	if _, ok := limiter.entries[42]; ok {
		t.Fatal("expected expired successful reminder entry to be removed")
	}
}

func TestReminderLimiterCheckReturnsReasonWithoutStarting(t *testing.T) {
	limiter := newReminderLimiter()
	now := time.Now()

	if reason := limiter.check(42, now); reason != reminderAllowed {
		t.Fatalf("expected reminder to be allowed, got %v", reason)
	}
	if entry, ok := limiter.entries[42]; ok && entry.pending {
		t.Fatal("expected check to avoid marking reminder as pending")
	}

	if !limiter.start(42, now) {
		t.Fatal("expected reminder attempt to start after check")
	}
	if reason := limiter.check(42, now); reason != reminderPending {
		t.Fatalf("expected pending reminder reason, got %v", reason)
	}

	limiter.finish(42, true, now)
	if reason := limiter.check(42, now.Add(30*time.Minute)); reason != reminderCoolingDown {
		t.Fatalf("expected cooling-down reminder reason, got %v", reason)
	}
}

func TestReminderLimiterCustomCooldown(t *testing.T) {
	limiter := newReminderLimiter()
	limiter.SetCooldown(30 * time.Second)
	messageID := uint(7)
	now := time.Now()

	if !limiter.start(messageID, now) {
		t.Fatal("expected first reminder attempt to start")
	}
	limiter.finish(messageID, true, now)
	if reason := limiter.check(messageID, now.Add(10*time.Second)); reason != reminderCoolingDown {
		t.Fatalf("expected cooling-down within 30s cooldown, got %v", reason)
	}
	if !limiter.start(messageID, now.Add(31*time.Second)) {
		t.Fatal("expected reminder to be allowed after 30s cooldown")
	}
}

func TestReminderLimiterZeroCooldownAllowsRepeat(t *testing.T) {
	limiter := newReminderLimiter()
	limiter.SetCooldown(0) // 不限制，可反复提醒
	messageID := uint(8)
	now := time.Now()

	if !limiter.start(messageID, now) {
		t.Fatal("expected first reminder attempt to start")
	}
	limiter.finish(messageID, true, now)
	if reason := limiter.check(messageID, now.Add(time.Second)); reason != reminderAllowed {
		t.Fatalf("expected reminder to be allowed immediately with 0 cooldown, got %v", reason)
	}
}

func TestReminderLimiterReasonMessage(t *testing.T) {
	tests := []struct {
		reason   reminderLimiterReason
		cooldown time.Duration
		want     string
	}{
		{reminderPending, time.Hour, "提醒发送中，请稍候"},
		{reminderCoolingDown, time.Hour, "该消息已提醒过，请 60 分钟后再试"},
		{reminderCoolingDown, 90 * time.Second, "该消息已提醒过，请 1 分钟后再试"},
		{reminderCoolingDown, 30 * time.Second, "该消息已提醒过，请 30 秒后再试"},
		{reminderCoolingDown, 0, "该消息已提醒过，请稍后再试"},
	}

	for _, tt := range tests {
		if got := reminderLimiterReasonMessage(tt.reason, tt.cooldown); got != tt.want {
			t.Fatalf("expected %q, got %q", tt.want, got)
		}
	}
}

func TestReminderLimiterDeferredFinishReleasesPendingAfterPanic(t *testing.T) {
	limiter := newReminderLimiter()
	messageID := uint(42)

	if !limiter.start(messageID, time.Now()) {
		t.Fatal("expected first reminder attempt to start")
	}

	func() {
		success := false
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("expected panic to be recovered")
			}
		}()
		defer finishReminderAttempt(limiter, messageID, &success)

		panic("webhook panic")
	}()

	if !limiter.start(messageID, time.Now()) {
		t.Fatal("expected reminder attempt to be retryable after panic")
	}
}
