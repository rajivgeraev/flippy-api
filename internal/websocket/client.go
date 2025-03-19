package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Максимальное время ожидания для pong от клиента
	pongWait = 60 * time.Second

	// Отправлять ping-сообщения клиенту с этим интервалом
	pingPeriod = (pongWait * 9) / 10

	// Максимальный размер сообщения от клиента
	maxMessageSize = 512 * 1024 // 512KB

	// Размер буфера для отправляемых сообщений
	writeBufferSize = 256
)

// Client представляет собой отдельное WebSocket соединение
type Client struct {
	ID        uuid.UUID
	UserID    string
	conn      *websocket.Conn
	send      chan []byte // Буферизованный канал исходящих сообщений
	manager   *Manager
	closeChan chan struct{}
}

// NewClient создает новый экземпляр Client
func NewClient(userID string, conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		ID:        uuid.New(),
		UserID:    userID,
		conn:      conn,
		send:      make(chan []byte, writeBufferSize),
		manager:   manager,
		closeChan: make(chan struct{}),
	}
}

// Start запускает клиентские горутины для чтения и записи
func (c *Client) Start() {
	// Добавляем клиент к менеджеру
	c.manager.AddClient(c)

	// Запускаем горутины для чтения и записи
	go c.readPump()
	go c.writePump()
}

// readPump обрабатывает входящие сообщения от клиента
func (c *Client) readPump() {
	defer func() {
		c.manager.RemoveClient(c.ID)
		c.conn.Close()
		close(c.closeChan)
	}()

	// Настраиваем соединение
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Бесконечный цикл чтения сообщений
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			}
			break
		}

		// Обрабатываем входящее сообщение
		c.handleIncomingMessage(message)
	}
}

// writePump отправляет сообщения клиенту
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Канал закрыт, отправляем сообщение о закрытии соединения
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Отправляем сообщение
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}
		case <-ticker.C:
			// Отправляем ping для поддержания соединения
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closeChan:
			// Соединение закрыто
			return
		}
	}
}

// handleIncomingMessage обрабатывает входящие сообщения от клиента
func (c *Client) handleIncomingMessage(message []byte) {
	// Парсим сообщение
	var event Event
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Error unmarshaling event: %v", err)
		return
	}

	// Проверяем, что userID в сообщении соответствует userID клиента
	// для предотвращения подделки отправителя
	if event.UserID != "" && event.UserID != c.UserID {
		log.Printf("UserID mismatch in message: %s vs %s", event.UserID, c.UserID)
		return
	}

	// Устанавливаем корректный userID и время
	event.UserID = c.UserID
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Обрабатываем различные типы событий
	switch event.Type {
	case EventTyping:
		if event.ChatID != "" {
			// TODO: Отправить уведомление о печати всем участникам чата
		}
	case EventStopTyping:
		if event.ChatID != "" {
			// TODO: Отправить уведомление о прекращении печати
		}
	case EventMessageRead:
		if event.ChatID != "" && event.MessageID != "" {
			// TODO: Обновить статус сообщения как прочитанное
			// и оповестить всех участников чата
		}
	// Другие типы событий могут быть обработаны здесь
	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}
}
