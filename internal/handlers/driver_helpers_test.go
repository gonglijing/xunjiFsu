package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvedDriversDir(t *testing.T) {
	h := &Handler{}
	if got := h.resolvedDriversDir(); got != "drivers" {
		t.Fatalf("resolvedDriversDir default = %q, want %q", got, "drivers")
	}

	h.driversDir = "custom-drivers"
	if got := h.resolvedDriversDir(); got != "custom-drivers" {
		t.Fatalf("resolvedDriversDir custom = %q, want %q", got, "custom-drivers")
	}
}

func TestIsWasmFileName(t *testing.T) {
	if !isWasmFileName("demo.wasm") {
		t.Fatal("demo.wasm should be recognized as wasm")
	}
	if !isWasmFileName("DEMO.WASM") {
		t.Fatal("DEMO.WASM should be recognized as wasm")
	}
	if isWasmFileName("demo.txt") {
		t.Fatal("demo.txt should not be recognized as wasm")
	}
}

func TestSaveDriverUploadFileAndListDriverWasmFiles(t *testing.T) {
	dir := t.TempDir()

	destPath, err := saveDriverUploadFile(dir, "abc.wasm", strings.NewReader("binary-data"))
	if err != nil {
		t.Fatalf("saveDriverUploadFile returned error: %v", err)
	}

	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("saved file stat failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write ignore file failed: %v", err)
	}

	files, err := listDriverWasmFiles(dir)
	if err != nil {
		t.Fatalf("listDriverWasmFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("files len = %d, want %d", len(files), 1)
	}

	if got := files[0]["name"]; got != "abc.wasm" {
		t.Fatalf("name = %v, want %q", got, "abc.wasm")
	}
}
