package cloudinary

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
)

// SetupRoutes настраивает маршруты для API объявлений
func (s *CloudinaryService) SetupRoutes(app *fiber.App) {
	// Группа для API объявлений
	api := app.Group("/api")

	// Защищенные маршруты
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(s.jwtService))

	// Маршрут для получения параметров загрузки
	protected.Get("/upload/params", s.GenerateUploadParams)

	// Другие маршруты для работы с объявлениями будут добавлены позже
}
