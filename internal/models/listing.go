package models

import (
	"time"

	"github.com/google/uuid"
)

// Listing представляет объявление в системе
type Listing struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Categories  []string       `json:"categories"`
	AllowTrade  bool           `json:"allow_trade"`
	Status      string         `json:"status"`
	Images      []ListingImage `json:"images"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ListingImage представляет изображение объявления
type ListingImage struct {
	ID        uuid.UUID `json:"id"`
	ListingID uuid.UUID `json:"listing_id"`
	URL       string    `json:"url"`
	PublicID  string    `json:"public_id"`
	FileName  string    `json:"file_name"`
	IsMain    bool      `json:"is_main"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

// Category представляет категорию объявления
type Category struct {
	Slug   string `json:"slug"`
	NameRu string `json:"name_ru"`
	NameEn string `json:"name_en"`
}
