package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims JWT令牌声明
type TokenClaims struct {
	UserID   uint     `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secretKey   string
	tokenExpiry time.Duration
}

// NewJWTManager 创建新的JWT管理器
func NewJWTManager(secretKey string, tokenExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:   secretKey,
		tokenExpiry: tokenExpiry,
	}
}

// GenerateToken 生成JWT令牌
func (j *JWTManager) GenerateToken(userID uint, username string, roles []string) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "nmp-platform",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ValidateToken 验证JWT令牌
func (j *JWTManager) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken 刷新JWT令牌
func (j *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// 检查令牌是否即将过期（在过期前30分钟内可以刷新）
	// 如果剩余时间超过30分钟，不需要刷新
	if time.Until(claims.ExpiresAt.Time) > 30*time.Minute {
		return "", errors.New("token is not eligible for refresh yet")
	}

	// 生成新令牌
	return j.GenerateToken(claims.UserID, claims.Username, claims.Roles)
}