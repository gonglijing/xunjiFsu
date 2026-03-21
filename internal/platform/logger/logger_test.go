package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// saveAndRestore 保存当前配置并返回恢复函数。
func saveAndRestore(t *testing.T) {
	t.Helper()
	origLevel := levelVar.Level()
	origJSON := useJSON
	origOutput := output
	t.Cleanup(func() {
		SetLevel(origLevel)
		useJSON = origJSON
		SetOutput(origOutput)
	})
}

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
		{"unknown", INFO},
		{"", INFO},
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
	for level, name := range LevelNames {
		if name == "" {
			t.Errorf("LevelNames[%d] is empty", level)
		}
	}
	if LevelNames[DEBUG] != "DEBUG" {
		t.Errorf("LevelNames[DEBUG] = %q", LevelNames[DEBUG])
	}
	if LevelNames[FATAL] != "FATAL" {
		t.Errorf("LevelNames[FATAL] = %q", LevelNames[FATAL])
	}
}

func TestSetLevel_And_Enabled(t *testing.T) {
	saveAndRestore(t)

	SetLevel(INFO)
	if Enabled(DEBUG) {
		t.Fatal("DEBUG should not be enabled at INFO level")
	}
	if !Enabled(INFO) {
		t.Fatal("INFO should be enabled at INFO level")
	}
	if !Enabled(ERROR) {
		t.Fatal("ERROR should be enabled at INFO level")
	}

	SetLevel(DEBUG)
	if !Enabled(DEBUG) {
		t.Fatal("DEBUG should be enabled at DEBUG level")
	}
}

func TestSetJSONOutput(t *testing.T) {
	saveAndRestore(t)
	var buf bytes.Buffer
	SetOutput(&buf)

	SetJSONOutput(true)
	Info("json test")

	var parsed map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &parsed); err != nil {
		t.Fatalf("JSON output expected, got: %s", buf.String())
	}
	if parsed["msg"] != "json test" {
		t.Errorf("msg = %v", parsed["msg"])
	}
}

func TestSetOutput(t *testing.T) {
	saveAndRestore(t)

	var buf bytes.Buffer
	SetOutput(&buf)
	Info("output test")
	if !strings.Contains(buf.String(), "output test") {
		t.Errorf("SetOutput not effective, got: %s", buf.String())
	}
}

func TestSetOutput_Nil_ResetToStdout(t *testing.T) {
	saveAndRestore(t)
	SetOutput(nil)
}

func TestGlobalFunctions(t *testing.T) {
	saveAndRestore(t)
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(DEBUG)

	Debug("debug msg")
	if !strings.Contains(buf.String(), "debug msg") {
		t.Errorf("Debug() output: %s", buf.String())
	}
	buf.Reset()

	Info("info msg")
	if !strings.Contains(buf.String(), "info msg") {
		t.Errorf("Info() output: %s", buf.String())
	}
	buf.Reset()

	Warn("warn msg")
	if !strings.Contains(buf.String(), "warn msg") {
		t.Errorf("Warn() output: %s", buf.String())
	}
	buf.Reset()

	Error("error msg", nil)
	if !strings.Contains(buf.String(), "error msg") {
		t.Errorf("Error() output: %s", buf.String())
	}
}

func TestError_WithError(t *testing.T) {
	saveAndRestore(t)
	var buf bytes.Buffer
	SetOutput(&buf)

	Error("oops", errForTest("boom"), "key", "val")
	out := buf.String()
	if !strings.Contains(out, "oops") || !strings.Contains(out, "boom") {
		t.Errorf("Error() output: %s", out)
	}
}

func TestPrintf(t *testing.T) {
	saveAndRestore(t)
	var buf bytes.Buffer
	SetOutput(&buf)

	Printf("test %s %d", "format", 123)
	if !strings.Contains(buf.String(), "test format 123") {
		t.Errorf("Printf() output: %s", buf.String())
	}
}

func TestFatal_CallsExit(t *testing.T) {
	saveAndRestore(t)
	var buf bytes.Buffer
	SetOutput(&buf)

	exitCalled := false
	origExit := exitFunc
	exitFunc = func(code int) { exitCalled = true }
	t.Cleanup(func() { exitFunc = origExit })

	Fatal("fatal msg", errForTest("critical"))
	if !exitCalled {
		t.Fatal("Fatal() did not call exit")
	}
	if !strings.Contains(buf.String(), "fatal msg") {
		t.Errorf("Fatal() output: %s", buf.String())
	}
}

func TestTextOutput_ContainsLevel(t *testing.T) {
	saveAndRestore(t)
	var buf bytes.Buffer
	SetJSONOutput(false)
	SetOutput(&buf)

	Info("level check")
	if !strings.Contains(buf.String(), "INFO") {
		t.Errorf("text output should contain INFO, got: %s", buf.String())
	}
}

// errForTest 是一个简单的测试用 error 实现。
type errForTest string

func (e errForTest) Error() string { return string(e) }
