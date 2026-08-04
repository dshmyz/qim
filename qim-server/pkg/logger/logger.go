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
	// 包初始化时先用环境变量兜底（config 尚未加载），app 启动后再用 Configure 覆盖。
	ensureInitialized()
}

// ensureInitialized 仅在尚未初始化时根据环境变量初始化一次。
func ensureInitialized() {
	if defaultLogger != nil {
		return
	}
	initLogger(os.Getenv("LOG_DIR"), os.Getenv("LOG_LEVEL"))
}

// Configure 用配置里的日志目录和级别（重新）初始化 logger。
// 环境变量 LOG_DIR / LOG_LEVEL / LOG_FORMAT 始终优先，便于部署脚本强制覆盖。
// dir 为空时回落到仅 stdout。
func Configure(dir, level string) {
	initLogger(resolveDir(dir), resolveLevel(level))
}

// resolveDir / resolveLevel：环境变量优先，否则用传入的配置值。
func resolveDir(dir string) string {
	if v := os.Getenv("LOG_DIR"); v != "" {
		return v
	}
	return dir
}

func resolveLevel(level string) string {
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		return v
	}
	return level
}

func initLogger(logDir, level string) {
	defaultLogger = newLoggerWith(logDir, level)
}

func newLoggerWith(logDir, level string) *slog.Logger {
	lvl := parseLevel(level)
	if logDir == "" {
		return buildStdoutLogger(lvl)
	}
	return buildFileLogger(logDir, lvl)
}

func buildStdoutLogger(level slog.Level) *slog.Logger {
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	return slog.New(handler)
}

func buildFileLogger(dir string, level slog.Level) *slog.Logger {
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

	return slog.New(newMultiHandler(targets))
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

// SetOutput 将日志输出切换到指定 writer（stdout 单层），保留当前日志级别。
func SetOutput(w io.Writer) {
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))})
	} else {
		handler = NewTextHandler(w, &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))})
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
