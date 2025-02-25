package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/services/auth"
)

func main() {
	app := fiber.New()

	// Инициализируем базу данных
	// db.InitDB()

	// Регистрируем маршруты авторизации
	auth.SetupRoutes(app)

	log.Fatal(app.Listen(":8080"))
}
