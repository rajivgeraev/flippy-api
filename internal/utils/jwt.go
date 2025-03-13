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
func (s *JWTService) GenerateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
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

// ExtractUserID извлекает ID пользователя из токена
func (s *JWTService) ExtractUserID(tokenString string) (string, error) {
	token, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	return userID, nil
}
