package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager представляет центральный менеджер для всех WebSocket соединений
type Manager struct {
	clients      map[uuid.UUID]*Client
	clientsMutex sync.RWMutex
	userClients  map[string]map[uuid.UUID]bool // userID -> map[clientID]bool
	userMutex    sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// EventType определяет тип события WebSocket
type EventType string

const (
	EventNewMessage       EventType = "new_message"
	EventMessageRead      EventType = "message_read"
	EventMessageDelivered EventType = "message_delivered"
	EventConnected        EventType = "connected"
	EventDisconnected     EventType = "disconnected"
	EventTyping           EventType = "typing"
	EventStopTyping       EventType = "stop_typing"
	EventUnreadCount      EventType = "unread_count"
)

// Event представляет структуру сообщения для WebSocket
type Event struct {
	Type      EventType       `json:"type"`
	ChatID    string          `json:"chat_id,omitempty"`
	MessageID string          `json:"message_id,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// NewManager создает новый экземпляр Manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		clients:     make(map[uuid.UUID]*Client),
		userClients: make(map[string]map[uuid.UUID]bool),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// AddClient регистрирует нового клиента
func (m *Manager) AddClient(client *Client) {
	m.clientsMutex.Lock()
	m.clients[client.ID] = client
	m.clientsMutex.Unlock()

	// Связываем клиент с пользователем
	m.userMutex.Lock()
	if _, exists := m.userClients[client.UserID]; !exists {
		m.userClients[client.UserID] = make(map[uuid.UUID]bool)
	}
	m.userClients[client.UserID][client.ID] = true
	m.userMutex.Unlock()

	log.Printf("WebSocket client %s connected for user %s", client.ID, client.UserID)
}

// RemoveClient удаляет клиента
func (m *Manager) RemoveClient(clientID uuid.UUID) {
	m.clientsMutex.RLock()
	client, exists := m.clients[clientID]
	m.clientsMutex.RUnlock()

	if !exists {
		return
	}

	userID := client.UserID

	// Удаляем клиент из связи с пользователем
	m.userMutex.Lock()
	if clients, ok := m.userClients[userID]; ok {
		delete(clients, clientID)
		// Если это был последний клиент пользователя, удаляем запись пользователя
		if len(clients) == 0 {
			delete(m.userClients, userID)
		}
	}
	m.userMutex.Unlock()

	// Удаляем клиент из общего списка
	m.clientsMutex.Lock()
	delete(m.clients, clientID)
	m.clientsMutex.Unlock()

	log.Printf("WebSocket client %s disconnected for user %s", clientID, userID)
}

// SendToUser отправляет сообщение всем соединениям конкретного пользователя
func (m *Manager) SendToUser(userID string, event Event) {
	if userID == "" {
		return
	}

	m.userMutex.RLock()
	clientIDs, exists := m.userClients[userID]
	m.userMutex.RUnlock()

	if !exists || len(clientIDs) == 0 {
		// Пользователь не онлайн, но сообщение все равно сохраняется в БД
		return
	}

	// Устанавливаем время события, если не установлено
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Отправляем событие всем соединениям пользователя
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	for clientID := range clientIDs {
		m.clientsMutex.RLock()
		client, exists := m.clients[clientID]
		m.clientsMutex.RUnlock()

		if !exists {
			continue
		}

		// Отправляем в неблокирующем режиме через горутину
		go func(c *Client) {
			select {
			case c.send <- eventJSON:
				// Сообщение успешно добавлено в очередь отправки
			default:
				// Канал заполнен, клиент слишком медленный - закрываем соединение
				log.Printf("Send channel full for client %s, closing connection", c.ID)
				c.conn.Close()
				m.RemoveClient(c.ID)
			}
		}(client)
	}
}

// SendToChat отправляет сообщение всем участникам чата
func (m *Manager) SendToChat(chatID string, event Event, excludeUserID string) {
	// В реальном приложении здесь должна быть логика получения участников чата из БД
	// и отправки сообщения всем участникам, кроме excludeUserID (обычно отправителя)

	// Для MVP мы можем упростить и использовать chatID для получения получателя
	// С учетом того, что у нас уже есть chats и messages таблицы

	// TODO: Реализовать логику получения участников чата
	// и отправки сообщения всем, кроме отправителя
}

// BroadcastUnreadCounts отправляет обновленное количество непрочитанных чатов пользователю
func (m *Manager) BroadcastUnreadCounts(userID string, unreadCounts int) {
	payload, _ := json.Marshal(map[string]int{"count": unreadCounts})

	m.SendToUser(userID, Event{
		Type:      EventUnreadCount,
		UserID:    userID,
		Timestamp: time.Now(),
		Payload:   payload,
	})
}

// Shutdown корректно завершает работу менеджера WebSocket
func (m *Manager) Shutdown() {
	m.cancel()

	m.clientsMutex.Lock()
	for _, client := range m.clients {
		client.conn.Close()
	}
	m.clients = make(map[uuid.UUID]*Client)
	m.clientsMutex.Unlock()

	m.userMutex.Lock()
	m.userClients = make(map[string]map[uuid.UUID]bool)
	m.userMutex.Unlock()
}
