package listing

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
)

// SetupRoutes настраивает маршруты для API объявлений
func (s *ListingService) SetupRoutes(app *fiber.App) {
	// Группа для API объявлений
	api := app.Group("/api/listings")

	// Защищенные маршруты
	api.Use(middleware.AuthMiddleware(s.jwtService))

	// Маршрут для создания объявления
	api.Post("/create", s.CreateListing)
}
