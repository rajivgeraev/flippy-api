package listing

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/services/cloudinary"
)

// SetupRoutes настраивает маршруты для API объявлений
func SetupRoutes(app *fiber.App, authMiddleware fiber.Handler, cloudinaryService *cloudinary.CloudinaryService) {
	// Группа для API объявлений
	api := app.Group("/api")

	// Публичные маршруты
	// Будут добавлены позже

	// Защищенные маршруты
	protected := api.Group("/")
	protected.Use(authMiddleware)

	// Маршрут для получения параметров загрузки
	protected.Get("/upload/params", cloudinaryService.GenerateUploadParams)

	// Другие маршруты для работы с объявлениями будут добавлены позже
}
