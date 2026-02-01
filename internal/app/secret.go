package app

import (
	"crypto/rand"
	"crypto/sha256"
	"os"
	"path/filepath"

	"github.com/gonglijing/xunjiFsu/internal/logger"
)

func loadOrGenerateSecretKey() []byte {
	if key := os.Getenv("SESSION_SECRET"); key != "" {
		h := sha256.Sum256([]byte(key))
		return h[:]
	}

	configDir := "config"
	keyFile := filepath.Join(configDir, "session_secret.key")
	if data, err := os.ReadFile(keyFile); err == nil {
		key := string(data)
		if len(key) >= 32 {
			h := sha256.Sum256([]byte(key))
			return h[:]
		}
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Warn("Failed to create config directory", "error", err)
	}

	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		logger.Fatal("Failed to generate secret key", err)
	}

	if err := os.WriteFile(keyFile, newKey, 0600); err != nil {
		logger.Warn("Failed to save session secret key", "error", err)
	} else {
		logger.Info("Generated new session secret key")
	}

	h := sha256.Sum256(newKey)
	return h[:]
}
