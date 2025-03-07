-- 000002_create_listings_tables.up.sql

-- Справочная таблица категорий
CREATE TABLE category_references (
    slug VARCHAR(50) PRIMARY KEY,
    name_ru VARCHAR(100) NOT NULL,
    name_en VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Таблица объявлений
CREATE TABLE listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    categories JSONB DEFAULT '[]',
    allow_trade BOOLEAN DEFAULT TRUE,
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- Добавили статус 'draft'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Таблица изображений объявлений
CREATE TABLE listing_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    preview_url TEXT,
    public_id VARCHAR(255) NOT NULL,
    file_name VARCHAR(255),
    is_main BOOLEAN DEFAULT FALSE,
    position INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для таблицы listings
CREATE INDEX idx_listings_user_id ON listings(user_id);
CREATE INDEX idx_listings_status ON listings(status);
CREATE INDEX idx_listings_created_at ON listings(created_at DESC);
CREATE INDEX idx_listings_categories ON listings USING gin(categories);

-- Индексы для таблицы listing_images
CREATE INDEX idx_listing_images_listing_id ON listing_images(listing_id);
CREATE INDEX idx_listing_images_listing_position ON listing_images(listing_id, position);
CREATE UNIQUE INDEX idx_listing_images_main ON listing_images(listing_id) WHERE is_main = TRUE;

-- Ограничение на уникальность позиции внутри одного объявления
ALTER TABLE listing_images ADD CONSTRAINT unique_position_per_listing 
    UNIQUE (listing_id, position);

-- Вставка базовых категорий
INSERT INTO category_references (slug, name_ru, name_en) VALUES
('dolls', 'Куклы', 'Dolls'),
('cars', 'Машинки', 'Cars'),
('construction', 'Конструкторы', 'Construction Sets'),
('plush', 'Мягкие игрушки', 'Plush Toys'),
('board_games', 'Настольные игры', 'Board Games'),
('educational', 'Развивающие игрушки', 'Educational Toys'),
('outdoor', 'Игрушки для улицы', 'Outdoor Toys'),
('creative', 'Творчество', 'Creative Arts & Crafts'),
('electronic', 'Электронные игрушки', 'Electronic Toys'),
('wooden', 'Деревянные игрушки', 'Wooden Toys'),
('baby', 'Игрушки для малышей', 'Baby Toys'),
('puzzles', 'Головоломки и пазлы', 'Puzzles'),
('lego', 'LEGO', 'LEGO'),
('action_figures', 'Экшн-фигурки', 'Action Figures'),
('other', 'Другое', 'Other');