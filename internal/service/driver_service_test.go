package service

import "testing"

func TestUnloadedDriverRuntime(t *testing.T) {
	runtime := unloadedDriverRuntime(42)
	if runtime == nil {
		t.Fatal("runtime is nil")
	}
	if runtime.ID != 42 {
		t.Fatalf("runtime.ID = %d", runtime.ID)
	}
	if runtime.Loaded {
		t.Fatal("runtime.Loaded = true, want false")
	}
}

func TestListDriverWasmFiles_MissingDir(t *testing.T) {
	_, err := ListDriverWasmFiles(t.TempDir() + "/missing")
	if err == nil {
		t.Fatal("expected error for missing dir")
	}
}
