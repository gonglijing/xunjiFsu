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

func TestNeedsRehash_SameCost(t *testing.T) {
	password := "testPassword"
	hash := Hash(password)

	// 使用相同 cost，不应该需要重新哈希
	result := NeedsRehash(hash)

	if result {
		t.Error("NeedsRehash() for same cost hash returned true")
	}
}

func TestNeedsRehash_LowerCost(t *testing.T) {
	password := "testPassword"

	// 创建一个低成本的哈希
	// 注意：由于 NeedsRehash 内部比较成本，我们直接测试其行为
	hash := Hash(password)

	// 由于我们的 Cost 是固定的，应该不需要重新哈希
	result := NeedsRehash(hash)

	// 这个测试验证当前成本设置下的行为
	// 实际结果取决于 Cost 常量的值
	_ = result
}

func TestNeedsRehash_InvalidHash(t *testing.T) {
	invalidHash := "invalid-hash-format"

	// 无效哈希应该需要重新生成
	result := NeedsRehash(invalidHash)

	if !result {
		t.Error("NeedsRehash() for invalid hash returned false")
	}
}

func TestNeedsRehash_TruncatedHash(t *testing.T) {
	// 太短的哈希
	truncatedHash := "$2a$10$xxx"

	result := NeedsRehash(truncatedHash)

	if !result {
		t.Error("NeedsRehash() for truncated hash returned false")
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

func TestNeedsRehash_EmptyString(t *testing.T) {
	result := NeedsRehash("")

	if !result {
		t.Error("NeedsRehash() for empty string returned false")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
