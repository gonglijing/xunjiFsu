// =============================================================================
// 密码工具模块单元测试
// =============================================================================
package pwdutil

import (
	"testing"
)

func TestHash(t *testing.T) {
	password := "testPassword123"

	hash := Hash(password)

	// 验证哈希不为空
	if hash == "" {
		t.Error("Hash() returned empty string")
	}

	// 验证哈希长度（bcrypt 哈希长度大约 60）
	if len(hash) < 50 {
		t.Errorf("Hash() length = %d, want at least 50", len(hash))
	}

	// 验证同一密码产生不同哈希（salt）
	hash2 := Hash(password)
	if hash == hash2 {
		t.Error("Same password should produce different hashes due to salt")
	}
}

func TestHash_DifferentPasswords(t *testing.T) {
	password1 := "password1"
	password2 := "password2"

	hash1 := Hash(password1)
	hash2 := Hash(password2)

	if hash1 == hash2 {
		t.Error("Different passwords should produce different hashes")
	}
}

func TestHash_EmptyPassword(t *testing.T) {
	hash := Hash("")

	// 空密码也应该能生成哈希
	if hash == "" {
		t.Error("Hash() for empty password returned empty string")
	}
}

func TestCompare_Valid(t *testing.T) {
	password := "mySecurePassword123"
	hash := Hash(password)

	result := Compare(password, hash)

	if !result {
		t.Error("Compare() for correct password returned false")
	}
}

func TestCompare_Invalid(t *testing.T) {
	password := "mySecurePassword123"
	wrongPassword := "wrongPassword"
	hash := Hash(password)

	result := Compare(wrongPassword, hash)

	if result {
		t.Error("Compare() for wrong password returned true")
	}
}

func TestCompare_EmptyPassword(t *testing.T) {
	password := "mySecurePassword123"
	hash := Hash(password)

	// 空密码应该不匹配
	result := Compare("", hash)

	if result {
		t.Error("Compare() for empty password returned true")
	}
}

func TestCompare_WrongHash(t *testing.T) {
	password := "mySecurePassword123"
	wrongHash := Hash("differentPassword")

	result := Compare(password, wrongHash)

	if result {
		t.Error("Compare() for wrong hash returned true")
	}
}

func TestCompare_CaseSensitive(t *testing.T) {
	password := "Password123"
	hash := Hash(password)

	// 大小写不同应该不匹配
	result := Compare("password123", hash)

	if result {
		t.Error("Compare() should be case-sensitive")
	}

	result = Compare("PASSWORD123", hash)

	if result {
		t.Error("Compare() should be case-sensitive")
	}
}

func TestHash_ProducesBCryptHash(t *testing.T) {
	password := "testPassword"
	hash := Hash(password)

	// 验证是 bcrypt 格式（以 $2a$ 或 $2b$ 开头）
	if hash[:4] != "$2a$" && hash[:4] != "$2b$" {
		t.Errorf("Hash() doesn't look like bcrypt format: %s", hash[:4])
	}
}

func TestCompare_MultiplePasswords(t *testing.T) {
	passwords := []string{
		"simple",
		"complex!@#$%^&*()",
		"中文密码",
		"123456",
		"a",
		"very-long-password-that-exceeds-normal-length-requirements-for-testing-purposes",
	}

	for _, password := range passwords {
		t.Run(password[:min(len(password), 20)], func(t *testing.T) {
			hash := Hash(password)

			// 正确密码应该匹配
			if !Compare(password, hash) {
				t.Errorf("Compare() failed for password: %s", password)
			}

			// 错误密码不应该匹配
			if Compare(password+"wrong", hash) {
				t.Errorf("Compare() matched wrong password for: %s", password)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
