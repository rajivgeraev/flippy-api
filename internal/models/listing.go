package models

import (
	"encoding/json"
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
	ID         uuid.UUID     `json:"id"`
	ListingID  uuid.UUID     `json:"listing_id"`
	URL        string        `json:"url"`
	PreviewURL string        `json:"preview_url,omitempty"`
	PublicID   string        `json:"public_id"`
	FileName   string        `json:"file_name,omitempty"`
	IsMain     bool          `json:"is_main"`
	Position   int           `json:"position"`
	Metadata   ImageMetadata `json:"metadata,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// ImageMetadata содержит ключевые метаданные изображения из Cloudinary
type ImageMetadata struct {
	AssetID   string    `json:"asset_id"`
	PublicID  string    `json:"public_id"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
	Bytes     int       `json:"bytes"`
}

// CloudinaryResponse представляет ответ от Cloudinary API
type CloudinaryResponse struct {
	AssetID           string    `json:"asset_id"`
	PublicID          string    `json:"public_id"`
	Version           int       `json:"version"`
	VersionID         string    `json:"version_id"`
	Signature         string    `json:"signature"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
	Format            string    `json:"format"`
	ResourceType      string    `json:"resource_type"`
	CreatedAt         time.Time `json:"created_at"`
	Tags              []string  `json:"tags"`
	Pages             int       `json:"pages"`
	Bytes             int       `json:"bytes"`
	Type              string    `json:"type"`
	Etag              string    `json:"etag"`
	Placeholder       bool      `json:"placeholder"`
	URL               string    `json:"url"`
	SecureURL         string    `json:"secure_url"`
	AssetFolder       string    `json:"asset_folder"`
	DisplayName       string    `json:"display_name"`
	Context           Context   `json:"context"`
	OriginalFilename  string    `json:"original_filename"`
	OriginalExtension string    `json:"original_extension"`
	Eager             []Eager   `json:"eager"`
	APIKey            string    `json:"api_key"`
}

// Context содержит пользовательские метаданные в ответе Cloudinary
type Context struct {
	Custom CustomContext `json:"custom"`
}

// CustomContext содержит пользовательские поля в контексте Cloudinary
type CustomContext struct {
	UserID        string `json:"user_id"`
	UploadGroupID string `json:"upload_group_id"`
}

// Eager содержит информацию о трансформациях изображения
type Eager struct {
	Status    string `json:"status"`
	BatchID   string `json:"batch_id"`
	URL       string `json:"url"`
	SecureURL string `json:"secure_url"`
}

// ExtractMetadata извлекает основные метаданные из ответа Cloudinary
func ExtractMetadata(cr CloudinaryResponse) ImageMetadata {
	return ImageMetadata{
		AssetID:   cr.AssetID,
		PublicID:  cr.PublicID,
		Width:     cr.Width,
		Height:    cr.Height,
		CreatedAt: cr.CreatedAt,
		Bytes:     cr.Bytes,
	}
}

// ExtractPreviewURL извлекает URL превью из ответа Cloudinary
func ExtractPreviewURL(cr CloudinaryResponse) string {
	for _, eager := range cr.Eager {
		if eager.Status == "processing" || eager.Status == "completed" {
			return eager.SecureURL
		}
	}
	return ""
}

// ParseCloudinaryResponse конвертирует JSON-ответ от Cloudinary в структуру
func ParseCloudinaryResponse(jsonData string) (CloudinaryResponse, error) {
	var response CloudinaryResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	return response, err
}
