package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config структура конфигурации
type Config struct {
	TelegramBotToken string
	JWTSecret        string
}

// LoadConfig загружает переменные из .env
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ .env файл не найден, используем переменные окружения")
	}

	cfg := &Config{
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		JWTSecret:        getEnv("JWT_SECRET", ""),
	}

	if cfg.TelegramBotToken == "" || cfg.JWTSecret == "" {
		log.Fatal("❌ Ошибка: Не заданы обязательные переменные окружения")
	}

	return cfg
}

// getEnv получает переменную окружения или использует дефолтное значение
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
