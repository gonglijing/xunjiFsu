package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSecretKeyFromEnv(t *testing.T) {
	t.Setenv("SESSION_SECRET", "env-secret-value-123456789012345")

	key, ok := loadSecretKeyFromEnv()
	if !ok {
		t.Fatal("loadSecretKeyFromEnv() = false, want true")
	}
	if len(key) != 32 {
		t.Fatalf("len(key) = %d, want 32", len(key))
	}
	if got := hashSessionSecret([]byte("env-secret-value-123456789012345")); string(key) != string(got) {
		t.Fatal("loadSecretKeyFromEnv() returned unexpected hash")
	}
}

func TestLoadSecretKeyFromFile(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "session_secret.key")
	rawKey := []byte("file-secret-value-123456789012345")
	if err := os.WriteFile(keyFile, rawKey, 0600); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	key, ok := loadSecretKeyFromFile(keyFile)
	if !ok {
		t.Fatal("loadSecretKeyFromFile() = false, want true")
	}
	if got := hashSessionSecret(rawKey); string(key) != string(got) {
		t.Fatal("loadSecretKeyFromFile() returned unexpected hash")
	}
}

func TestLoadSecretKeyFromFile_RejectsShortKey(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "session_secret.key")
	if err := os.WriteFile(keyFile, []byte("short-key"), 0600); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	if _, ok := loadSecretKeyFromFile(keyFile); ok {
		t.Fatal("loadSecretKeyFromFile() = true, want false")
	}
}

func TestLoadOrGenerateSecretKey_PrefersEnvOverFile(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir() failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.MkdirAll("config", 0755); err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}
	if err := os.WriteFile(resolveSessionSecretFilePath(), []byte("file-secret-value-123456789012345"), 0600); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	t.Setenv("SESSION_SECRET", "env-secret-value-123456789012345")
	key := loadOrGenerateSecretKey()

	if got := hashSessionSecret([]byte("env-secret-value-123456789012345")); string(key) != string(got) {
		t.Fatal("loadOrGenerateSecretKey() did not prefer env secret")
	}
}
