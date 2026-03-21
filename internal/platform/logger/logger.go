// Package logger 提供基于 log/slog 的结构化日志，作为 platform 层日志入口。
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// LogLevel 日志级别，直接复用 slog.Level 的底层类型以便互操作。
type LogLevel = slog.Level

const (
	DEBUG LogLevel = slog.LevelDebug
	INFO  LogLevel = slog.LevelInfo
	WARN  LogLevel = slog.LevelWarn
	ERROR LogLevel = slog.LevelError
	FATAL LogLevel = slog.Level(12) // 高于 ERROR，用于致命错误
)

// LevelNames 级别名称映射，供外部查表使用。
var LevelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var (
	levelVar slog.LevelVar
	useJSON  bool
	output   io.Writer = os.Stdout
	exitFunc           = os.Exit
)

func init() {
	levelVar.Set(INFO)
	rebuildHandler()
}

// rebuildHandler 根据当前配置重建全局 slog handler。
func rebuildHandler() {
	opts := &slog.HandlerOptions{
		Level: &levelVar,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if level, ok := a.Value.Any().(slog.Level); ok && level == FATAL {
					a.Value = slog.StringValue("FATAL")
				}
			}
			return a
		},
	}
	var h slog.Handler
	if useJSON {
		h = slog.NewJSONHandler(output, opts)
	} else {
		h = slog.NewTextHandler(output, opts)
	}
	slog.SetDefault(slog.New(h))
}

// ParseLevel 解析日志级别字符串。
func ParseLevel(s string) LogLevel {
	switch strings.ToLower(s) {
	case "debug":
		return DEBUG
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return INFO
	}
}

// SetLevel 设置全局日志级别。
func SetLevel(level LogLevel) {
	levelVar.Set(level)
}

// Enabled 判断指定级别是否会被输出。
func Enabled(level LogLevel) bool {
	return slog.Default().Enabled(context.Background(), level)
}

// SetJSONOutput 切换 JSON / 文本输出格式。
func SetJSONOutput(enabled bool) {
	useJSON = enabled
	rebuildHandler()
}

// SetOutput 设置日志输出目标。传 nil 重置为 os.Stdout。
func SetOutput(writer io.Writer) {
	if writer == nil {
		writer = os.Stdout
	}
	output = writer
	rebuildHandler()
}

// Debug 输出调试日志。keysAndValues 为交替的 key-value 对。
func Debug(msg string, keysAndValues ...any) {
	slog.Debug(msg, keysAndValues...)
}

// Info 输出信息日志。
func Info(msg string, keysAndValues ...any) {
	slog.Info(msg, keysAndValues...)
}

// Warn 输出警告日志。
func Warn(msg string, keysAndValues ...any) {
	slog.Warn(msg, keysAndValues...)
}

// Error 输出错误日志。err 参数会作为 "error" 属性输出。
func Error(msg string, err error, keysAndValues ...any) {
	if err != nil {
		args := make([]any, 0, 2+len(keysAndValues))
		args = append(args, "error", err)
		args = append(args, keysAndValues...)
		slog.Error(msg, args...)
	} else {
		slog.Error(msg, keysAndValues...)
	}
}

// Fatal 输出致命错误日志后退出进程。
func Fatal(msg string, err error) {
	if err != nil {
		slog.Log(context.Background(), FATAL, msg, "error", err)
	} else {
		slog.Log(context.Background(), FATAL, msg)
	}
	exitFunc(1)
}

// Printf 兼容 log.Printf 风格，输出为 INFO 级别。
func Printf(format string, v ...any) {
	msg := format
	if len(v) != 0 {
		msg = fmt.Sprintf(format, v...)
	}
	slog.Log(context.Background(), INFO, msg)
}
