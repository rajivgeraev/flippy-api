package listing

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	validConditions := map[string]bool{
		"new": true, "excellent": true, "good": true,
		"used": true, "needs_repair": true, "damaged": true,
	}

	if !validConditions[requestData.Condition] {
		requestData.Condition = "new" // По умолчанию - новое
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
		INSERT INTO listings (id, user_id, title, description, categories, condition, allow_trade, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, listingID, userUUID, requestData.Title, requestData.Description,
		requestData.Categories, requestData.Condition, requestData.AllowTrade, requestData.Status)

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

// GetMyListings возвращает список объявлений текущего пользователя
func (s *ListingService) GetMyListings(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем userID в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Параметры фильтрации и пагинации
	status := c.Query("status", "all") // all, active, draft
	limit := 20                        // По умолчанию показываем 20 объявлений
	offsetStr := c.Query("offset", "0")
	offset, _ := strconv.Atoi(offsetStr)

	// Получаем объявления из базы данных
	ctx, cancel := db.GetContext()
	defer cancel()

	var listings []models.Listing
	var rows pgx.Rows
	var queryErr error

	if status == "all" {
		rows, queryErr = db.Pool.Query(ctx, `
			SELECT id, user_id, title, description, categories, condition, allow_trade, status, created_at, updated_at
			FROM listings
			WHERE user_id = $1
			ORDER BY updated_at DESC
			LIMIT $2 OFFSET $3
		`, userUUID, limit, offset)
	} else {
		rows, queryErr = db.Pool.Query(ctx, `
			SELECT id, user_id, title, description, categories, condition, allow_trade, status, created_at, updated_at
			FROM listings
			WHERE user_id = $1 AND status = $2
			ORDER BY updated_at DESC
			LIMIT $3 OFFSET $4
		`, userUUID, status, limit, offset)
	}

	if queryErr != nil {
		log.Printf("Ошибка запроса объявлений: %v", queryErr)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения объявлений"})
	}
	defer rows.Close()

	// Обрабатываем результаты
	for rows.Next() {
		var listing models.Listing
		if err := rows.Scan(
			&listing.ID,
			&listing.UserID,
			&listing.Title,
			&listing.Description,
			&listing.Categories,
			&listing.Condition,
			&listing.AllowTrade,
			&listing.Status,
			&listing.CreatedAt,
			&listing.UpdatedAt,
		); err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}

		// Получаем изображения для объявления
		imgRows, err := db.Pool.Query(ctx, `
			SELECT id, listing_id, url, preview_url, public_id, file_name, is_main, position, metadata, created_at
			FROM listing_images
			WHERE listing_id = $1
			ORDER BY position ASC
		`, listing.ID)

		if err != nil {
			log.Printf("Ошибка запроса изображений: %v", err)
			continue
		}

		var images []models.ListingImage
		for imgRows.Next() {
			var img models.ListingImage
			var metadataBytes []byte

			if err := imgRows.Scan(
				&img.ID,
				&img.ListingID,
				&img.URL,
				&img.PreviewURL,
				&img.PublicID,
				&img.FileName,
				&img.IsMain,
				&img.Position,
				&metadataBytes,
				&img.CreatedAt,
			); err != nil {
				log.Printf("Ошибка сканирования изображения: %v", err)
				continue
			}

			// Преобразуем метаданные из JSON, если они есть
			if metadataBytes != nil {
				if err := json.Unmarshal(metadataBytes, &img.Metadata); err != nil {
					log.Printf("Ошибка разбора метаданных: %v", err)
				}
			}

			images = append(images, img)
		}
		imgRows.Close()

		listing.Images = images
		listings = append(listings, listing)
	}

	// Получаем общее количество объявлений для пагинации
	var total int
	var countErr error

	if status == "all" {
		countErr = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM listings WHERE user_id = $1
		`, userUUID).Scan(&total)
	} else {
		countErr = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM listings WHERE user_id = $1 AND status = $2
		`, userUUID, status).Scan(&total)
	}

	if countErr != nil {
		log.Printf("Ошибка подсчета объявлений: %v", countErr)
		// Игнорируем ошибку, просто не вернем общее количество
	}

	return c.JSON(fiber.Map{
		"listings": listings,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetListing возвращает детальную информацию об объявлении
func (s *ListingService) GetListing(c fiber.Ctx) error {
	listingID := c.Params("id")
	if listingID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID объявления не указан"})
	}

	// Проверяем, что ID является валидным UUID
	listingUUID, err := uuid.Parse(listingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления"})
	}

	// Получаем текущего пользователя
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Получаем объявление из базы данных
	ctx, cancel := db.GetContext()
	defer cancel()

	var listing models.Listing
	var ownerID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
		SELECT id, user_id, title, description, categories, condition, allow_trade, status, created_at, updated_at
		FROM listings
		WHERE id = $1
	`, listingUUID).Scan(
		&listing.ID,
		&ownerID,
		&listing.Title,
		&listing.Description,
		&listing.Categories,
		&listing.Condition,
		&listing.AllowTrade,
		&listing.Status,
		&listing.CreatedAt,
		&listing.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Объявление не найдено"})
		}
		log.Printf("Ошибка получения объявления: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения объявления"})
	}

	// Проверка доступа: если объявление в статусе черновика, то его может видеть только автор
	listing.UserID = ownerID
	if listing.Status == "draft" && listing.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "У вас нет доступа к этому объявлению"})
	}

	// Получаем изображения для объявления
	rows, err := db.Pool.Query(ctx, `
		SELECT id, listing_id, url, preview_url, public_id, file_name, is_main, position, metadata, created_at
		FROM listing_images
		WHERE listing_id = $1
		ORDER BY position ASC
	`, listingUUID)

	if err != nil {
		log.Printf("Ошибка запроса изображений: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения изображений"})
	}
	defer rows.Close()

	var images []models.ListingImage
	for rows.Next() {
		var img models.ListingImage
		var metadataBytes []byte

		if err := rows.Scan(
			&img.ID,
			&img.ListingID,
			&img.URL,
			&img.PreviewURL,
			&img.PublicID,
			&img.FileName,
			&img.IsMain,
			&img.Position,
			&metadataBytes,
			&img.CreatedAt,
		); err != nil {
			log.Printf("Ошибка сканирования изображения: %v", err)
			continue
		}

		// Преобразуем метаданные из JSON, если они есть
		if metadataBytes != nil {
			if err := json.Unmarshal(metadataBytes, &img.Metadata); err != nil {
				log.Printf("Ошибка разбора метаданных: %v", err)
			}
		}

		images = append(images, img)
	}

	listing.Images = images

	// Получаем информацию о пользователе
	var user db.User
	err = db.Pool.QueryRow(ctx, `
		SELECT id, username, first_name, last_name, avatar_url
		FROM users
		WHERE id = $1
	`, ownerID).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
	)

	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Ошибка получения данных пользователя: %v", err)
	}

	// Формируем ответ
	return c.JSON(fiber.Map{
		"listing": listing,
		"user": fiber.Map{
			"id":         user.ID,
			"username":   user.Username,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"avatar_url": user.AvatarURL,
		},
		"is_owner": ownerID == userID,
	})
}

// UpdateListing обновляет существующее объявление
func (s *ListingService) UpdateListing(c fiber.Ctx) error {
	listingID := c.Params("id")
	userIDStr := c.Locals("userID").(string)

	if listingID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID объявления не указан"})
	}

	// Проверяем, что ID является валидным UUID
	listingUUID, err := uuid.Parse(listingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления"})
	}

	userID, err := uuid.Parse(userIDStr)
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

	// Проверка статуса
	if requestData.Status != "active" && requestData.Status != "draft" {
		requestData.Status = "draft" // По умолчанию - черновик
	}

	validConditions := map[string]bool{
		"new": true, "excellent": true, "good": true,
		"used": true, "needs_repair": true, "damaged": true,
	}

	if !validConditions[requestData.Condition] {
		requestData.Condition = "new" // По умолчанию - новое
	}

	// Проверяем, что объявление существует и принадлежит пользователю
	ctx, cancel := db.GetContext()
	defer cancel()

	var ownerID uuid.UUID
	err = db.Pool.QueryRow(ctx, "SELECT user_id FROM listings WHERE id = $1", listingUUID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Объявление не найдено"})
		}
		log.Printf("Ошибка запроса объявления: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения объявления"})
	}

	// Проверка, что пользователь является владельцем объявления
	if ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "У вас нет доступа к редактированию этого объявления"})
	}

	// Начинаем транзакцию
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}
	defer tx.Rollback(ctx)

	// Обновляем основную информацию объявления
	_, err = tx.Exec(ctx, `
		UPDATE listings 
		SET title = $1, description = $2, categories = $3, condition = $4, allow_trade = $5, status = $6, updated_at = NOW()
		WHERE id = $7
	`, requestData.Title, requestData.Description, requestData.Categories, requestData.Condition, requestData.AllowTrade, requestData.Status, listingUUID)

	if err != nil {
		log.Printf("Ошибка обновления объявления: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка обновления объявления"})
	}

	// Если есть изображения, обновляем их
	if len(requestData.Images) > 0 {
		// Сначала удаляем все существующие изображения
		_, err = tx.Exec(ctx, "DELETE FROM listing_images WHERE listing_id = $1", listingUUID)
		if err != nil {
			log.Printf("Ошибка удаления старых изображений: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка обновления изображений"})
		}

		// Добавляем новые изображения
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
			`, listingUUID, img.URL, previewURL, img.PublicID, img.FileName, isMain, i, metadata)

			if err != nil {
				log.Printf("Ошибка вставки изображения: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка сохранения изображений"})
			}
		}
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Printf("Ошибка фиксации транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"listing_id": listingID,
		"message":    "Объявление успешно обновлено",
	})
}

// DeleteListing удаляет объявление
func (s *ListingService) DeleteListing(c fiber.Ctx) error {
	listingID := c.Params("id")
	userIDStr := c.Locals("userID").(string)

	if listingID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID объявления не указан"})
	}

	// Проверяем, что ID является валидным UUID
	listingUUID, err := uuid.Parse(listingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Проверяем, что объявление существует и принадлежит пользователю
	ctx, cancel := db.GetContext()
	defer cancel()

	var ownerID uuid.UUID
	err = db.Pool.QueryRow(ctx, "SELECT user_id FROM listings WHERE id = $1", listingUUID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Объявление не найдено"})
		}
		log.Printf("Ошибка запроса объявления: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения объявления"})
	}

	// Проверка, что пользователь является владельцем объявления
	if ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "У вас нет доступа к удалению этого объявления"})
	}

	// Начинаем транзакцию
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}
	defer tx.Rollback(ctx)

	// Сначала удаляем связанные изображения
	_, err = tx.Exec(ctx, "DELETE FROM listing_images WHERE listing_id = $1", listingUUID)
	if err != nil {
		log.Printf("Ошибка удаления изображений: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка удаления объявления"})
	}

	// Удаляем само объявление
	_, err = tx.Exec(ctx, "DELETE FROM listings WHERE id = $1", listingUUID)
	if err != nil {
		log.Printf("Ошибка удаления объявления: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка удаления объявления"})
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Printf("Ошибка фиксации транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Объявление успешно удалено",
	})
}

// GetPublicListings возвращает список публичных активных объявлений с пагинацией
func (s *ListingService) GetPublicListings(c fiber.Ctx) error {
	// Параметры пагинации
	limit := 20 // По умолчанию показываем 20 объявлений
	offsetStr := c.Query("offset", "0")
	offset, _ := strconv.Atoi(offsetStr)

	// Получаем объявления из базы данных
	ctx, cancel := db.GetContext()
	defer cancel()

	var listings []models.Listing

	rows, queryErr := db.Pool.Query(ctx, `
        SELECT id, user_id, title, description, categories, condition, allow_trade, status, created_at, updated_at
        FROM listings
        WHERE status = 'active'  -- Берем только активные объявления
        ORDER BY created_at DESC  -- Сначала новые
        LIMIT $1 OFFSET $2
    `, limit, offset)

	if queryErr != nil {
		log.Printf("Ошибка запроса объявлений: %v", queryErr)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения объявлений"})
	}
	defer rows.Close()

	// Обрабатываем результаты
	for rows.Next() {
		var listing models.Listing
		if err := rows.Scan(
			&listing.ID,
			&listing.UserID,
			&listing.Title,
			&listing.Description,
			&listing.Categories,
			&listing.Condition,
			&listing.AllowTrade,
			&listing.Status,
			&listing.CreatedAt,
			&listing.UpdatedAt,
		); err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}

		// Получаем изображения для объявления
		imgRows, err := db.Pool.Query(ctx, `
            SELECT id, listing_id, url, preview_url, public_id, file_name, is_main, position, metadata, created_at
            FROM listing_images
            WHERE listing_id = $1
            ORDER BY position ASC
        `, listing.ID)

		if err != nil {
			log.Printf("Ошибка запроса изображений: %v", err)
			continue
		}

		var images []models.ListingImage
		for imgRows.Next() {
			var img models.ListingImage
			var metadataBytes []byte

			if err := imgRows.Scan(
				&img.ID,
				&img.ListingID,
				&img.URL,
				&img.PreviewURL,
				&img.PublicID,
				&img.FileName,
				&img.IsMain,
				&img.Position,
				&metadataBytes,
				&img.CreatedAt,
			); err != nil {
				log.Printf("Ошибка сканирования изображения: %v", err)
				continue
			}

			// Преобразуем метаданные из JSON, если они есть
			if metadataBytes != nil {
				if err := json.Unmarshal(metadataBytes, &img.Metadata); err != nil {
					log.Printf("Ошибка разбора метаданных: %v", err)
				}
			}

			images = append(images, img)
		}
		imgRows.Close()

		listing.Images = images

		// Для каждого объявления получаем информацию о пользователе
		var user struct {
			ID        uuid.UUID `json:"id"`
			Username  string    `json:"username"`
			FirstName string    `json:"first_name"`
			LastName  string    `json:"last_name"`
			AvatarURL string    `json:"avatar_url"`
		}

		err = db.Pool.QueryRow(ctx, `
            SELECT id, username, first_name, last_name, avatar_url
            FROM users
            WHERE id = $1
        `, listing.UserID).Scan(
			&user.ID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.AvatarURL,
		)

		// Даже если не удалось получить данные пользователя, мы все равно добавляем объявление
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Ошибка получения данных пользователя: %v", err)
		}

		listings = append(listings, listing)
	}

	// Получаем общее количество объявлений для пагинации
	var total int
	countErr := db.Pool.QueryRow(ctx, `
        SELECT COUNT(*) FROM listings WHERE status = 'active'
    `).Scan(&total)

	if countErr != nil {
		log.Printf("Ошибка подсчета объявлений: %v", countErr)
		// Игнорируем ошибку, просто не вернем общее количество
	}

	return c.JSON(fiber.Map{
		"listings": listings,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}
