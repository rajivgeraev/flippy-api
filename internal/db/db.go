package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajivgeraev/flippy-api/internal/config"
)

// Pool представляет пул соединений с базой данных
var Pool *pgxpool.Pool

// InitDB инициализирует соединение с базой данных
func InitDB(cfg *config.Config) error {
	var err error

	log.Printf("Подключение к базе данных: %s\n", cfg.DatabaseURL)

	// Создаем контекст с таймаутом для подключения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Настраиваем конфигурацию пула соединений
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("ошибка при разборе URL базы данных: %w", err)
	}

	// Дополнительная настройка пула соединений
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2

	// Создаем пул соединений
	Pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("ошибка при создании пула соединений: %w", err)
	}

	// Проверяем соединение
	if err = Pool.Ping(ctx); err != nil {
		return fmt.Errorf("ошибка при проверке соединения: %w", err)
	}

	log.Println("✅ Успешное подключение к базе данных")
	return nil
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	if Pool != nil {
		Pool.Close()
	}
}

// GetContext возвращает контекст с таймаутом для запросов к базе данных
func GetContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
