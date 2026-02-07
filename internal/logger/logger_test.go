// =============================================================================
// 日志模块单元测试
// =============================================================================
package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"Debug", DEBUG},
		{"info", INFO},
		{"INFO", INFO},
		{"warn", WARN},
		{"WARN", WARN},
		{"warning", WARN},
		{"WARNING", WARN},
		{"error", ERROR},
		{"ERROR", ERROR},
		{"fatal", FATAL},
		{"FATAL", FATAL},
		{"unknown", INFO}, // 默认值
		{"", INFO},        // 默认值
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLevelNames(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := LevelNames[tt.level]
			if result != tt.expected {
				t.Errorf("LevelNames[%d] = %s, want %s", tt.level, result, tt.expected)
			}
		})
	}
}

func TestLevelNames_Unknown(t *testing.T) {
	unknown := LogLevel(100)
	result := LevelNames[unknown]
	if result != "" {
		t.Errorf("Unknown level returned %s, want empty string", result)
	}
}

func TestNewStructuredLogger(t *testing.T) {
	logger := NewStructuredLogger(INFO, "test", false)

	if logger == nil {
		t.Fatal("NewStructuredLogger returned nil")
	}

	if logger.level != INFO {
		t.Errorf("level = %v, want %v", logger.level, INFO)
	}
	if logger.module != "test" {
		t.Errorf("module = %s, want 'test'", logger.module)
	}
	if logger.jsonOutput != false {
		t.Errorf("jsonOutput = %v, want false", logger.jsonOutput)
	}
	if logger.logger == nil {
		t.Error("logger is nil")
	}
}

func TestStructuredLogger_WithModule(t *testing.T) {
	original := NewStructuredLogger(DEBUG, "original", true)

	withModule := original.WithModule("new_module")

	if withModule.module != "new_module" {
		t.Errorf("module = %s, want 'new_module'", withModule.module)
	}

	// 验证原始 logger 不受影响
	if original.module != "original" {
		t.Errorf("original.module changed to %s", original.module)
	}
}

func TestStructuredLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(DEBUG, "test", false)
	logger.logger.SetOutput(&buf)

	logger.Debug("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Debug output = %s, want to contain 'test message'", output)
	}
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Debug output = %s, want to contain 'DEBUG'", output)
	}
}

func TestStructuredLogger_Debug_Suppressed(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", false)
	logger.logger.SetOutput(&buf)

	logger.Debug("test message")

	output := buf.String()
	if output != "" {
		t.Errorf("Debug output = %s, want empty (suppressed)", output)
	}
}

func TestStructuredLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", false)
	logger.logger.SetOutput(&buf)

	logger.Info("info message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "info message") {
		t.Errorf("Info output = %s, want to contain 'info message'", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Info output = %s, want to contain 'INFO'", output)
	}
}

func TestStructuredLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", false)
	logger.logger.SetOutput(&buf)

	logger.Warn("warning message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Errorf("Warn output = %s, want to contain 'warning message'", output)
	}
	if !strings.Contains(output, "WARN") {
		t.Errorf("Warn output = %s, want to contain 'WARN'", output)
	}
}

func TestStructuredLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", false)
	logger.logger.SetOutput(&buf)

	testErr := os.ErrNotExist
	logger.Error("error message", testErr, "key", "value")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("Error output = %s, want to contain 'error message'", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Error output = %s, want to contain 'ERROR'", output)
	}
	if !strings.Contains(output, "file does not exist") {
		t.Errorf("Error output should contain error description")
	}
}

func TestStructuredLogger_Error_NoError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", false)
	logger.logger.SetOutput(&buf)

	logger.Error("error message", nil, "key", "value")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("Error output = %s, want to contain 'error message'", output)
	}
}

func TestStructuredLogger_Fatal(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", false)
	logger.logger.SetOutput(&buf)
	originalExit := exitFunc
	defer func() { exitFunc = originalExit }()

	exitCalled := false
	exitCode := 0
	exitFunc = func(code int) {
		exitCalled = true
		exitCode = code
	}

	testErr := os.ErrPermission
	logger.Fatal("fatal message", testErr)

	if !exitCalled {
		t.Fatal("expected exit function to be called")
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	output := buf.String()
	if !strings.Contains(output, "fatal message") {
		t.Errorf("Fatal output = %s, want to contain 'fatal message'", output)
	}
}

func TestParseKeyValues(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected map[string]interface{}
	}{
		{
			name:     "single pair",
			input:    []interface{}{"key1", "value1"},
			expected: map[string]interface{}{"key1": "value1"},
		},
		{
			name:     "multiple pairs",
			input:    []interface{}{"key1", "value1", "key2", 123, "key3", true},
			expected: map[string]interface{}{"key1": "value1", "key2": 123, "key3": true},
		},
		{
			name:     "empty",
			input:    []interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name:     "odd number of args",
			input:    []interface{}{"key1", "value1", "key2"},
			expected: map[string]interface{}{"key1": "value1"},
		},
		{
			name:     "non-string key ignored",
			input:    []interface{}{123, "value1", "key2", "value2"},
			expected: map[string]interface{}{"key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeyValues(tt.input...)
			if len(result) != len(tt.expected) {
				t.Errorf("parseKeyValues() returned %d entries, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("parseKeyValues()[%s] = %v, want %v", k, result[k], v)
				}
			}
		})
	}
}

func TestLogEntry_JSONFields(t *testing.T) {
	entry := &LogEntry{
		Level:     "INFO",
		Timestamp: "2024-01-01T00:00:00Z",
		Message:   "test message",
		Module:    "test",
		Caller:    "main.test",
		Error:     "error",
		Fields:    map[string]interface{}{"key": "value"},
	}

	// 验证可以序列化为 JSON
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal LogEntry: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal LogEntry: %v", err)
	}

	if result["level"] != "INFO" {
		t.Errorf("level = %v, want 'INFO'", result["level"])
	}
	if result["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", result["message"])
	}
}

func TestSetLevel(t *testing.T) {
	originalLevel := global.level

	SetLevel(DEBUG)
	if global.level != DEBUG {
		t.Errorf("global.level = %v, want %v", global.level, DEBUG)
	}

	// 恢复
	SetLevel(originalLevel)
}

func TestSetJSONOutput(t *testing.T) {
	original := global.jsonOutput

	SetJSONOutput(true)
	if global.jsonOutput != true {
		t.Errorf("global.jsonOutput = %v, want true", global.jsonOutput)
	}

	SetJSONOutput(false)
	if global.jsonOutput != false {
		t.Errorf("global.jsonOutput = %v, want false", global.jsonOutput)
	}

	// 恢复
	SetJSONOutput(original)
}

func TestSetOutput(t *testing.T) {
	originalOutput := Output()
	t.Cleanup(func() {
		SetOutput(originalOutput)
	})

	var buf bytes.Buffer
	SetOutput(&buf)

	if global.logger.Writer() != &buf {
		t.Error("global.logger writer not updated")
	}
}

func TestOutput(t *testing.T) {
	SetOutput(os.Stdout)

	output := Output()
	if output != os.Stdout {
		t.Errorf("Output() = %v, want os.Stdout", output)
	}
}

func TestGlobalFunctions(t *testing.T) {
	originalOutput := Output()
	originalLevel := global.level
	t.Cleanup(func() {
		SetOutput(originalOutput)
		SetLevel(originalLevel)
	})

	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(DEBUG)

	Debug("debug msg")
	if !strings.Contains(buf.String(), "debug msg") {
		t.Error("Debug() not working")
	}
	buf.Reset()

	Info("info msg")
	if !strings.Contains(buf.String(), "info msg") {
		t.Error("Info() not working")
	}
	buf.Reset()

	Warn("warn msg")
	if !strings.Contains(buf.String(), "warn msg") {
		t.Error("Warn() not working")
	}
	buf.Reset()

	Error("error msg", nil)
	if !strings.Contains(buf.String(), "error msg") {
		t.Error("Error() not working")
	}
}

func TestPrintf(t *testing.T) {
	originalOutput := Output()
	t.Cleanup(func() {
		SetOutput(originalOutput)
	})

	var buf bytes.Buffer
	SetOutput(&buf)

	Printf("test %s %d", "format", 123)

	output := buf.String()
	if !strings.Contains(output, "test format 123") {
		t.Errorf("Printf() output = %s", output)
	}
}

func TestPrintln(t *testing.T) {
	originalOutput := Output()
	t.Cleanup(func() {
		SetOutput(originalOutput)
	})

	var buf bytes.Buffer
	SetOutput(&buf)

	Println("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Println() output = %s", output)
	}
}

func TestStructuredLogger_Output_JSON(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", true)
	logger.logger.SetOutput(&buf)

	logger.Info("test message", "key", "value")

	output := buf.String()

	// 验证是有效的 JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %s, error: %v", output, err)
	}

	if result["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", result["message"])
	}
}

func TestStructuredLogger_Output_Text(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(INFO, "test", false)
	logger.logger.SetOutput(&buf)

	logger.Info("test message", "key", "value")

	output := buf.String()

	// 验证是文本格式，包含 JSON 字段部分
	if !strings.Contains(output, "test message") {
		t.Errorf("Output doesn't contain message: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Output doesn't contain level: %s", output)
	}
}

func TestNewStructuredLogger_JSONOutput(t *testing.T) {
	logger := NewStructuredLogger(INFO, "test", true)

	if logger.jsonOutput != true {
		t.Errorf("jsonOutput = %v, want true", logger.jsonOutput)
	}
}
