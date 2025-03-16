package models

import (
	"time"

	"github.com/google/uuid"
)

// Chat представляет чат между двумя пользователями
type Chat struct {
	ID              uuid.UUID  `json:"id"`
	TradeID         *uuid.UUID `json:"trade_id,omitempty"`
	SenderID        uuid.UUID  `json:"sender_id"`
	ReceiverID      uuid.UUID  `json:"receiver_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastMessageText string     `json:"last_message_text,omitempty"`
	LastMessageTime *time.Time `json:"last_message_time,omitempty"`
	IsActive        bool       `json:"is_active"`

	// Дополнительные поля для API
	Sender      *User  `json:"sender,omitempty"`
	Receiver    *User  `json:"receiver,omitempty"`
	Trade       *Trade `json:"trade,omitempty"`
	UnreadCount int    `json:"unread_count,omitempty"`
}

// Message представляет сообщение в чате
type Message struct {
	ID        uuid.UUID `json:"id"`
	ChatID    uuid.UUID `json:"chat_id"`
	SenderID  uuid.UUID `json:"sender_id"`
	Text      string    `json:"text"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Дополнительные поля для API
	Sender *User `json:"sender,omitempty"`
}
