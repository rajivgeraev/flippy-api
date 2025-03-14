package models

import (
	"time"

	"github.com/google/uuid"
)

// Trade представляет предложение об обмене
// Trade представляет предложение об обмене
type Trade struct {
	ID                uuid.UUID `json:"id"`
	SenderID          uuid.UUID `json:"sender_id"`
	ReceiverID        uuid.UUID `json:"receiver_id"`
	SenderListingID   uuid.UUID `json:"sender_listing_id"`
	ReceiverListingID uuid.UUID `json:"receiver_listing_id"`
	Status            string    `json:"status"` // pending, accepted, rejected, canceled
	Message           string    `json:"message"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Дополнительные поля для API
	SenderListing   *Listing  `json:"sender_listing,omitempty"`
	ReceiverListing *Listing  `json:"receiver_listing,omitempty"`
	Sender          *User     `json:"sender,omitempty"`
	Receiver        *User     `json:"receiver,omitempty"`
	ChatID          uuid.UUID `json:"chat_id,omitempty"` // ID связанного чата
}

// User представляет минимальную информацию о пользователе для API
type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username,omitempty"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
}
