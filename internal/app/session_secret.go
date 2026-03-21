package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"log/slog"
	"os"
	"path/filepath"
)

func loadOrGenerateSecretKey() []byte {
	if key, ok := loadSecretKeyFromEnv(); ok {
		return key
	}

	keyFile := resolveSessionSecretFilePath()
	if key, ok := loadSecretKeyFromFile(keyFile); ok {
		return key
	}

	if err := os.MkdirAll(filepath.Dir(keyFile), 0755); err != nil {
		slog.Warn("Failed to create config directory", "error", err)
	}

	newKey := generateSessionSecretKey()

	if err := os.WriteFile(keyFile, newKey, 0600); err != nil {
		slog.Warn("Failed to save session secret key", "error", err)
	} else {
		slog.Info("Generated new session secret key")
	}

	return hashSessionSecret(newKey)
}

func loadSecretKeyFromEnv() ([]byte, bool) {
	key := os.Getenv("SESSION_SECRET")
	if key == "" {
		return nil, false
	}
	return hashSessionSecret([]byte(key)), true
}

func resolveSessionSecretFilePath() string {
	return filepath.Join("config", "session_secret.key")
}

func loadSecretKeyFromFile(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 32 {
		return nil, false
	}
	return hashSessionSecret(data), true
}

func generateSessionSecretKey() []byte {
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		slog.Log(context.Background(), slog.Level(12), "Failed to generate secret key", "error", err)
		os.Exit(1)
	}
	return newKey
}

func hashSessionSecret(key []byte) []byte {
	h := sha256.Sum256(key)
	return h[:]
}
