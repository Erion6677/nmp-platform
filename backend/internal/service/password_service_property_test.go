package service

import (
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// Feature: nmp-bugfix-iteration, Property 2: 密码哈希存储
// **Validates: Requirements 2.3, 5.4**
// *For any* 创建或更新用户密码的操作，数据库中存储的密码必须是 bcrypt 哈希值
// （以 `$2a$` 或 `$2b$` 开头），而非明文密码。

func TestPasswordHashStorageProperty(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	passwordService := NewPasswordService()

	// 属性1: 哈希后的密码必须以 $2a$ 或 $2b$ 开头
	properties.Property("hashed password should start with bcrypt prefix",
		prop.ForAll(
			func(password string) bool {
				if password == "" {
					return true // 跳过空密码
				}

				hash, err := passwordService.Hash(password)
				if err != nil {
					return false
				}

				// bcrypt 哈希必须以 $2a$、$2b$ 或 $2y$ 开头
				return strings.HasPrefix(hash, "$2a$") ||
					strings.HasPrefix(hash, "$2b$") ||
					strings.HasPrefix(hash, "$2y$")
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
		))

	// 属性2: 哈希后的密码长度必须是 60 字符
	properties.Property("hashed password should be exactly 60 characters",
		prop.ForAll(
			func(password string) bool {
				if password == "" {
					return true
				}

				hash, err := passwordService.Hash(password)
				if err != nil {
					return false
				}

				return len(hash) == 60
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
		))

	// 属性3: 哈希后的密码不等于原始密码
	properties.Property("hashed password should not equal original password",
		prop.ForAll(
			func(password string) bool {
				if password == "" {
					return true
				}

				hash, err := passwordService.Hash(password)
				if err != nil {
					return false
				}

				return hash != password
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
		))

	// 属性4: 相同密码的哈希值应该不同（因为 salt 不同）
	properties.Property("same password should produce different hashes",
		prop.ForAll(
			func(password string) bool {
				if password == "" {
					return true
				}

				hash1, err1 := passwordService.Hash(password)
				hash2, err2 := passwordService.Hash(password)

				if err1 != nil || err2 != nil {
					return false
				}

				// 两次哈希应该产生不同的结果（因为 salt 不同）
				return hash1 != hash2
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
		))

	// 属性5: 验证函数应该能正确验证哈希后的密码
	properties.Property("verify should return true for correct password",
		prop.ForAll(
			func(password string) bool {
				if password == "" {
					return true
				}

				hash, err := passwordService.Hash(password)
				if err != nil {
					return false
				}

				// 正确的密码应该验证通过
				return passwordService.Verify(password, hash)
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
		))

	// 属性6: 验证函数应该拒绝错误的密码
	properties.Property("verify should return false for wrong password",
		prop.ForAll(
			func(password, wrongPassword string) bool {
				if password == "" || wrongPassword == "" || password == wrongPassword {
					return true // 跳过无效输入
				}

				hash, err := passwordService.Hash(password)
				if err != nil {
					return false
				}

				// 错误的密码应该验证失败
				return !passwordService.Verify(wrongPassword, hash)
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
		))

	// 属性7: IsBcryptHash 应该正确识别 bcrypt 哈希
	properties.Property("IsBcryptHash should correctly identify bcrypt hashes",
		prop.ForAll(
			func(password string) bool {
				if password == "" {
					return true
				}

				hash, err := passwordService.Hash(password)
				if err != nil {
					return false
				}

				// 哈希后的值应该被识别为 bcrypt 哈希
				return IsBcryptHash(hash)
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 72 }),
		))

	// 属性8: IsBcryptHash 应该拒绝非 bcrypt 字符串
	properties.Property("IsBcryptHash should reject non-bcrypt strings",
		prop.ForAll(
			func(plaintext string) bool {
				// 普通字符串不应该被识别为 bcrypt 哈希
				// 除非它恰好符合 bcrypt 格式（极不可能）
				if strings.HasPrefix(plaintext, "$2a$") ||
					strings.HasPrefix(plaintext, "$2b$") ||
					strings.HasPrefix(plaintext, "$2y$") {
					return true // 跳过恰好以 bcrypt 前缀开头的字符串
				}
				return !IsBcryptHash(plaintext)
			},
			gen.AlphaString(),
		))
}

// 单元测试：验证具体场景
func TestPasswordServiceUnit(t *testing.T) {
	passwordService := NewPasswordService()

	t.Run("hash empty password should fail", func(t *testing.T) {
		_, err := passwordService.Hash("")
		assert.Error(t, err)
	})

	t.Run("verify with empty password should fail", func(t *testing.T) {
		result := passwordService.Verify("", "$2a$10$somevalidhash")
		assert.False(t, result)
	})

	t.Run("verify with empty hash should fail", func(t *testing.T) {
		result := passwordService.Verify("password", "")
		assert.False(t, result)
	})

	t.Run("hash and verify round trip", func(t *testing.T) {
		password := "testPassword123!"
		hash, err := passwordService.Hash(password)
		assert.NoError(t, err)
		assert.True(t, passwordService.Verify(password, hash))
	})

	t.Run("IsBcryptHash with valid hash", func(t *testing.T) {
		hash, _ := passwordService.Hash("test")
		assert.True(t, IsBcryptHash(hash))
	})

	t.Run("IsBcryptHash with invalid strings", func(t *testing.T) {
		assert.False(t, IsBcryptHash(""))
		assert.False(t, IsBcryptHash("plaintext"))
		assert.False(t, IsBcryptHash("$2a$")) // 太短
		assert.False(t, IsBcryptHash("notahash"))
	})
}

func TestPasswordServiceWithCustomCost(t *testing.T) {
	t.Run("create with valid cost", func(t *testing.T) {
		ps, err := NewPasswordServiceWithCost(12)
		assert.NoError(t, err)
		assert.NotNil(t, ps)
	})

	t.Run("create with invalid cost should fail", func(t *testing.T) {
		_, err := NewPasswordServiceWithCost(3) // 太低
		assert.Error(t, err)

		_, err = NewPasswordServiceWithCost(32) // 太高
		assert.Error(t, err)
	})
}
