package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type outputTarget struct {
	handler slog.Handler
	level   slog.Level
	module  string
}

type multiHandler struct {
	targets []outputTarget
}

func newMultiHandler(targets []outputTarget) *multiHandler {
	return &multiHandler{targets: targets}
}

func (h *multiHandler) Enabled(_ context.Context, level slog.Level) bool {
	for _, t := range h.targets {
		if level >= t.level {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var module string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "module" {
			module = a.Value.String()
			return false
		}
		return true
	})

	for _, t := range h.targets {
		if r.Level < t.level {
			continue
		}
		if t.module != "" && !strings.EqualFold(t.module, module) {
			continue
		}
		if err := t.handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	targets := make([]outputTarget, len(h.targets))
	for i, t := range h.targets {
		targets[i] = outputTarget{
			handler: t.handler.WithAttrs(attrs),
			level:   t.level,
			module:  t.module,
		}
	}
	return &multiHandler{targets: targets}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	targets := make([]outputTarget, len(h.targets))
	for i, t := range h.targets {
		targets[i] = outputTarget{
			handler: t.handler.WithGroup(name),
			level:   t.level,
			module:  t.module,
		}
	}
	return &multiHandler{targets: targets}
}

func createHandler(w io.Writer, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{Level: level}
	if os.Getenv("LOG_FORMAT") == "json" {
		return slog.NewJSONHandler(w, opts)
	}
	return NewTextHandler(w, opts)
}
