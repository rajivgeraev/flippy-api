package favorite

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
)

// SetupRoutes настраивает маршруты для API избранного
func (s *FavoriteService) SetupRoutes(app *fiber.App) {
	// Группа для API избранного
	api := app.Group("/api/favorites")

	// Защищенные маршруты (требуют авторизации)
	api.Use(middleware.AuthMiddleware(s.jwtService))

	// Маршрут для получения списка избранных объявлений
	api.Get("/", s.GetFavorites)

	// Маршрут для добавления объявления в избранное
	api.Post("/", s.AddToFavorites)

	// Маршрут для удаления объявления из избранного
	api.Delete("/:id", s.RemoveFromFavorites)

	// Маршрут для проверки, находится ли объявление в избранном
	api.Get("/:id/check", s.CheckFavorite)
}
