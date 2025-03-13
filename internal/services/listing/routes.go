package listing

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
)

// SetupRoutes настраивает маршруты для API объявлений
func (s *ListingService) SetupRoutes(app *fiber.App) {
	// Группа для API объявлений
	api := app.Group("/api/listings")

	// Защищенные маршруты (требуют авторизации)
	api.Use(middleware.AuthMiddleware(s.jwtService))

	// Маршрут для создания объявления
	api.Post("/create", s.CreateListing)

	// Маршрут для получения списка своих объявлений
	api.Get("/my", s.GetMyListings)

	// Маршрут для получения одного объявления по ID
	api.Get("/:id", s.GetListing)

	// Маршрут для обновления объявления
	api.Put("/:id", s.UpdateListing)

	// Маршрут для удаления объявления
	api.Delete("/:id", s.DeleteListing)
}

// SetupPublicRoutes настраивает публичные маршруты для листингов
func (s *ListingService) SetupPublicRoutes(app *fiber.App) {
	// Публичный маршрут для списка объявлений
	app.Get("/api/listings", s.GetPublicListings)
}
