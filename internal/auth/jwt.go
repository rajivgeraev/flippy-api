package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rajivgeraev/flippy-api/internal/config"
)

func GenerateJWT(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(), // 3 дня
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}
