package chat

import (
	"context"
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

// ChatService представляет сервис для работы с чатами
type ChatService struct {
	cfg        *config.Config
	jwtService *utils.JWTService
}

// NewChatService создает новый экземпляр ChatService
func NewChatService(cfg *config.Config) *ChatService {
	return &ChatService{
		cfg:        cfg,
		jwtService: utils.NewJWTService(cfg.JWTSecret),
	}
}

// GetChats возвращает список чатов пользователя
func (s *ChatService) GetChats(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем userID в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	// Получаем контекст для работы с БД
	ctx, cancel := db.GetContext()
	defer cancel()

	// Запрос списка чатов
	query := `
        SELECT c.id, c.trade_id, c.sender_id, c.receiver_id, c.created_at, c.updated_at,
               c.last_message_text, c.last_message_time, c.is_active,
               COUNT(m.id) FILTER (WHERE m.sender_id != $1 AND m.is_read = false) AS unread_count
        FROM chats c
        LEFT JOIN messages m ON c.id = m.chat_id
        WHERE c.sender_id = $1 OR c.receiver_id = $1
        GROUP BY c.id
        ORDER BY c.last_message_time DESC NULLS LAST, c.created_at DESC
    `

	rows, err := db.Pool.Query(ctx, query, userUUID)
	if err != nil {
		log.Printf("Ошибка запроса чатов: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения чатов"})
	}
	defer rows.Close()

	// Обрабатываем результаты
	var chats []models.Chat
	for rows.Next() {
		var chat models.Chat
		var tradeID *uuid.UUID
		var lastMessageTime *time.Time
		var unreadCount int

		if err := rows.Scan(
			&chat.ID,
			&tradeID,
			&chat.SenderID,
			&chat.ReceiverID,
			&chat.CreatedAt,
			&chat.UpdatedAt,
			&chat.LastMessageText,
			&lastMessageTime,
			&chat.IsActive,
			&unreadCount,
		); err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}

		chat.TradeID = tradeID
		chat.LastMessageTime = lastMessageTime
		chat.UnreadCount = unreadCount

		// Получаем данные о другом участнике чата (не текущем пользователе)
		var otherUserID uuid.UUID
		if chat.SenderID == userUUID {
			otherUserID = chat.ReceiverID
			chat.Receiver = getUserInfo(ctx, otherUserID)
		} else {
			otherUserID = chat.SenderID
			chat.Sender = getUserInfo(ctx, otherUserID)
		}

		// Если есть связанный обмен, получаем информацию о нем
		if chat.TradeID != nil {
			chat.Trade = getTradeInfo(ctx, *chat.TradeID)
		}

		chats = append(chats, chat)
	}

	return c.JSON(fiber.Map{
		"chats": chats,
		"count": len(chats),
	})
}

// GetChatMessages возвращает сообщения конкретного чата
func (s *ChatService) GetChatMessages(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	chatID := c.Params("id")

	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем ID в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	chatUUID, err := uuid.Parse(chatID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID чата"})
	}

	// Проверяем, имеет ли пользователь доступ к этому чату
	ctx, cancel := db.GetContext()
	defer cancel()

	var count int
	err = db.Pool.QueryRow(ctx, `
        SELECT COUNT(*) FROM chats 
        WHERE id = $1 AND (sender_id = $2 OR receiver_id = $2)
    `, chatUUID, userUUID).Scan(&count)

	if err != nil {
		log.Printf("Ошибка проверки доступа к чату: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки доступа к чату"})
	}

	if count == 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "У вас нет доступа к этому чату"})
	}

	// Получаем сообщения
	limit := 50 // Ограничение количества сообщений

	// Обрабатываем пагинацию
	before := c.Query("before")
	var query string
	var queryArgs []interface{}

	if before != "" {
		beforeUUID, err := uuid.Parse(before)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID сообщения"})
		}

		query = `
            SELECT m.id, m.chat_id, m.sender_id, m.text, m.is_read, m.created_at, m.updated_at
            FROM messages m
            WHERE m.chat_id = $1 AND m.id < $2
            ORDER BY m.created_at DESC
            LIMIT $3
        `
		queryArgs = []interface{}{chatUUID, beforeUUID, limit}
	} else {
		query = `
            SELECT m.id, m.chat_id, m.sender_id, m.text, m.is_read, m.created_at, m.updated_at
            FROM messages m
            WHERE m.chat_id = $1
            ORDER BY m.created_at DESC
            LIMIT $2
        `
		queryArgs = []interface{}{chatUUID, limit}
	}

	rows, err := db.Pool.Query(ctx, query, queryArgs...)
	if err != nil {
		log.Printf("Ошибка запроса сообщений: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка получения сообщений"})
	}
	defer rows.Close()

	// Обрабатываем результаты
	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Text,
			&msg.IsRead,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		); err != nil {
			log.Printf("Ошибка сканирования сообщения: %v", err)
			continue
		}

		// Добавляем информацию об отправителе
		msg.Sender = getUserInfo(ctx, msg.SenderID)
		messages = append(messages, msg)
	}

	// Отмечаем сообщения как прочитанные
	_, err = db.Pool.Exec(ctx, `
        UPDATE messages
        SET is_read = true
        WHERE chat_id = $1 AND sender_id != $2 AND is_read = false
    `, chatUUID, userUUID)

	if err != nil {
		log.Printf("Ошибка обновления статуса прочтения: %v", err)
		// Не возвращаем ошибку, т.к. основная функциональность выполнена
	}

	return c.JSON(fiber.Map{
		"messages": messages,
		"has_more": len(messages) == limit,
	})
}

// SendMessage отправляет новое сообщение
func (s *ChatService) SendMessage(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	chatID := c.Params("id")

	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Преобразуем ID в UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID пользователя"})
	}

	chatUUID, err := uuid.Parse(chatID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID чата"})
	}

	// Получаем данные запроса
	var requestData struct {
		Text string `json:"text"`
	}

	if err := c.Bind().Body(&requestData); err != nil {
		log.Printf("Ошибка чтения тела запроса: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	if requestData.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Текст сообщения не может быть пустым"})
	}

	// Проверяем, имеет ли пользователь доступ к этому чату
	ctx, cancel := db.GetContext()
	defer cancel()

	var chat models.Chat
	err = db.Pool.QueryRow(ctx, `
        SELECT id, sender_id, receiver_id, is_active FROM chats 
        WHERE id = $1 AND (sender_id = $2 OR receiver_id = $2)
    `, chatUUID, userUUID).Scan(&chat.ID, &chat.SenderID, &chat.ReceiverID, &chat.IsActive)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "У вас нет доступа к этому чату"})
		}
		log.Printf("Ошибка проверки доступа к чату: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки доступа к чату"})
	}

	if !chat.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Чат неактивен"})
	}

	// Начинаем транзакцию
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}
	defer tx.Rollback(ctx)

	// Создаем новое сообщение
	messageID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(ctx, `
        INSERT INTO messages (id, chat_id, sender_id, text, is_read, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, messageID, chatUUID, userUUID, requestData.Text, false, now, now)

	if err != nil {
		log.Printf("Ошибка создания сообщения: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка сохранения сообщения"})
	}

	// Обновляем информацию о чате
	_, err = tx.Exec(ctx, `
        UPDATE chats
        SET last_message_text = $1, last_message_time = $2, updated_at = $3
        WHERE id = $4
    `, requestData.Text, now, now, chatUUID)

	if err != nil {
		log.Printf("Ошибка обновления информации о чате: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка обновления информации о чате"})
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Printf("Ошибка фиксации транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}

	// Создаем объект сообщения для ответа
	message := models.Message{
		ID:        messageID,
		ChatID:    chatUUID,
		SenderID:  userUUID,
		Text:      requestData.Text,
		IsRead:    false,
		CreatedAt: now,
		UpdatedAt: now,
		Sender:    getUserInfo(ctx, userUUID),
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": message,
		"success": true,
	})
}

// CreateChat создает новый чат между пользователями
func (s *ChatService) CreateChat(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Пользователь не авторизован"})
	}

	// Получаем данные запроса
	var requestData struct {
		ReceiverID string `json:"receiver_id"`
		TradeID    string `json:"trade_id,omitempty"`
		Message    string `json:"message,omitempty"`
	}

	if err := c.Bind().Body(&requestData); err != nil {
		log.Printf("Ошибка чтения тела запроса: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат данных"})
	}

	if requestData.ReceiverID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID получателя не указан"})
	}

	// Преобразуем ID в UUID
	senderUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID отправителя"})
	}

	receiverUUID, err := uuid.Parse(requestData.ReceiverID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID получателя"})
	}

	// Проверяем, что пользователь не создает чат с самим собой
	if senderUUID == receiverUUID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Нельзя создать чат с самим собой"})
	}

	// Проверяем, существует ли получатель
	ctx, cancel := db.GetContext()
	defer cancel()

	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE id = $1", receiverUUID).Scan(&count)
	if err != nil {
		log.Printf("Ошибка проверки существования получателя: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки получателя"})
	}

	if count == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Получатель не найден"})
	}

	// Проверяем, существует ли уже чат между этими пользователями
	var existingChatID *uuid.UUID
	err = db.Pool.QueryRow(ctx, `
        SELECT id FROM chats 
        WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
    `, senderUUID, receiverUUID).Scan(&existingChatID)

	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Ошибка проверки существующего чата: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки существования чата"})
	}

	// Если чат существует, возвращаем его ID
	if existingChatID != nil {
		// Если указано сообщение, отправляем его
		if requestData.Message != "" {
			// Отправляем сообщение в существующий чат
			now := time.Now()
			messageID := uuid.New()

			// Начинаем транзакцию
			tx, err := db.Pool.Begin(ctx)
			if err != nil {
				log.Printf("Ошибка начала транзакции: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
			}
			defer tx.Rollback(ctx)

			_, err = tx.Exec(ctx, `
                INSERT INTO messages (id, chat_id, sender_id, text, is_read, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7)
            `, messageID, existingChatID, senderUUID, requestData.Message, false, now, now)

			if err != nil {
				log.Printf("Ошибка создания сообщения: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка сохранения сообщения"})
			}

			// Обновляем информацию о чате
			_, err = tx.Exec(ctx, `
                UPDATE chats
                SET last_message_text = $1, last_message_time = $2, updated_at = $3
                WHERE id = $4
            `, requestData.Message, now, now, existingChatID)

			if err != nil {
				log.Printf("Ошибка обновления информации о чате: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка обновления информации о чате"})
			}

			// Фиксируем транзакцию
			if err = tx.Commit(ctx); err != nil {
				log.Printf("Ошибка фиксации транзакции: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
			}
		}

		return c.JSON(fiber.Map{
			"chat_id": existingChatID,
			"is_new":  false,
			"success": true,
		})
	}

	// Преобразуем TradeID в UUID, если он указан
	var tradeUUID *uuid.UUID
	if requestData.TradeID != "" {
		parsed, err := uuid.Parse(requestData.TradeID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат ID обмена"})
		}
		tradeUUID = &parsed

		// Проверяем существование обмена
		var tradeExists bool
		err = db.Pool.QueryRow(ctx, `
            SELECT EXISTS(SELECT 1 FROM trades WHERE id = $1)
        `, tradeUUID).Scan(&tradeExists)

		if err != nil {
			log.Printf("Ошибка проверки существования обмена: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка проверки обмена"})
		}

		if !tradeExists {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Указанный обмен не найден"})
		}
	}

	// Начинаем транзакцию
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}
	defer tx.Rollback(ctx)

	// Создаем новый чат
	chatID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(ctx, `
        INSERT INTO chats (id, trade_id, sender_id, receiver_id, created_at, updated_at, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, chatID, tradeUUID, senderUUID, receiverUUID, now, now, true)

	if err != nil {
		log.Printf("Ошибка создания чата: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка создания чата"})
	}

	// Если указано начальное сообщение, создаем его
	if requestData.Message != "" {
		messageID := uuid.New()

		_, err = tx.Exec(ctx, `
            INSERT INTO messages (id, chat_id, sender_id, text, is_read, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
        `, messageID, chatID, senderUUID, requestData.Message, false, now, now)

		if err != nil {
			log.Printf("Ошибка создания сообщения: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка сохранения сообщения"})
		}

		// Обновляем информацию о чате
		_, err = tx.Exec(ctx, `
            UPDATE chats
            SET last_message_text = $1, last_message_time = $2
            WHERE id = $3
        `, requestData.Message, now, chatID)

		if err != nil {
			log.Printf("Ошибка обновления информации о чате: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка обновления информации о чате"})
		}
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Printf("Ошибка фиксации транзакции: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Ошибка базы данных"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"chat_id": chatID,
		"is_new":  true,
		"success": true,
	})
}

// getUserInfo получает базовую информацию о пользователе
func getUserInfo(ctx context.Context, userID uuid.UUID) *models.User {
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
		log.Printf("Ошибка получения данных пользователя %s: %v", userID, err)
		return nil
	}

	return &user
}

// getTradeInfo получает базовую информацию об обмене
func getTradeInfo(ctx context.Context, tradeID uuid.UUID) *models.Trade {
	var trade models.Trade
	err := db.Pool.QueryRow(ctx, `
        SELECT id, sender_id, receiver_id, status, created_at
        FROM trades
        WHERE id = $1
    `, tradeID).Scan(
		&trade.ID,
		&trade.SenderID,
		&trade.ReceiverID,
		&trade.Status,
		&trade.CreatedAt,
	)

	if err != nil {
		log.Printf("Ошибка получения данных обмена %s: %v", tradeID, err)
		return nil
	}

	return &trade
}
