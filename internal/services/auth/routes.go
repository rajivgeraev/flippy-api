package auth

import "github.com/gofiber/fiber/v3"

func SetupRoutes(app *fiber.App) {
	app.Post("/api/auth/telegram", TelegramAuthHandler)
}
