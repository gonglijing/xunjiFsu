package pwdutil

import (
	"golang.org/x/crypto/bcrypt"
)

// Hash 生成密码哈希
func Hash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
