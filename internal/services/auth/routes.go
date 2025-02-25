package auth

import (
	"github.com/gofiber/fiber/v3"
)

// SetupRoutes регистрирует маршруты в Fiber
func (s *AuthService) SetupRoutes(app *fiber.App) {
	app.Post("/api/auth/telegram", s.TelegramAuthHandler)
}
