package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestBuildUnloadedDriverRuntime(t *testing.T) {
	runtime := buildUnloadedDriverRuntime(7)

	if runtime["id"] != int64(7) {
		t.Fatalf("runtime[id] = %#v, want %#v", runtime["id"], int64(7))
	}
	if runtime["loaded"] != false {
		t.Fatalf("runtime[loaded] = %#v, want false", runtime["loaded"])
	}
}

func TestEnsureDriverFileExists(t *testing.T) {
	dir := t.TempDir()
	driverPath := filepath.Join(dir, "demo.wasm")
	if err := os.WriteFile(driverPath, []byte("wasm"), 0644); err != nil {
		t.Fatalf("write driver file: %v", err)
	}

	handler := &Handler{}
	driverModel := &models.Driver{Name: "demo", FilePath: driverPath}

	if !handler.ensureDriverFileExists(driverModel) {
		t.Fatal("expected ensureDriverFileExists to return true")
	}
}

func TestEnsureDriverFileExists_MissingFile(t *testing.T) {
	handler := &Handler{}
	driverModel := &models.Driver{Name: "missing", FilePath: filepath.Join(t.TempDir(), "missing.wasm")}

	if handler.ensureDriverFileExists(driverModel) {
		t.Fatal("expected ensureDriverFileExists to return false")
	}
}
