package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/config"

	// "github.com/rajivgeraev/flippy-api/internal/db"
	"github.com/rajivgeraev/flippy-api/internal/services/auth"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "Flippy (MVP)",
	})

	// Загружаем конфигурацию
	cfg := config.LoadConfig()

	// Инициализируем базу данных
	// db.InitDB()

	// Создаём AuthService и регистрируем маршруты
	authService := auth.NewAuthService(cfg)
	authService.SetupRoutes(app)

	log.Fatal(app.Listen(":8080"))
}
