package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
)

// SetupRoutes регистрирует маршруты в Fiber
func (s *AuthService) SetupRoutes(app *fiber.App) {
	// Основной маршрут для аутентификации через Telegram
	app.Post("/api/auth/telegram", s.TelegramAuthHandler)

	// Добавляем тестовые маршруты только для разработки
	if s.cfg.AppEnv == "development" {
		app.Post("/api/auth/test-login", s.TestLoginHandler)
	}

	// Защищенные маршруты
	protected := app.Group("/api")
	protected.Use(middleware.AuthMiddleware(s.jwtService))

	// Добавляем эндпоинт профиля
	protected.Get("/profile", func(c fiber.Ctx) error {
		userID := c.Locals("userID").(string)
		return c.JSON(fiber.Map{
			"message":   "Это защищенные данные для пользователя " + userID,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})
}
