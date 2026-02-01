package pwdutil

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// Cost 密码哈希成本（值越大越安全，但速度越慢）
	// 12是默认值，14提供更好的安全性
	// 生产环境建议使用12-14
	Cost = 12
)

// Hash 生成密码哈希
func Hash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), Cost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// Compare 比较密码和哈希
func Compare(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
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
