package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	TelegramBot string
	JWTSecret   string
}

// Глобальная конфигурация
var AppConfig Config

func LoadConfig() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	AppConfig = Config{
		Port:        os.Getenv("PORT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		TelegramBot: os.Getenv("TELEGRAM_BOT_TOKEN"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
}
