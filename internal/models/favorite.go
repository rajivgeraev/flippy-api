package models

import (
	"time"

	"github.com/google/uuid"
)

// Favorite представляет запись избранного объявления
type Favorite struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	ListingID uuid.UUID `json:"listing_id"`
	CreatedAt time.Time `json:"created_at"`

	// Дополнительные поля для API
	Listing *Listing `json:"listing,omitempty"`
}

// FavoriteResponse представляет структуру ответа API с избранными объявлениями
type FavoriteResponse struct {
	Favorites []Favorite `json:"favorites"`
	Total     int        `json:"total"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}
