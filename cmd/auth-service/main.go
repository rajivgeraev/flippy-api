package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rajivgeraev/flippy-api/internal/config"
	"github.com/rajivgeraev/flippy-api/internal/db"
	"github.com/rajivgeraev/flippy-api/internal/middleware"
	"github.com/rajivgeraev/flippy-api/internal/services/auth"
	"github.com/rajivgeraev/flippy-api/internal/services/cloudinary"
	"github.com/rajivgeraev/flippy-api/internal/services/listing"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.LoadConfig()

	// Инициализируем базу данных
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("❌ Ошибка при инициализации базы данных: %v", err)
	}
	defer db.CloseDB()

	// Создаём экземпляр Fiber
	app := fiber.New(fiber.Config{
		AppName:      "Flippy API (MVP)",
		ErrorHandler: errorHandler,
	})

	// Добавляем middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: false,
	}))

	// Создаём сервисы
	authService := auth.NewAuthService(cfg)
	cloudinaryService := cloudinary.NewCloudinaryService(cfg)

	// Настраиваем middleware для аутентификации
	authMiddleware := middleware.AuthMiddleware(authService.GetJWTService())

	// Регистрируем маршруты
	authService.SetupRoutes(app)
	listing.SetupRoutes(app, authMiddleware, cloudinaryService)

	// Запускаем сервер
	log.Println("✅ Flippy API запущен на порту 8080")
	log.Fatal(app.Listen(":8080"))
}

// errorHandler обрабатывает ошибки Fiber
func errorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	// Проверяем, является ли ошибка из Fiber
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Отправляем ошибку в JSON
	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}
