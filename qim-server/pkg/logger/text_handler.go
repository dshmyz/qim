package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
)

type TextHandler struct {
	opts  slog.HandlerOptions
	attrs []slog.Attr
	mu    sync.Mutex
	w     io.Writer
}

func NewTextHandler(w io.Writer, opts *slog.HandlerOptions) *TextHandler {
	h := &TextHandler{w: w}
	if opts != nil {
		h.opts = *opts
	}
	return h
}

func (h *TextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *TextHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	timeStr := r.Time.Format("15:04:05.000")

	levelStr := levelToString(r.Level)

	module := ""
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "module" {
			module = a.Value.String()
			return false
		}
		return true
	})

	caller := ""
	if h.opts.AddSource && r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		if frame, ok := frames.Next(); ok {
			caller = fmt.Sprintf("%s:%d", shortFile(frame.File), frame.Line)
		}
	}

	var attrsStr string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "module" {
			return true
		}
		attrsStr += fmt.Sprintf(" %s=%v", a.Key, a.Value.Any())
		return true
	})

	parts := ""
	if module != "" {
		parts = fmt.Sprintf("%s [%s] %s%s %s", timeStr, levelStr, module, caller, r.Message)
	} else {
		parts = fmt.Sprintf("%s [%s]%s %s", timeStr, levelStr, caller, r.Message)
	}

	_, err := fmt.Fprintln(h.w, parts+attrsStr)
	return err
}

func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)
	return h2
}

func (h *TextHandler) WithGroup(_ string) slog.Handler {
	return h.clone()
}

func (h *TextHandler) clone() *TextHandler {
	return &TextHandler{
		w:     h.w,
		opts:  h.opts,
		attrs: append([]slog.Attr{}, h.attrs...),
	}
}

func levelToString(l slog.Level) string {
	switch l {
	case slog.LevelDebug:
		return "DEBG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERRO"
	default:
		return l.String()
	}
}

func shortFile(file string) string {
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			return file[i+1:]
		}
	}
	return file
}
