package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rajivgeraev/flippy-api/internal/config"
	"github.com/rajivgeraev/flippy-api/internal/db"
	"github.com/rajivgeraev/flippy-api/internal/services/auth"
	"github.com/rajivgeraev/flippy-api/internal/services/chat"
	"github.com/rajivgeraev/flippy-api/internal/services/cloudinary"
	"github.com/rajivgeraev/flippy-api/internal/services/listing"
	"github.com/rajivgeraev/flippy-api/internal/services/trade"
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
	listingService := listing.NewListingService(cfg)
	tradeService := trade.NewTradeService(cfg)
	chatService := chat.NewChatService(cfg)

	// Вначале регистрируем публичные маршруты
	listingService.SetupPublicRoutes(app)
	// Временный эндпоинт для категорий
	app.Get("/api/categories", func(c fiber.Ctx) error {
		categories := []map[string]string{
			{"slug": "dolls", "name_ru": "Куклы", "name_en": "Dolls"},
			{"slug": "cars", "name_ru": "Машинки", "name_en": "Cars"},
			{"slug": "construction", "name_ru": "Конструкторы", "name_en": "Construction Sets"},
			{"slug": "plush", "name_ru": "Мягкие игрушки", "name_en": "Plush Toys"},
			{"slug": "board_games", "name_ru": "Настольные игры", "name_en": "Board Games"},
			{"slug": "educational", "name_ru": "Развивающие игрушки", "name_en": "Educational Toys"},
			{"slug": "outdoor", "name_ru": "Игрушки для улицы", "name_en": "Outdoor Toys"},
			{"slug": "creative", "name_ru": "Творчество", "name_en": "Creative Arts & Crafts"},
			{"slug": "electronic", "name_ru": "Электронные игрушки", "name_en": "Electronic Toys"},
			{"slug": "wooden", "name_ru": "Деревянные игрушки", "name_en": "Wooden Toys"},
			{"slug": "baby", "name_ru": "Игрушки для малышей", "name_en": "Baby Toys"},
			{"slug": "puzzles", "name_ru": "Головоломки и пазлы", "name_en": "Puzzles"},
			{"slug": "lego", "name_ru": "LEGO", "name_en": "LEGO"},
			{"slug": "action_figures", "name_ru": "Экшн-фигурки", "name_en": "Action Figures"},
			{"slug": "other", "name_ru": "Другое", "name_en": "Other"},
		}
		return c.JSON(fiber.Map{
			"categories": categories,
		})
	})

	// Регистрируем маршруты
	authService.SetupRoutes(app)
	cloudinaryService.SetupRoutes(app)
	listingService.SetupRoutes(app)
	tradeService.SetupRoutes(app)
	chatService.SetupRoutes(app)

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
