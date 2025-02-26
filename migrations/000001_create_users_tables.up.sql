CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Создание таблицы пользователей
CREATE TABLE
    IF NOT EXISTS users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        username VARCHAR(255) UNIQUE,
        first_name VARCHAR(255),
        last_name VARCHAR(255),
        email VARCHAR(255) UNIQUE,
        phone VARCHAR(50),
        bio TEXT,
        avatar_url TEXT,
        location VARCHAR(255),
        created_at TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            last_login_at TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            is_active BOOLEAN DEFAULT TRUE
    );

-- Создание таблицы для пользователей телеграм
CREATE TABLE
    IF NOT EXISTS telegram_users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        telegram_id BIGINT NOT NULL UNIQUE,
        username VARCHAR(255),
        first_name VARCHAR(255),
        last_name VARCHAR(255),
        photo_url TEXT,
        is_premium BOOLEAN DEFAULT FALSE,
        language_code VARCHAR(10),
        raw_data JSONB,
        created_at TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

-- Создание таблицы для истории изменений пользователей
CREATE TABLE
    IF NOT EXISTS user_history (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        reference_table VARCHAR(50) NOT NULL,
        reference_id UUID NOT NULL,
        data JSONB NOT NULL,
        created_at TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

-- Создание таблицы для сессий пользователей
CREATE TABLE
    IF NOT EXISTS user_sessions (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        login_time TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            logout_time TIMESTAMP
        WITH
            TIME ZONE,
            last_active TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            device_info TEXT,
            ip_address VARCHAR(45),
            created_at TIMESTAMP
        WITH
            TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

-- Индексы для оптимизации запросов
CREATE INDEX idx_telegram_users_user_id ON telegram_users (user_id);

CREATE INDEX idx_telegram_users_telegram_id ON telegram_users (telegram_id);

CREATE INDEX idx_user_history_user_id ON user_history (user_id);

CREATE INDEX idx_user_history_reference_table_reference_id ON user_history (reference_table, reference_id);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);

CREATE INDEX idx_users_username ON users (username)
WHERE
    username IS NOT NULL;

CREATE INDEX idx_users_email ON users (email)
WHERE
    email IS NOT NULL;