package chat

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
)

// SetupRoutes настраивает маршруты для API чатов
func (s *ChatService) SetupRoutes(app *fiber.App) {
	// Группа для API чатов
	api := app.Group("/api/chats")

	// Защищенные маршруты (требуют авторизации)
	api.Use(middleware.AuthMiddleware(s.jwtService))

	// Маршрут для получения всех чатов пользователя
	api.Get("/", s.GetChats)

	// Маршрут для создания нового чата
	api.Post("/", s.CreateChat)

	// Маршрут для получения сообщений чата
	api.Get("/:id/messages", s.GetChatMessages)

	// Маршрут для отправки сообщения
	api.Post("/:id/messages", s.SendMessage)
}
