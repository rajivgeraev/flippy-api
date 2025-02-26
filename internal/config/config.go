package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config структура конфигурации
type Config struct {
	TelegramBotToken string
	JWTSecret        string
	DatabaseURL      string
	DatabaseConfig   DatabaseConfig
}

// DatabaseConfig содержит конфигурацию базы данных
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// LoadConfig загружает переменные из .env
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ .env файл не найден, используем переменные окружения")
	}

	dbConfig := DatabaseConfig{
		Host:     getEnv("PGHOST", "localhost"),
		Port:     getEnv("PGPORT", "5432"),
		User:     getEnv("PGUSER", "flippy_user"),
		Password: getEnv("PGPASSWORD", "flippy_pass"),
		Name:     getEnv("PGDATABASE", "flippy"),
		SSLMode:  getEnv("PGSSLMODE", "disable"),
	}

	// Формируем строку подключения к базе данных
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Name, dbConfig.SSLMode)

	cfg := &Config{
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		DatabaseURL:      dbURL,
		DatabaseConfig:   dbConfig,
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
