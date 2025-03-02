package cloudinary

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rajivgeraev/flippy-api/internal/config"
)

// CloudinaryService предоставляет методы для работы с Cloudinary
type CloudinaryService struct {
	cfg          *config.Config
	uploadFolder string
	uploadPreset string
}

// NewCloudinaryService создает новый экземпляр CloudinaryService
func NewCloudinaryService(cfg *config.Config) *CloudinaryService {
	return &CloudinaryService{
		cfg:          cfg,
		uploadFolder: cfg.CloudinaryConfig.UploadFolder,
		uploadPreset: cfg.CloudinaryConfig.UploadPreset,
	}
}

// GenerateSignature создаёт корректную подпись для Cloudinary
func (s *CloudinaryService) GenerateSignature(params map[string]string) string {
	// Сортируем ключи параметров
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Формируем строку для подписи
	var signParts []string
	for _, k := range keys {
		signParts = append(signParts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	signatureString := strings.Join(signParts, "&")

	// Добавляем API-секрет в конец строки
	signatureString += s.cfg.CloudinaryConfig.APISecret

	// Создаем SHA-1 хеш
	h := sha1.New()
	h.Write([]byte(signatureString))

	// Возвращаем подпись в виде шестнадцатеричной строки
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateUploadParams создаёт параметры для загрузки изображений
func (s *CloudinaryService) GenerateUploadParams(c fiber.Ctx) error {
	// Генерируем ID для объявления, если не передан
	listingID := c.Query("listing_id")
	if listingID == "" {
		listingID = uuid.New().String()
	}

	// Текущий timestamp
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Параметры для подписи
	params := map[string]string{
		"timestamp": timestamp,
	}

	// Генерируем подпись
	signature := s.GenerateSignature(params)

	// Возвращаем параметры
	return c.JSON(fiber.Map{
		"timestamp":  timestamp,
		"signature":  signature,
		"api_key":    s.cfg.CloudinaryConfig.APIKey,
		"cloud_name": s.cfg.CloudinaryConfig.CloudName,
		"listing_id": listingID,
	})
}
