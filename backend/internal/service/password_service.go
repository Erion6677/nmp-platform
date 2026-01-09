package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// PasswordService 密码哈希服务
// 使用 bcrypt 算法进行密码哈希和验证
type PasswordService struct {
	cost int
}

// NewPasswordService 创建密码服务实例
func NewPasswordService() *PasswordService {
	return &PasswordService{cost: bcrypt.DefaultCost}
}

// NewPasswordServiceWithCost 创建指定 cost 的密码服务实例
// cost 值越高，哈希计算越慢，安全性越高
// 推荐值：10-14，默认值为 10
func NewPasswordServiceWithCost(cost int) (*PasswordService, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, errors.New("cost must be between 4 and 31")
	}
	return &PasswordService{cost: cost}, nil
}

// Hash 对密码进行哈希
// 返回 bcrypt 哈希字符串（以 $2a$ 或 $2b$ 开头）
func (s *PasswordService) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify 验证密码是否匹配哈希
// password: 明文密码
// hash: bcrypt 哈希字符串
// 返回 true 表示密码匹配
func (s *PasswordService) Verify(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// IsBcryptHash 检查字符串是否是有效的 bcrypt 哈希
// bcrypt 哈希以 $2a$、$2b$ 或 $2y$ 开头
func IsBcryptHash(s string) bool {
	if len(s) < 4 {
		return false
	}
	// bcrypt 哈希格式: $2a$cost$salt+hash 或 $2b$cost$salt+hash
	return (s[0:4] == "$2a$" || s[0:4] == "$2b$" || s[0:4] == "$2y$") && len(s) == 60
}

// 全局密码服务实例
var globalPasswordService *PasswordService

// GetPasswordService 获取全局密码服务实例
func GetPasswordService() *PasswordService {
	if globalPasswordService == nil {
		globalPasswordService = NewPasswordService()
	}
	return globalPasswordService
}

// SetPasswordService 设置全局密码服务实例
func SetPasswordService(ps *PasswordService) {
	globalPasswordService = ps
}
