package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

var defaultLogger *slog.Logger

// RotateConfig 文件日志的轮转策略（对应 config.yaml 中 log.* 的一段）。
type RotateConfig struct {
	MaxSizeMB  int  // 单文件最大体积（MB），达到后轮转；<=0 用 lumberjack 默认 100
	MaxBackups int  // 保留的旧文件份数；0 表示保留全部（仍受 MaxAgeDays 限制）
	MaxAgeDays int  // 旧文件最大保留天数；0 表示不按天数清理
	Compress   bool // 是否 gzip 压缩归档旧文件
}

// DefaultRotateConfig 未显式配置时的默认轮转策略。
func DefaultRotateConfig() RotateConfig {
	return RotateConfig{
		MaxSizeMB:  100,
		MaxBackups: 14,
		MaxAgeDays: 30,
		Compress:   true,
	}
}

// newRotateFile 用 lumberjack 构建一个按策略轮转的文件 writer。
func newRotateFile(path string, rc RotateConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    rc.MaxSizeMB,
		MaxBackups: rc.MaxBackups,
		MaxAge:     rc.MaxAgeDays,
		LocalTime:  true, // 归档文件名时间戳用本地时区，便于排查
		Compress:   rc.Compress,
	}
}

func init() {
	// 包初始化时先用环境变量兜底（config 尚未加载），app 启动后再用 Configure 覆盖。
	ensureInitialized()
}

// ensureInitialized 仅在尚未初始化时根据环境变量初始化一次。
func ensureInitialized() {
	if defaultLogger != nil {
		return
	}
	initLogger(os.Getenv("LOG_DIR"), os.Getenv("LOG_LEVEL"), DefaultRotateConfig())
}

// Configure 用配置里的日志目录、级别和轮转策略（重新）初始化 logger。
// 环境变量 LOG_DIR / LOG_LEVEL / LOG_FORMAT 始终优先，便于部署脚本强制覆盖。
// dir 为空时回落到仅 stdout。
func Configure(dir, level string, rc RotateConfig) {
	initLogger(resolveDir(dir), resolveLevel(level), rc)
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

func initLogger(logDir, level string, rc RotateConfig) {
	defaultLogger = newLoggerWith(logDir, level, rc)
}

func newLoggerWith(logDir, level string, rc RotateConfig) *slog.Logger {
	lvl := parseLevel(level)
	if logDir == "" {
		return buildStdoutLogger(lvl)
	}
	return buildFileLogger(logDir, lvl, rc)
}

func buildStdoutLogger(level slog.Level) *slog.Logger {
	// 与 buildFileLogger 共用 createHandler，避免 JSON/Text 选择逻辑在多处重复。
	return slog.New(createHandler(os.Stdout, level))
}

func buildFileLogger(dir string, level slog.Level, rc RotateConfig) *slog.Logger {
	os.MkdirAll(dir, 0755)

	var targets []outputTarget

	stdoutHandler := createHandler(os.Stdout, level)
	targets = append(targets, outputTarget{
		handler: stdoutHandler,
		level:   slog.LevelInfo,
	})

	targets = append(targets, outputTarget{
		handler: createHandler(newRotateFile(filepath.Join(dir, "qim.log"), rc), slog.LevelDebug),
		level:   slog.LevelDebug,
	})

	targets = append(targets, outputTarget{
		handler: createHandler(newRotateFile(filepath.Join(dir, "error.log"), rc), slog.LevelError),
		level:   slog.LevelError,
	})

	targets = append(targets, outputTarget{
		handler: createHandler(newRotateFile(filepath.Join(dir, "auth.log"), rc), slog.LevelDebug),
		level:   slog.LevelDebug,
		module:  "auth",
	})

	targets = append(targets, outputTarget{
		handler: createHandler(newRotateFile(filepath.Join(dir, "ai.log"), rc), slog.LevelDebug),
		level:   slog.LevelDebug,
		module:  "ai",
	})

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
	// 复用 createHandler，避免与 buildStdoutLogger 的格式选择逻辑漂移。
	defaultLogger = slog.New(createHandler(w, parseLevel(os.Getenv("LOG_LEVEL"))))
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
