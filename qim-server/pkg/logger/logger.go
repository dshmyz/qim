package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

func init() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	var handler slog.Handler

	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	defaultLogger = slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func L() *slog.Logger {
	return defaultLogger
}

func SetOutput(w io.Writer) {
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: parseLevel(os.Getenv("LOG_LEVEL")),
		})
	} else {
		handler = NewTextHandler(w, &slog.HandlerOptions{
			Level: parseLevel(os.Getenv("LOG_LEVEL")),
		})
	}
	defaultLogger = slog.New(handler)
}

func WithModule(module string) *slog.Logger {
	return defaultLogger.With("module", module)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}