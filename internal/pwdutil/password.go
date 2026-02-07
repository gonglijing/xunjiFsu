package pwdutil

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const (
	// Cost 密码哈希成本（值越大越安全，但速度越慢）
	// 12是默认值，14提供更好的安全性
	// 生产环境建议使用12-14
	Cost = 12
)

func normalizePassword(password string) []byte {
	passwordBytes := []byte(password)
	if len(passwordBytes) <= 72 {
		return passwordBytes
	}

	// bcrypt 仅使用前 72 字节，超长密码先做一次稳定哈希，
	// 避免不同长密码因前缀相同产生碰撞。
	digest := sha256.Sum256(passwordBytes)
	return []byte("sha256:" + hex.EncodeToString(digest[:]))
}

func legacyTruncatedPassword(password string) []byte {
	passwordBytes := []byte(password)
	if len(passwordBytes) > 72 {
		return passwordBytes[:72]
	}
	return passwordBytes
}

// Hash 生成密码哈希
func Hash(password string) string {
	hash, err := bcrypt.GenerateFromPassword(normalizePassword(password), Cost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// Compare 比较密码和哈希
func Compare(password, hash string) bool {
	if bcrypt.CompareHashAndPassword([]byte(hash), normalizePassword(password)) == nil {
		return true
	}

	// 兼容历史版本：旧逻辑会截断前 72 字节。
	if len([]byte(password)) > 72 {
		return bcrypt.CompareHashAndPassword([]byte(hash), legacyTruncatedPassword(password)) == nil
	}

	return false
}

// NeedsRehash 检查密码哈希是否需要重新生成
// 当Cost改变时返回true
func NeedsRehash(hash string) bool {
	// 解析现有的哈希成本
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true // 无法解析，视为需要重新生成
	}
	return cost < Cost
}
