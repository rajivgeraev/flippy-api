CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sender_listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    receiver_listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, accepted, rejected, canceled
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для оптимизации запросов
CREATE INDEX idx_trades_sender_id ON trades(sender_id);
CREATE INDEX idx_trades_receiver_id ON trades(receiver_id);
CREATE INDEX idx_trades_status ON trades(status);
CREATE INDEX idx_trades_sender_listing_id ON trades(sender_listing_id);
CREATE INDEX idx_trades_receiver_listing_id ON trades(receiver_listing_id);