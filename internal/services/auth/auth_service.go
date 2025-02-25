package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/config"
	"github.com/rajivgeraev/flippy-api/internal/utils"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

// AuthService – структура для обработки авторизации
type AuthService struct {
	cfg        *config.Config
	jwtService *utils.JWTService
}

// NewAuthService – конструктор AuthService
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		cfg:        cfg,
		jwtService: utils.NewJWTService(cfg.JWTSecret),
	}
}

// TelegramAuthHandler проверяет initData, создает JWT и возвращает его
func (s *AuthService) TelegramAuthHandler(c fiber.Ctx) error {
	var payload struct {
		InitData string `json:"init_data"`
	}

	if err := c.Bind().Body(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Проверяем initData
	expiration := 24 * time.Hour
	if err := initdata.Validate(payload.InitData, s.cfg.TelegramBotToken, expiration); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid Telegram data"})
	}

	// Парсим данные
	data, err := initdata.Parse(payload.InitData)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse initData"})
	}

	// Генерируем JWT
	jwtToken, err := s.jwtService.GenerateToken(data.User.ID)
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
