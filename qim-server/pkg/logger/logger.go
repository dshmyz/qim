package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var defaultLogger *slog.Logger

func init() {
	initLogger()
}

func initLogger() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	logDir := os.Getenv("LOG_DIR")

	if logDir == "" {
		initStdout(level)
		return
	}

	initFileOutput(logDir, level)
}

func initStdout(level slog.Level) {
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	defaultLogger = slog.New(handler)
}

func initFileOutput(dir string, level slog.Level) {
	os.MkdirAll(dir, 0755)

	var targets []outputTarget

	stdoutHandler := createHandler(os.Stdout, level)
	targets = append(targets, outputTarget{
		handler: stdoutHandler,
		level:   slog.LevelInfo,
	})

	if f, err := NewRotateFile(filepath.Join(dir, "qim.log")); err == nil {
		targets = append(targets, outputTarget{
			handler: createHandler(f, slog.LevelDebug),
			level:   slog.LevelDebug,
		})
	}

	if f, err := NewRotateFile(filepath.Join(dir, "error.log")); err == nil {
		targets = append(targets, outputTarget{
			handler: createHandler(f, slog.LevelError),
			level:   slog.LevelError,
		})
	}

	if f, err := NewRotateFile(filepath.Join(dir, "auth.log")); err == nil {
		targets = append(targets, outputTarget{
			handler: createHandler(f, slog.LevelDebug),
			level:   slog.LevelDebug,
			module:  "auth",
		})
	}

	defaultLogger = slog.New(newMultiHandler(targets))
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
	initStdout(parseLevel(os.Getenv("LOG_LEVEL")))
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
