package trade

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rajivgeraev/flippy-api/internal/config"
	"github.com/rajivgeraev/flippy-api/internal/db"
	"github.com/rajivgeraev/flippy-api/internal/models"
	"github.com/rajivgeraev/flippy-api/internal/utils"
)

// TradeService представляет сервис для работы с обменами
type TradeService struct {
	cfg        *config.Config
	jwtService *utils.JWTService
}

// NewTradeService создает новый экземпляр TradeService
func NewTradeService(cfg *config.Config) *TradeService {
	return &TradeService{
		cfg:        cfg,
		jwtService: utils.NewJWTService(cfg.JWTSecret),
	}
}

// CreateTrade создает новое предложение обмена
func (s *TradeService) CreateTrade(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем userID в UUID
	senderID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Извлекаем данные из запроса
	var requestData struct {
		ReceiverListingID string `json:"receiver_listing_id"`
		SenderListingID   string `json:"sender_listing_id"`
		Message           string `json:"message"`
	}

	if err := c.Bind().Body(&requestData); err != nil {
		log.Printf("Ошибка декодирования тела запроса: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	// Проверка обязательных полей
	if requestData.ReceiverListingID == "" || requestData.SenderListingID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Необходимо указать ID объявлений для обмена"})
	}

	// Преобразуем ID в UUID
	receiverListingID, err := uuid.Parse(requestData.ReceiverListingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления получателя"})
	}

	senderListingID, err := uuid.Parse(requestData.SenderListingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID объявления отправителя"})
	}

	// Получаем контекст для работы с БД
	ctx, cancel := db.GetContext()
	defer cancel()

	// Проверяем, что объявление отправителя принадлежит ему
	var senderListingOwnerID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
        SELECT user_id FROM listings WHERE id = $1
    `, senderListingID).Scan(&senderListingOwnerID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Объявление отправителя не найдено"})
		}
		log.Printf("Ошибка запроса объявления отправителя: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки объявления"})
	}

	if senderListingOwnerID != senderID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Вы не можете предложить чужое объявление для обмена"})
	}

	// Получаем ID владельца объявления получателя
	var receiverID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
        SELECT user_id FROM listings WHERE id = $1
    `, receiverListingID).Scan(&receiverID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Объявление получателя не найдено"})
		}
		log.Printf("Ошибка запроса объявления получателя: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки объявления"})
	}

	// Проверяем, что пользователь не предлагает обмен самому себе
	if receiverID == senderID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Вы не можете предложить обмен самому себе"})
	}

	// Проверяем, не существует ли уже предложение обмена с такими же объявлениями
	var existingTradeCount int
	err = db.Pool.QueryRow(ctx, `
        SELECT COUNT(*) FROM trades 
        WHERE sender_listing_id = $1 AND receiver_listing_id = $2 AND status = 'pending'
    `, senderListingID, receiverListingID).Scan(&existingTradeCount)

	if err != nil {
		log.Printf("Ошибка проверки существующих предложений: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки существующих обменов"})
	}

	if existingTradeCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Такое предложение обмена уже существует"})
	}

	// Начинаем транзакцию
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}
	defer tx.Rollback(ctx)

	// Создаем ID для нового предложения обмена
	tradeID := uuid.New()

	// Вставляем предложение обмена
	_, err = tx.Exec(ctx, `
        INSERT INTO trades (id, sender_id, receiver_id, sender_listing_id, receiver_listing_id, status, message)
        VALUES ($1, $2, $3, $4, $5, 'pending', $6)
    `, tradeID, senderID, receiverID, senderListingID, receiverListingID, requestData.Message)

	if err != nil {
		log.Printf("Ошибка создания предложения обмена: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка сохранения предложения обмена"})
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Printf("Ошибка фиксации транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":  true,
		"trade_id": tradeID,
		"message":  "Предложение обмена успешно создано",
	})
}

// GetMyTrades возвращает список входящих и исходящих предложений обмена
func (s *TradeService) GetMyTrades(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем userID в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Получаем тип предложений (входящие/исходящие/все)
	tradeType := c.Query("type", "all") // all, incoming, outgoing
	status := c.Query("status", "all")  // all, pending, accepted, rejected

	// Получаем контекст для работы с БД
	ctx, cancel := db.GetContext()
	defer cancel()

	// Формируем базовый запрос в зависимости от типа и статуса
	var query string
	var args []interface{}

	if tradeType == "incoming" {
		if status == "all" {
			query = `
                SELECT t.id, t.sender_id, t.receiver_id, t.sender_listing_id, t.receiver_listing_id,
                       t.status, t.message, t.created_at, t.updated_at
                FROM trades t
                WHERE t.receiver_id = $1
                ORDER BY t.created_at DESC
            `
			args = []interface{}{userUUID}
		} else {
			query = `
                SELECT t.id, t.sender_id, t.receiver_id, t.sender_listing_id, t.receiver_listing_id,
                       t.status, t.message, t.created_at, t.updated_at
                FROM trades t
                WHERE t.receiver_id = $1 AND t.status = $2
                ORDER BY t.created_at DESC
            `
			args = []interface{}{userUUID, status}
		}
	} else if tradeType == "outgoing" {
		if status == "all" {
			query = `
                SELECT t.id, t.sender_id, t.receiver_id, t.sender_listing_id, t.receiver_listing_id,
                       t.status, t.message, t.created_at, t.updated_at
                FROM trades t
                WHERE t.sender_id = $1
                ORDER BY t.created_at DESC
            `
			args = []interface{}{userUUID}
		} else {
			query = `
                SELECT t.id, t.sender_id, t.receiver_id, t.sender_listing_id, t.receiver_listing_id,
                       t.status, t.message, t.created_at, t.updated_at
                FROM trades t
                WHERE t.sender_id = $1 AND t.status = $2
                ORDER BY t.created_at DESC
            `
			args = []interface{}{userUUID, status}
		}
	} else { // all
		if status == "all" {
			query = `
                SELECT t.id, t.sender_id, t.receiver_id, t.sender_listing_id, t.receiver_listing_id,
                       t.status, t.message, t.created_at, t.updated_at
                FROM trades t
                WHERE t.sender_id = $1 OR t.receiver_id = $1
                ORDER BY t.created_at DESC
            `
			args = []interface{}{userUUID}
		} else {
			query = `
                SELECT t.id, t.sender_id, t.receiver_id, t.sender_listing_id, t.receiver_listing_id,
                       t.status, t.message, t.created_at, t.updated_at
                FROM trades t
                WHERE (t.sender_id = $1 OR t.receiver_id = $1) AND t.status = $2
                ORDER BY t.created_at DESC
            `
			args = []interface{}{userUUID, status}
		}
	}

	// Выполняем запрос
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("Ошибка запроса предложений обмена: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения предложений обмена"})
	}
	defer rows.Close()

	// Обрабатываем результаты
	var trades []models.Trade
	for rows.Next() {
		var trade models.Trade
		if err := rows.Scan(
			&trade.ID,
			&trade.SenderID,
			&trade.ReceiverID,
			&trade.SenderListingID,
			&trade.ReceiverListingID,
			&trade.Status,
			&trade.Message,
			&trade.CreatedAt,
			&trade.UpdatedAt,
		); err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}

		// Загружаем дополнительную информацию об объявлениях и пользователях
		trade.SenderListing = s.getListingInfo(ctx, trade.SenderListingID)
		trade.ReceiverListing = s.getListingInfo(ctx, trade.ReceiverListingID)
		trade.Sender = s.getUserInfo(ctx, trade.SenderID)
		trade.Receiver = s.getUserInfo(ctx, trade.ReceiverID)

		// Получаем ID чата, связанного с этим обменом (если есть)
		var chatID *uuid.UUID
		err = db.Pool.QueryRow(ctx, `
            SELECT id FROM chats WHERE trade_id = $1 LIMIT 1
        `, trade.ID).Scan(&chatID)

		if err == nil && chatID != nil {
			trade.ChatID = *chatID // Добавляем ID чата к данным обмена
		}

		trades = append(trades, trade)
	}

	return c.JSON(fiber.Map{
		"trades": trades,
		"count":  len(trades),
	})
}

// UpdateTradeStatus обновляет статус предложения обмена (принятие/отклонение)
func (s *TradeService) UpdateTradeStatus(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Получаем ID предложения обмена из URL
	tradeID := c.Params("id")
	if tradeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID предложения обмена не указан"})
	}

	// Получаем новый статус из запроса
	var requestData struct {
		Status string `json:"status"` // accepted, rejected, canceled
	}

	if err := c.Bind().Body(&requestData); err != nil {
		log.Printf("Ошибка декодирования тела запроса: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	// Проверяем допустимость статуса
	if requestData.Status != "accepted" && requestData.Status != "rejected" && requestData.Status != "canceled" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Недопустимый статус предложения обмена"})
	}

	// Преобразуем ID в UUID
	tradeUUID, err := uuid.Parse(tradeID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID предложения обмена"})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Получаем контекст для работы с БД
	ctx, cancel := db.GetContext()
	defer cancel()

	// Проверяем, существует ли предложение обмена и принадлежит ли оно пользователю
	var trade models.Trade
	err = db.Pool.QueryRow(ctx, `
        SELECT id, sender_id, receiver_id, status
        FROM trades
        WHERE id = $1
    `, tradeUUID).Scan(&trade.ID, &trade.SenderID, &trade.ReceiverID, &trade.Status)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Предложение обмена не найдено"})
		}
		log.Printf("Ошибка запроса предложения обмена: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения предложения обмена"})
	}

	// Проверяем право на изменение статуса
	isReceiver := trade.ReceiverID == userUUID
	isSender := trade.SenderID == userUUID

	if requestData.Status == "accepted" || requestData.Status == "rejected" {
		// Только получатель может принять или отклонить предложение
		if !isReceiver {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Только получатель предложения может его принять или отклонить"})
		}
	} else if requestData.Status == "canceled" {
		// Только отправитель может отменить предложение
		if !isSender {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Только отправитель предложения может его отменить"})
		}
	}

	// Проверяем текущий статус
	if trade.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Нельзя изменить статус предложения, которое уже не находится в ожидании",
		})
	}

	// Обновляем статус предложения обмена
	_, err = db.Pool.Exec(ctx, `
        UPDATE trades
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `, requestData.Status, tradeUUID)

	if err != nil {
		log.Printf("Ошибка обновления статуса предложения: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка обновления статуса предложения"})
	}

	// Если обмен принят, создаем чат между участниками
	var chatID uuid.UUID
	if requestData.Status == "accepted" {
		// Создаем чат
		chatID = uuid.New()
		now := time.Now()
		initialMessage := "Обмен был принят. Вы можете обсудить детали здесь."

		// Создаем запись в таблице чатов
		_, err = db.Pool.Exec(ctx, `
            INSERT INTO chats (id, trade_id, sender_id, receiver_id, created_at, updated_at, last_message_text, last_message_time, is_active)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        `, chatID, tradeUUID, trade.SenderID, trade.ReceiverID, now, now, initialMessage, now, true)

		if err != nil {
			log.Printf("Ошибка создания чата для обмена: %v", err)
			// Не возвращаем ошибку, т.к. основная функциональность выполнена
		} else {
			// Добавляем системное сообщение
			messageID := uuid.New()
			_, err = db.Pool.Exec(ctx, `
                INSERT INTO messages (id, chat_id, sender_id, text, is_read, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7)
            `, messageID, chatID, trade.SenderID, initialMessage, false, now, now)

			if err != nil {
				log.Printf("Ошибка создания системного сообщения: %v", err)
				// Не возвращаем ошибку, т.к. основная функциональность выполнена
			}
		}
	}

	// Формируем сообщение в зависимости от нового статуса
	var message string
	switch requestData.Status {
	case "accepted":
		message = "Предложение обмена принято"
	case "rejected":
		message = "Предложение обмена отклонено"
	case "canceled":
		message = "Предложение обмена отменено"
	}

	response := fiber.Map{
		"success":  true,
		"message":  message,
		"trade_id": tradeID,
		"status":   requestData.Status,
	}

	// Если был создан чат, включаем его ID в ответ
	if requestData.Status == "accepted" {
		response["chat_id"] = chatID
	}

	return c.JSON(response)
}

// getListingInfo получает информацию об объявлении
func (s *TradeService) getListingInfo(ctx context.Context, listingID uuid.UUID) *models.Listing {
	var listing models.Listing
	var categoriesData []byte

	err := db.Pool.QueryRow(ctx, `
        SELECT id, user_id, title, description, categories, condition, allow_trade, status
        FROM listings
        WHERE id = $1
    `, listingID).Scan(
		&listing.ID,
		&listing.UserID,
		&listing.Title,
		&listing.Description,
		&categoriesData,
		&listing.Condition,
		&listing.AllowTrade,
		&listing.Status,
	)

	if err != nil {
		log.Printf("Ошибка получения объявления %s: %v", listingID, err)
		return nil
	}

	// Преобразуем JSONB категории в массив строк
	if err := json.Unmarshal(categoriesData, &listing.Categories); err != nil {
		log.Printf("Ошибка разбора категорий: %v", err)
		listing.Categories = []string{}
	}

	// Получаем изображения объявления
	rows, err := db.Pool.Query(ctx, `
        SELECT id, url, preview_url, is_main
        FROM listing_images
        WHERE listing_id = $1
        ORDER BY position ASC
    `, listingID)

	if err != nil {
		log.Printf("Ошибка получения изображений: %v", err)
	} else {
		defer rows.Close()

		var images []models.ListingImage
		for rows.Next() {
			var image models.ListingImage
			if err := rows.Scan(&image.ID, &image.URL, &image.PreviewURL, &image.IsMain); err != nil {
				log.Printf("Ошибка сканирования изображения: %v", err)
				continue
			}
			image.ListingID = listingID
			images = append(images, image)
		}
		listing.Images = images
	}

	return &listing
}

// getUserInfo получает информацию о пользователе
func (s *TradeService) getUserInfo(ctx context.Context, userID uuid.UUID) *models.User {
	var user models.User
	err := db.Pool.QueryRow(ctx, `
        SELECT id, username, first_name, last_name, avatar_url
        FROM users
        WHERE id = $1
    `, userID).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
	)

	if err != nil {
		log.Printf("Ошибка получения пользователя %s: %v", userID, err)
		return nil
	}

	return &user
}
