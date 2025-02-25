package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

// TelegramAuthHandler проверяет initData, создает JWT и возвращает его
func TelegramAuthHandler(c fiber.Ctx) error {
	var payload struct {
		InitData string `json:"init_data"`
	}

	if err := c.Bind().Body(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Получаем токен бота из переменных окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Bot token not configured"})
	}

	// Проверяем initData
	expiration := 24 * time.Hour
	if err := initdata.Validate(payload.InitData, botToken, expiration); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid Telegram data"})
	}

	// Парсим данные
	data, err := initdata.Parse(payload.InitData)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse initData"})
	}

	// Генерируем JWT
	jwtToken, err := generateJWT(data.User.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate JWT"})
	}

	return c.JSON(fiber.Map{
		"token": jwtToken,
		"user": fiber.Map{
			"id":         data.User.ID,
			"first_name": data.User.FirstName,
			"last_name":  data.User.LastName,
			"username":   data.User.Username,
			"photo_url":  data.User.PhotoURL,
		},
	})
}

// generateJWT создает JWT токен для пользователя
func generateJWT(userID int64) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is not set")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
