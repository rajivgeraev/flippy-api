package trade

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
)

// SetupRoutes настраивает маршруты для API обменов
func (s *TradeService) SetupRoutes(app *fiber.App) {
	// Группа для API обменов
	api := app.Group("/api/trades")

	// Защищенные маршруты (требуют авторизации)
	api.Use(middleware.AuthMiddleware(s.jwtService))

	// Маршрут для создания предложения обмена
	api.Post("/", s.CreateTrade)

	// Маршрут для получения списка предложений обмена
	api.Get("/", s.GetMyTrades)

	// Маршрут для обновления статуса предложения обмена
	api.Put("/:id/status", s.UpdateTradeStatus)
}
