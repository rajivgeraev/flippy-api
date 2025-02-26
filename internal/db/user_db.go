package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// User представляет пользователя в системе
type User struct {
	ID          uuid.UUID
	Username    string
	FirstName   string
	LastName    string
	Email       string
	Phone       string
	Bio         string
	AvatarURL   string
	Location    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt time.Time
	IsActive    bool
}

// TelegramUser представляет данные пользователя из Telegram
type TelegramUser struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	TelegramID   int64
	Username     string
	FirstName    string
	LastName     string
	PhotoURL     string
	IsPremium    bool
	LanguageCode string
	RawData      []byte // JSONB данные
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateOrUpdateTelegramUser создает нового пользователя через Telegram или обновляет существующего
func CreateOrUpdateTelegramUser(telegramID int64, username, firstName, lastName, photoURL string,
	isPremium bool, languageCode string, rawData []byte) (*User, error) {
	ctx, cancel := GetContext()
	defer cancel()

	// Начинаем транзакцию
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка при начале транзакции: %w", err)
	}
	defer tx.Rollback(ctx) // Откатываем транзакцию в случае ошибки

	// Проверяем, существует ли пользователь Telegram
	var telegramUserID uuid.UUID
	var userID uuid.UUID
	var exists bool

	row := tx.QueryRow(ctx, `
		SELECT id, user_id, true FROM telegram_users WHERE telegram_id = $1
	`, telegramID)

	err = row.Scan(&telegramUserID, &userID, &exists)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("ошибка при проверке существования пользователя Telegram: %w", err)
	}

	// Если пользователь не существует, создаем нового
	if err == pgx.ErrNoRows {
		// Создаем запись в users
		err = tx.QueryRow(ctx, `
			INSERT INTO users (first_name, last_name, username, avatar_url, last_login_at)
			VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
			RETURNING id
		`, firstName, lastName, username, photoURL).Scan(&userID)

		if err != nil {
			return nil, fmt.Errorf("ошибка при создании пользователя: %w", err)
		}

		// Создаем запись в telegram_users
		err = tx.QueryRow(ctx, `
			INSERT INTO telegram_users (user_id, telegram_id, username, first_name, last_name, photo_url, is_premium, language_code, raw_data)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`, userID, telegramID, username, firstName, lastName, photoURL, isPremium, languageCode, rawData).Scan(&telegramUserID)

		if err != nil {
			return nil, fmt.Errorf("ошибка при создании Telegram пользователя: %w", err)
		}

		// Добавляем записи в history
		if err = addToUserHistory(ctx, tx, userID, "users", userID); err != nil {
			return nil, err
		}

		if err = addToUserHistory(ctx, tx, userID, "telegram_users", telegramUserID); err != nil {
			return nil, err
		}
	} else {
		// Обновляем только last_login_at у существующего пользователя
		_, err = tx.Exec(ctx, `
			UPDATE users 
			SET last_login_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, userID)

		if err != nil {
			return nil, fmt.Errorf("ошибка при обновлении времени входа пользователя: %w", err)
		}

		// Получаем текущие данные пользователя для сравнения
		var currentTgUser struct {
			Username     pgtype.Text
			FirstName    pgtype.Text
			LastName     pgtype.Text
			PhotoURL     pgtype.Text
			IsPremium    bool
			LanguageCode pgtype.Text
		}

		err = tx.QueryRow(ctx, `
			SELECT username, first_name, last_name, photo_url, is_premium, language_code
			FROM telegram_users
			WHERE id = $1
		`, telegramUserID).Scan(
			&currentTgUser.Username,
			&currentTgUser.FirstName,
			&currentTgUser.LastName,
			&currentTgUser.PhotoURL,
			&currentTgUser.IsPremium,
			&currentTgUser.LanguageCode,
		)

		if err != nil {
			return nil, fmt.Errorf("ошибка при получении текущих данных Telegram пользователя: %w", err)
		}

		// Проверяем, есть ли изменения
		hasChanges := false

		// Преобразуем nullable поля для сравнения
		currentUsername := ""
		if currentTgUser.Username.Valid {
			currentUsername = currentTgUser.Username.String
		}

		currentFirstName := ""
		if currentTgUser.FirstName.Valid {
			currentFirstName = currentTgUser.FirstName.String
		}

		currentLastName := ""
		if currentTgUser.LastName.Valid {
			currentLastName = currentTgUser.LastName.String
		}

		currentPhotoURL := ""
		if currentTgUser.PhotoURL.Valid {
			currentPhotoURL = currentTgUser.PhotoURL.String
		}

		currentLanguageCode := ""
		if currentTgUser.LanguageCode.Valid {
			currentLanguageCode = currentTgUser.LanguageCode.String
		}

		// Сравниваем значения
		if username != currentUsername ||
			firstName != currentFirstName ||
			lastName != currentLastName ||
			photoURL != currentPhotoURL ||
			isPremium != currentTgUser.IsPremium ||
			languageCode != currentLanguageCode {
			hasChanges = true
		}

		// Обновляем данные telegram_users
		_, err = tx.Exec(ctx, `
			UPDATE telegram_users 
			SET username = $1, first_name = $2, last_name = $3, photo_url = $4, 
				is_premium = $5, language_code = $6, raw_data = $7, updated_at = CURRENT_TIMESTAMP
			WHERE id = $8
		`, username, firstName, lastName, photoURL, isPremium, languageCode, rawData, telegramUserID)

		if err != nil {
			return nil, fmt.Errorf("ошибка при обновлении Telegram пользователя: %w", err)
		}

		// Добавляем запись в историю только если были изменения
		if hasChanges {
			if err = addToUserHistory(ctx, tx, userID, "telegram_users", telegramUserID); err != nil {
				return nil, err
			}
		}
	}

	// Создаем запись в user_sessions
	_, err = tx.Exec(ctx, `
		INSERT INTO user_sessions (user_id, login_time)
		VALUES ($1, CURRENT_TIMESTAMP)
	`, userID)

	if err != nil {
		return nil, fmt.Errorf("ошибка при создании сессии пользователя: %w", err)
	}

	// Получаем пользователя
	user, err := getUserByID(ctx, tx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении пользователя: %w", err)
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ошибка при фиксации транзакции: %w", err)
	}

	return user, nil
}

// getUserByID получает пользователя по ID внутри транзакции
func getUserByID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*User, error) {
	var user User
	var username, firstName, lastName, email, phone, bio, avatarURL, location pgtype.Text

	err := tx.QueryRow(ctx, `
		SELECT id, username, first_name, last_name, email, phone, bio, avatar_url, 
			   location, created_at, updated_at, last_login_at, is_active
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &username, &firstName, &lastName,
		&email, &phone, &bio, &avatarURL,
		&location, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &user.IsActive,
	)

	if err != nil {
		return nil, err
	}

	// Преобразуем nullable поля
	if username.Valid {
		user.Username = username.String
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if location.Valid {
		user.Location = location.String
	}

	return &user, nil
}

// GetUserByID получает пользователя по ID (публичная версия)
func GetUserByID(userID uuid.UUID) (*User, error) {
	ctx, cancel := GetContext()
	defer cancel()

	var user User
	var username, firstName, lastName, email, phone, bio, avatarURL, location pgtype.Text

	err := Pool.QueryRow(ctx, `
		SELECT id, username, first_name, last_name, email, phone, bio, avatar_url, 
			   location, created_at, updated_at, last_login_at, is_active
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &username, &firstName, &lastName,
		&email, &phone, &bio, &avatarURL,
		&location, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &user.IsActive,
	)

	if err != nil {
		return nil, err
	}

	// Преобразуем nullable поля
	if username.Valid {
		user.Username = username.String
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if location.Valid {
		user.Location = location.String
	}

	return &user, nil
}

// GetUserByTelegramID получает пользователя по ID Telegram
func GetUserByTelegramID(telegramID int64) (*User, error) {
	ctx, cancel := GetContext()
	defer cancel()

	var userID uuid.UUID

	err := Pool.QueryRow(ctx, `
		SELECT user_id FROM telegram_users WHERE telegram_id = $1
	`, telegramID).Scan(&userID)

	if err != nil {
		return nil, err
	}

	return GetUserByID(userID)
}

// addToUserHistory добавляет запись в историю изменений пользователя
func addToUserHistory(ctx context.Context, tx pgx.Tx, userID uuid.UUID, tableName string, referenceID uuid.UUID) error {
	var data []byte
	var err error

	// Получаем текущие данные из таблицы
	switch tableName {
	case "users":
		var user User
		var username, firstName, lastName, email, phone, bio, avatarURL, location pgtype.Text

		err = tx.QueryRow(ctx, `
			SELECT id, username, first_name, last_name, email, phone, bio, avatar_url, 
				   location, created_at, updated_at, last_login_at, is_active
			FROM users WHERE id = $1
		`, referenceID).Scan(
			&user.ID, &username, &firstName, &lastName,
			&email, &phone, &bio, &avatarURL,
			&location, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &user.IsActive,
		)

		if err != nil {
			return fmt.Errorf("ошибка при получении данных пользователя: %w", err)
		}

		// Преобразуем nullable поля
		if username.Valid {
			user.Username = username.String
		}
		if firstName.Valid {
			user.FirstName = firstName.String
		}
		if lastName.Valid {
			user.LastName = lastName.String
		}
		if email.Valid {
			user.Email = email.String
		}
		if phone.Valid {
			user.Phone = phone.String
		}
		if bio.Valid {
			user.Bio = bio.String
		}
		if avatarURL.Valid {
			user.AvatarURL = avatarURL.String
		}
		if location.Valid {
			user.Location = location.String
		}

		data, err = json.Marshal(user)

	case "telegram_users":
		var telegramUser TelegramUser
		var username, firstName, lastName, photoURL, languageCode pgtype.Text
		var isPremium bool

		err = tx.QueryRow(ctx, `
			SELECT id, user_id, telegram_id, username, first_name, last_name, photo_url, 
			       is_premium, language_code, raw_data, created_at, updated_at
			FROM telegram_users WHERE id = $1
		`, referenceID).Scan(
			&telegramUser.ID, &telegramUser.UserID, &telegramUser.TelegramID,
			&username, &firstName, &lastName, &photoURL,
			&isPremium, &languageCode, &telegramUser.RawData,
			&telegramUser.CreatedAt, &telegramUser.UpdatedAt,
		)

		if err != nil {
			return fmt.Errorf("ошибка при получении данных Telegram пользователя: %w", err)
		}

		// Преобразуем nullable поля
		if username.Valid {
			telegramUser.Username = username.String
		}
		if firstName.Valid {
			telegramUser.FirstName = firstName.String
		}
		if lastName.Valid {
			telegramUser.LastName = lastName.String
		}
		if photoURL.Valid {
			telegramUser.PhotoURL = photoURL.String
		}
		if languageCode.Valid {
			telegramUser.LanguageCode = languageCode.String
		}

		telegramUser.IsPremium = isPremium

		data, err = json.Marshal(telegramUser)

	default:
		return fmt.Errorf("неизвестное имя таблицы: %s", tableName)
	}

	if err != nil {
		return fmt.Errorf("ошибка при сериализации данных: %w", err)
	}

	// Добавляем запись в историю
	_, err = tx.Exec(ctx, `
		INSERT INTO user_history (user_id, reference_table, reference_id, data)
		VALUES ($1, $2, $3, $4)
	`, userID, tableName, referenceID, data)

	if err != nil {
		return fmt.Errorf("ошибка при добавлении записи в историю: %w", err)
	}

	return nil
}
