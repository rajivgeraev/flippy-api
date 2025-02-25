package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService отвечает за создание и валидацию JWT токенов
type JWTService struct {
	secretKey string
}

// NewJWTService создаёт новый экземпляр JWTService
func NewJWTService(secretKey string) *JWTService {
	return &JWTService{secretKey: secretKey}
}

// GenerateToken создаёт JWT токен
func (s *JWTService) GenerateToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ValidateToken проверяет JWT токен
func (s *JWTService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})
}
