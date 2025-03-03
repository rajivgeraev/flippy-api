package cloudinary

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rajivgeraev/flippy-api/internal/config"
	"github.com/rajivgeraev/flippy-api/internal/utils"
)

// CloudinaryService предоставляет методы для работы с Cloudinary
type CloudinaryService struct {
	cfg        *config.Config
	jwtService *utils.JWTService

	uploadPreset string
}

// NewCloudinaryService создает новый экземпляр CloudinaryService
func NewCloudinaryService(cfg *config.Config) *CloudinaryService {
	return &CloudinaryService{
		cfg:          cfg,
		jwtService:   utils.NewJWTService(cfg.JWTSecret),
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
	// Получаем `userID` из контекста
	userID := c.Locals("userID").(string)

	// Генерируем `listing_id`, если его нет
	listingID := c.Query("listing_id")
	if listingID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "listing_id is required")
	}
	// Текущий timestamp
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Формируем context (добавляем `listing_id`, если он передан)
	context := fmt.Sprintf("user_id=%s|listing_id=%s", userID, listingID)

	// Поля, которые подписываем
	signParams := map[string]string{
		"timestamp":     timestamp,
		"context":       context,
		"upload_preset": s.cfg.CloudinaryConfig.UploadPreset,
	}

	// Генерируем подпись
	signature := s.GenerateSignature(signParams)

	// Формируем ответ
	return c.JSON(fiber.Map{
		"api_key":       s.cfg.CloudinaryConfig.APIKey,
		"cloud_name":    s.cfg.CloudinaryConfig.CloudName,
		"upload_preset": s.cfg.CloudinaryConfig.UploadPreset,
		"context":       context,
		"timestamp":     timestamp,
		"signature":     signature,
		"listing_id":    listingID,
	})
}
