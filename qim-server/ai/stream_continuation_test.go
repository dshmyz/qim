package ai

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func finishPtr(s string) *string { return &s }

func TestRunStreamContinuation_LengthThenSucceeds(t *testing.T) {
	var forwarded []string
	onChunk := func(c StreamChunk) error {
		if c.Content != "" {
			forwarded = append(forwarded, c.Content)
		}
		return nil
	}

	primary := func(_ context.Context, cb func(StreamChunk) error) error {
		_ = cb(StreamChunk{Content: "正文前半段"})
		return cb(StreamChunk{Finish: finishPtr("length")})
	}
	continuation := func(_ context.Context, partial string, cb func(StreamChunk) error) error {
		if partial != "正文前半段" {
			t.Fatalf("unexpected partial: %q", partial)
		}
		_ = cb(StreamChunk{Content: "接续后半段"})
		return cb(StreamChunk{Finish: finishPtr("stop")})
	}

	if err := runStreamWithContinuation(context.Background(), primary, continuation, onChunk); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if strings.Join(forwarded, "|") != "正文前半段|接续后半段" {
		t.Fatalf("unexpected chunks: %v", forwarded)
	}
}

func TestRunStreamContinuation_StillLengthAppendsMarker(t *testing.T) {
	var forwarded []string
	onChunk := func(c StreamChunk) error {
		forwarded = append(forwarded, c.Content)
		return nil
	}
	primary := func(_ context.Context, cb func(StreamChunk) error) error {
		_ = cb(StreamChunk{Content: "很长的回答"})
		return cb(StreamChunk{Finish: finishPtr("length")})
	}
	continuation := func(_ context.Context, _ string, cb func(StreamChunk) error) error {
		_ = cb(StreamChunk{Content: "还是很长"})
		return cb(StreamChunk{Finish: finishPtr("length")})
	}

	err := runStreamWithContinuation(context.Background(), primary, continuation, onChunk)
	if !errors.Is(err, ErrReplyTruncated) {
		t.Fatalf("expected ErrReplyTruncated, got %v", err)
	}
	if !errors.Is(err, errFinishLength) {
		t.Fatalf("errFinishLength must be joined, got %v", err)
	}
	last := forwarded[len(forwarded)-1]
	if last != truncatedMarker {
		t.Fatalf("marker must be appended last, got %q", last)
	}
}

func TestRunStreamContinuation_ErrorAfterContentAppendsMarker(t *testing.T) {
	var sawMarker bool
	onChunk := func(c StreamChunk) error {
		if c.Content == truncatedMarker {
			sawMarker = true
		}
		return nil
	}
	boom := errors.New("connection reset")
	primary := func(_ context.Context, cb func(StreamChunk) error) error {
		_ = cb(StreamChunk{Content: "半截"})
		return boom
	}
	continuation := func(_ context.Context, _ string, _ func(StreamChunk) error) error {
		t.Fatal("continuation must not be called for generic errors")
		return nil
	}

	err := runStreamWithContinuation(context.Background(), primary, continuation, onChunk)
	if !errors.Is(err, ErrReplyTruncated) || !errors.Is(err, boom) {
		t.Fatalf("expected joined truncation+boom, got %v", err)
	}
	if !sawMarker {
		t.Fatal("marker expected")
	}

	if InterruptReasonFromError(err) != "error" {
		t.Fatalf("reason should be error")
	}
}

func TestRunStreamContinuation_NoContentErrorReturnedRaw(t *testing.T) {
	primary := func(_ context.Context, _ func(StreamChunk) error) error {
		return io.ErrUnexpectedEOF
	}
	continuation := func(_ context.Context, _ string, _ func(StreamChunk) error) error {
		t.Fatal("must not continue without content")
		return nil
	}
	err := runStreamWithContinuation(context.Background(), primary, continuation,
		func(StreamChunk) error { return nil })
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("raw error expected, got %v", err)
	}
}

func TestRunStreamContinuation_NoDoneTriggersContinue(t *testing.T) {
	onChunk := func(StreamChunk) error { return nil }
	primary := func(_ context.Context, cb func(StreamChunk) error) error {
		_ = cb(StreamChunk{Content: "已有正文"})
		return ErrStreamIncomplete
	}
	continued := false
	continuation := func(_ context.Context, _ string, cb func(StreamChunk) error) error {
		continued = true
		return cb(StreamChunk{Finish: finishPtr("stop")})
	}
	if err := runStreamWithContinuation(context.Background(), primary, continuation, onChunk); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !continued {
		t.Fatal("no-DONE must trigger continuation")
	}
	if got := InterruptReasonFromError(ErrStreamIncomplete); got != "no_done" {
		t.Fatalf("got %s", got)
	}
}
