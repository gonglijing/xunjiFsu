package pwdutil

import "golang.org/x/crypto/bcrypt"

const (
	// Cost 密码哈希成本（值越大越安全，但速度越慢）
	// 12是默认值，14提供更好的安全性
	// 生产环境建议使用12-14
	Cost = 12
)

// Hash 生成密码哈希
func Hash(password string) string {
	// bcrypt 只使用前 72 字节，超过部分会被忽略，这里显式截断避免 panic
	pwBytes := []byte(password)
	if len(pwBytes) > 72 {
		pwBytes = pwBytes[:72]
	}

	hash, err := bcrypt.GenerateFromPassword(pwBytes, Cost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

// Compare 比较密码和哈希
func Compare(password, hash string) bool {
	// 与 Hash 保持一致，对密码进行相同的截断处理
	pwBytes := []byte(password)
	if len(pwBytes) > 72 {
		pwBytes = pwBytes[:72]
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), pwBytes)
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
