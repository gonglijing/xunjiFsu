package driver

import (
	"runtime"
	"testing"
)

func TestBuildSttyArgs_Default(t *testing.T) {
	args, err := buildSttyArgs("/dev/ttyUSB0", serialOpenMode{BaudRate: 9600, DataBits: 8, Parity: "N", StopBits: 1})
	if err != nil {
		t.Fatalf("buildSttyArgs error: %v", err)
	}
	if len(args) < 7 {
		t.Fatalf("args too short: %v", args)
	}

	if runtime.GOOS == "darwin" {
		if args[0] != "-f" {
			t.Fatalf("expected -f on darwin, got %q", args[0])
		}
	} else {
		if args[0] != "-F" {
			t.Fatalf("expected -F on non-darwin, got %q", args[0])
		}
	}

	if args[1] != "/dev/ttyUSB0" {
		t.Fatalf("unexpected path arg: %v", args)
	}
	if args[2] != "9600" {
		t.Fatalf("unexpected baud arg: %v", args)
	}
}

func TestBuildSttyArgs_ParityAndStopBits(t *testing.T) {
	args, err := buildSttyArgs("/dev/ttyS1", serialOpenMode{BaudRate: 19200, DataBits: 7, Parity: "EVEN", StopBits: 2})
	if err != nil {
		t.Fatalf("buildSttyArgs error: %v", err)
	}

	has := func(needle string) bool {
		for _, arg := range args {
			if arg == needle {
				return true
			}
		}
		return false
	}

	if !has("19200") || !has("cs7") || !has("cstopb") || !has("parenb") || !has("-parodd") {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestBuildSttyArgs_EmptyPath(t *testing.T) {
	if _, err := buildSttyArgs("", serialOpenMode{}); err == nil {
		t.Fatal("expected error for empty path")
	}
}
