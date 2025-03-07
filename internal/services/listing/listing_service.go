package listing

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rajivgeraev/flippy-api/internal/config"
	"github.com/rajivgeraev/flippy-api/internal/db"
	"github.com/rajivgeraev/flippy-api/internal/models"
	"github.com/rajivgeraev/flippy-api/internal/utils"
)

// RequestImage представляет структуру изображения в запросе создания объявления
type RequestImage struct {
	URL                string          `json:"url"`
	PublicID           string          `json:"public_id"`
	FileName           string          `json:"file_name"`
	IsMain             bool            `json:"is_main"`
	CloudinaryResponse json.RawMessage `json:"cloudinary_response,omitempty"`
}

// ListingService представляет сервис для работы с объявлениями
type ListingService struct {
	cfg        *config.Config
	jwtService *utils.JWTService
}

// NewListingService создает новый экземпляр ListingService
func NewListingService(cfg *config.Config) *ListingService {
	return &ListingService{
		cfg:        cfg,
		jwtService: utils.NewJWTService(cfg.JWTSecret),
	}
}

// CreateListing обрабатывает создание нового объявления
func (s *ListingService) CreateListing(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем userID в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Извлекаем данные из запроса
	var requestData struct {
		Title         string         `json:"title"`
		Description   string         `json:"description"`
		Categories    []string       `json:"categories"`
		Condition     string         `json:"condition"`
		AllowTrade    bool           `json:"allow_trade"`
		Status        string         `json:"status"`
		UploadGroupID string         `json:"upload_group_id"`
		Images        []RequestImage `json:"images"`
	}

	if err := c.Bind().Body(&requestData); err != nil {
		log.Printf("Ошибка декодирования тела запроса: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	// Валидация обязательных полей
	if requestData.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Название обязательно"})
	}

	// Проверка, что хотя бы одна категория добавлена для активных объявлений
	if requestData.Status == "active" && len(requestData.Categories) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Выберите хотя бы одну категорию"})
	}

	// Проверка, что хотя бы одно изображение добавлено для активных объявлений
	if requestData.Status == "active" && len(requestData.Images) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Добавьте хотя бы одно изображение"})
	}

	// Проверка валидности status
	if requestData.Status != "active" && requestData.Status != "draft" {
		requestData.Status = "draft" // По умолчанию - черновик
	}

	// Создаем ID для нового объявления
	listingID := uuid.New()

	// Начинаем транзакцию
	ctx, cancel := db.GetContext()
	defer cancel()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}
	defer tx.Rollback(ctx)

	// Вставляем объявление
	_, err = tx.Exec(ctx, `
		INSERT INTO listings (id, user_id, title, description, categories, allow_trade, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, listingID, userUUID, requestData.Title, requestData.Description, requestData.Categories, requestData.AllowTrade, requestData.Status)

	if err != nil {
		log.Printf("Ошибка вставки объявления: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка сохранения объявления"})
	}

	// Вставляем изображения, если они есть
	for i, img := range requestData.Images {
		isMain := i == 0 // Первое изображение - основное

		var cloudinaryResp models.CloudinaryResponse
		var metadata []byte
		var previewURL string

		// Обрабатываем данные из Cloudinary
		if img.CloudinaryResponse != nil && len(img.CloudinaryResponse) > 0 {
			// Парсим JSON-ответ Cloudinary
			if err := json.Unmarshal(img.CloudinaryResponse, &cloudinaryResp); err != nil {
				log.Printf("Ошибка парсинга ответа Cloudinary: %v", err)
			} else {
				// Извлекаем URL превью
				previewURL = models.ExtractPreviewURL(cloudinaryResp)

				// Формируем метаданные для сохранения
				metadataObj := models.ExtractMetadata(cloudinaryResp)
				metadata, _ = json.Marshal(metadataObj)
			}
		}

		// Вставляем информацию об изображении
		_, err = tx.Exec(ctx, `
			INSERT INTO listing_images (listing_id, url, preview_url, public_id, file_name, is_main, position, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, listingID, img.URL, previewURL, img.PublicID, img.FileName, isMain, i, metadata)

		if err != nil {
			log.Printf("Ошибка вставки изображения: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка сохранения изображений"})
		}
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Printf("Ошибка фиксации транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":    true,
		"listing_id": listingID,
		"message":    "Объявление успешно создано",
	})
}
