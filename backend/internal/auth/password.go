package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost bcrypt默认成本
	DefaultCost = bcrypt.DefaultCost
)

// PasswordManager 密码管理器
type PasswordManager struct {
	cost int
}

// NewPasswordManager 创建新的密码管理器
func NewPasswordManager() *PasswordManager {
	return &PasswordManager{
		cost: DefaultCost,
	}
}

// HashPassword 加密密码
func (p *PasswordManager) HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// VerifyPassword 验证密码
func (p *PasswordManager) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// IsValidPassword 检查密码强度
func (p *PasswordManager) IsValidPassword(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters long")
	}

	// 可以添加更多密码强度检查规则
	// 例如：必须包含大小写字母、数字、特殊字符等

	return nil
}