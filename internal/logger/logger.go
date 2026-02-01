package logger

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// LevelNames 级别名称映射
var LevelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// ParseLevel 解析日志级别
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

// StructuredLogger 结构化日志
type StructuredLogger struct {
	level      LogLevel
	module     string
	jsonOutput bool
	logger     *log.Logger
}

// NewStructuredLogger 创建结构化日志
func NewStructuredLogger(level LogLevel, module string, jsonOutput bool) *StructuredLogger {
	return &StructuredLogger{
		level:      level,
		module:     module,
		jsonOutput: jsonOutput,
		logger:     log.New(os.Stdout, "", 0),
	}
}

// WithModule 创建带模块名的日志
func (l *StructuredLogger) WithModule(module string) *StructuredLogger {
	return &StructuredLogger{
		level:      l.level,
		module:     module,
		jsonOutput: l.jsonOutput,
		logger:     l.logger,
	}
}

// Debug 调试日志
func (l *StructuredLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.log(DEBUG, msg, keysAndValues...)
}

// Info 信息日志
func (l *StructuredLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log(INFO, msg, keysAndValues...)
}

// Warn 警告日志
func (l *StructuredLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.log(WARN, msg, keysAndValues...)
}

// Error 错误日志
func (l *StructuredLogger) Error(msg string, err error, keysAndValues ...interface{}) {
	entry := l.newEntry(ERROR, msg)
	if err != nil {
		entry.Error = err.Error()
	}
	if len(keysAndValues) > 0 {
		entry.Fields = parseKeyValues(keysAndValues...)
	}
	l.output(entry)
}

// Fatal 致命日志
func (l *StructuredLogger) Fatal(msg string, err error) {
	entry := l.newEntry(FATAL, msg)
	if err != nil {
		entry.Error = err.Error()
	}
	l.output(entry)
	os.Exit(1)
}

// log 内部日志方法
func (l *StructuredLogger) log(level LogLevel, msg string, keysAndValues ...interface{}) {
	if level < l.level {
		return
	}

	entry := l.newEntry(level, msg)
	if len(keysAndValues) > 0 {
		entry.Fields = parseKeyValues(keysAndValues...)
	}
	l.output(entry)
}

// newEntry 创建日志条目
func (l *StructuredLogger) newEntry(level LogLevel, msg string) *LogEntry {
	// 获取调用者信息
	pc, _, _, _ := runtime.Caller(2)
	caller := runtime.FuncForPC(pc).Name()

	return &LogEntry{
		Level:     LevelNames[level],
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   msg,
		Module:    l.module,
		Caller:    caller,
	}
}

// output 输出日志
func (l *StructuredLogger) output(entry *LogEntry) {
	if l.jsonOutput {
		data, _ := json.Marshal(entry)
		l.logger.Println(string(data))
	} else {
		fields := ""
		if len(entry.Fields) > 0 {
			data, _ := json.Marshal(entry.Fields)
			fields = " " + string(data)
		}
		l.logger.Printf("[%s] %s %s%s\n", entry.Level, entry.Timestamp, entry.Message, fields)
	}
}

// parseKeyValues 解析键值对
func parseKeyValues(keysAndValues ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		result[key] = keysAndValues[i+1]
	}
	return result
}

// LogEntry 日志条目
type LogEntry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	Module    string                 `json:"module,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// 全局logger
var global *StructuredLogger

func init() {
	global = NewStructuredLogger(INFO, "gogw", false)
}

// SetLevel 设置日志级别
func SetLevel(level LogLevel) {
	global.level = level
}

// SetJSONOutput 设置JSON输出
func SetJSONOutput(enabled bool) {
	global.jsonOutput = enabled
}

// Debug 全局调试日志
func Debug(msg string, keysAndValues ...interface{}) {
	global.log(DEBUG, msg, keysAndValues)
}

// Info 全局信息日志
func Info(msg string, keysAndValues ...interface{}) {
	global.log(INFO, msg, keysAndValues)
}

// Warn 全局警告日志
func Warn(msg string, keysAndValues ...interface{}) {
	global.log(WARN, msg, keysAndValues)
}

// Error 全局错误日志
func Error(msg string, err error, keysAndValues ...interface{}) {
	entry := global.newEntry(ERROR, msg)
	if err != nil {
		entry.Error = err.Error()
	}
	if len(keysAndValues) > 0 {
		entry.Fields = parseKeyValues(keysAndValues...)
	}
	global.output(entry)
}

// Fatal 全局致命日志
func Fatal(msg string, err error) {
	entry := global.newEntry(FATAL, msg)
	if err != nil {
		entry.Error = err.Error()
	}
	global.output(entry)
	os.Exit(1)
}

// Printf 格式化日志
func Printf(format string, v ...interface{}) {
	global.logger.Printf(format, v...)
}

// Println 日志
func Println(v ...interface{}) {
	global.logger.Println(v...)
}
