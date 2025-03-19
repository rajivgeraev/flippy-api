package favorite

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

// FavoriteService представляет сервис для работы с избранными объявлениями
type FavoriteService struct {
	cfg        *config.Config
	jwtService *utils.JWTService
}

// NewFavoriteService создает новый экземпляр FavoriteService
func NewFavoriteService(cfg *config.Config) *FavoriteService {
	return &FavoriteService{
		cfg:        cfg,
		jwtService: utils.NewJWTService(cfg.JWTSecret),
	}
}

// AddToFavorites добавляет объявление в избранное
func (s *FavoriteService) AddToFavorites(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Извлекаем ID объявления из запроса
	var requestData struct {
		ListingID string `json:"listing_id"`
	}

	if err := c.Bind().Body(&requestData); err != nil {
		log.Printf("Ошибка декодирования тела запроса: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	// Проверяем, что listing_id указан
	if requestData.ListingID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID объявления не указан"})
	}

	// Преобразуем строки в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	listingUUID, err := uuid.Parse(requestData.ListingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления"})
	}

	// Проверяем, существует ли объявление
	ctx, cancel := db.GetContext()
	defer cancel()

	var exists bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM listings WHERE id = $1 AND status = 'active')
	`, listingUUID).Scan(&exists)

	if err != nil {
		log.Printf("Ошибка проверки существования объявления: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки объявления"})
	}

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Объявление не найдено или не активно"})
	}

	// Проверяем, не добавлено ли уже это объявление в избранное
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND listing_id = $2)
	`, userUUID, listingUUID).Scan(&exists)

	if err != nil {
		log.Printf("Ошибка проверки избранного: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки избранного"})
	}

	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Объявление уже добавлено в избранное"})
	}

	// Добавляем объявление в избранное
	favoriteID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO favorites (id, user_id, listing_id)
		VALUES ($1, $2, $3)
	`, favoriteID, userUUID, listingUUID)

	if err != nil {
		log.Printf("Ошибка добавления в избранное: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка добавления в избранное"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"id":      favoriteID,
		"message": "Объявление успешно добавлено в избранное",
	})
}

// RemoveFromFavorites удаляет объявление из избранного
func (s *FavoriteService) RemoveFromFavorites(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	listingID := c.Params("id")

	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	if listingID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID объявления не указан"})
	}

	// Преобразуем строки в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	listingUUID, err := uuid.Parse(listingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления"})
	}

	// Проверяем, есть ли объявление в избранном
	ctx, cancel := db.GetContext()
	defer cancel()

	var exists bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND listing_id = $2)
	`, userUUID, listingUUID).Scan(&exists)

	if err != nil {
		log.Printf("Ошибка проверки избранного: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки избранного"})
	}

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Объявление не найдено в избранном"})
	}

	// Удаляем объявление из избранного
	_, err = db.Pool.Exec(ctx, `
		DELETE FROM favorites WHERE user_id = $1 AND listing_id = $2
	`, userUUID, listingUUID)

	if err != nil {
		log.Printf("Ошибка удаления из избранного: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка удаления из избранного"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Объявление успешно удалено из избранного",
	})
}

// GetFavorites возвращает список избранных объявлений пользователя
func (s *FavoriteService) GetFavorites(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем userID в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Параметры пагинации
	limit := 20 // По умолчанию показываем 20 объявлений
	offsetStr := c.Query("offset", "0")
	offset, _ := strconv.Atoi(offsetStr)

	// Получаем избранные объявления из базы данных
	ctx, cancel := db.GetContext()
	defer cancel()

	// Запрос на получение избранных объявлений с информацией об объявлениях
	query := `
		SELECT f.id, f.user_id, f.listing_id, f.created_at,
			   l.id, l.user_id, l.title, l.description, l.categories, l.condition, l.allow_trade, l.status, l.created_at, l.updated_at
		FROM favorites f
		JOIN listings l ON f.listing_id = l.id
		WHERE f.user_id = $1 AND l.status = 'active'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.Pool.Query(ctx, query, userUUID, limit, offset)
	if err != nil {
		log.Printf("Ошибка запроса избранных объявлений: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения избранных объявлений"})
	}
	defer rows.Close()

	var favorites []models.Favorite
	for rows.Next() {
		var favorite models.Favorite
		var listing models.Listing
		var categoriesData []byte

		if err := rows.Scan(
			&favorite.ID,
			&favorite.UserID,
			&favorite.ListingID,
			&favorite.CreatedAt,
			&listing.ID,
			&listing.UserID,
			&listing.Title,
			&listing.Description,
			&categoriesData,
			&listing.Condition,
			&listing.AllowTrade,
			&listing.Status,
			&listing.CreatedAt,
			&listing.UpdatedAt,
		); err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}

		// Преобразуем JSONB категории в массив строк
		if err := json.Unmarshal(categoriesData, &listing.Categories); err != nil {
			log.Printf("Ошибка разбора категорий: %v", err)
			listing.Categories = []string{}
		}

		// Получаем изображения для объявления
		imgRows, err := db.Pool.Query(ctx, `
			SELECT id, listing_id, url, preview_url, public_id, file_name, is_main, position, created_at
			FROM listing_images
			WHERE listing_id = $1
			ORDER BY position ASC
		`, listing.ID)

		if err != nil {
			log.Printf("Ошибка запроса изображений: %v", err)
		} else {
			defer imgRows.Close()

			var images []models.ListingImage
			for imgRows.Next() {
				var img models.ListingImage
				if err := imgRows.Scan(
					&img.ID,
					&img.ListingID,
					&img.URL,
					&img.PreviewURL,
					&img.PublicID,
					&img.FileName,
					&img.IsMain,
					&img.Position,
					&img.CreatedAt,
				); err != nil {
					log.Printf("Ошибка сканирования изображения: %v", err)
					continue
				}
				images = append(images, img)
			}
			listing.Images = images
		}

		favorite.Listing = &listing
		favorites = append(favorites, favorite)
	}

	// Получаем общее количество избранных объявлений для пагинации
	var total int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM favorites f
		JOIN listings l ON f.listing_id = l.id
		WHERE f.user_id = $1 AND l.status = 'active'
	`, userUUID).Scan(&total)

	if err != nil {
		log.Printf("Ошибка подсчета избранных объявлений: %v", err)
		// Игнорируем ошибку, просто не вернем общее количество
	}

	return c.JSON(fiber.Map{
		"favorites": favorites,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// CheckFavorite проверяет, добавлено ли объявление в избранное
func (s *FavoriteService) CheckFavorite(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	listingID := c.Params("id")

	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	if listingID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID объявления не указан"})
	}

	// Преобразуем строки в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	listingUUID, err := uuid.Parse(listingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления"})
	}

	// Проверяем, есть ли объявление в избранном
	ctx, cancel := db.GetContext()
	defer cancel()

	var exists bool
	var favoriteID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
		SELECT id, true FROM favorites WHERE user_id = $1 AND listing_id = $2
	`, userUUID, listingUUID).Scan(&favoriteID, &exists)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.JSON(fiber.Map{
				"is_favorite": false,
			})
		}
		log.Printf("Ошибка проверки избранного: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки избранного"})
	}

	return c.JSON(fiber.Map{
		"is_favorite": true,
		"favorite_id": favoriteID,
	})
}
