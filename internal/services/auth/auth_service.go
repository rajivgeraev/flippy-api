package auth

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rajivgeraev/flippy-api/internal/config"
	"github.com/rajivgeraev/flippy-api/internal/db"
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

// GetJWTService возвращает JWT сервис для использования в middleware
func (s *AuthService) GetJWTService() *utils.JWTService {
	return s.jwtService
}

// TelegramAuthHandler проверяет initData, создает или обновляет пользователя и возвращает JWT
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

	// Сериализуем raw_data для хранения
	rawData, err := json.Marshal(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to serialize user data"})
	}

	// Создаем или обновляем пользователя
	username := data.User.Username
	if username == "" {
		username = "user_" + data.User.FirstName
	}

	// Используем поля напрямую из структуры User
	isPremium := data.User.IsPremium
	languageCode := data.User.LanguageCode

	user, err := db.CreateOrUpdateTelegramUser(
		data.User.ID,
		username,
		data.User.FirstName,
		data.User.LastName,
		data.User.PhotoURL,
		isPremium,
		languageCode,
		rawData,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create/update user"})
	}

	// Генерируем JWT
	userIDString := user.ID.String()
	jwtToken, err := s.jwtService.GenerateToken(userIDString)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate JWT"})
	}

	return c.JSON(fiber.Map{
		"token": jwtToken,
		"user": fiber.Map{
			"id":         userIDString,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"username":   user.Username,
			"avatar_url": user.AvatarURL,
		},
	})
}

// TestLoginHandler генерирует JWT для тестирования (только для разработки)
func (s *AuthService) TestLoginHandler(c fiber.Ctx) error {
	// Этот метод должен быть доступен только в режиме разработки
	if s.cfg.AppEnv != "development" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Route not found",
		})
	}

	var payload struct {
		UserID string `json:"user_id"`
	}

	if err := c.Bind().Body(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Проверяем, что userID является валидным UUID
	_, err := uuid.Parse(payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID format"})
	}

	// Генерируем JWT
	jwtToken, err := s.jwtService.GenerateToken(payload.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate JWT"})
	}

	return c.JSON(fiber.Map{
		"jwt_token": jwtToken,
		"user_id":   payload.UserID,
	})
}
