
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    phone VARCHAR(20) UNIQUE,
    role VARCHAR(10) NOT NULL CHECK (role IN ('admin', 'owner', 'client')),
    is_active BOOLEAN DEFAULT TRUE,
    session_data BYTEA,
    last_active TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE bots (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE bot_access (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bot_id BIGINT NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    access_level VARCHAR(20) NOT NULL CHECK (access_level IN ('owner', 'editor', 'viewer')),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, bot_id)
);

CREATE TABLE bot_templates (
    id BIGSERIAL PRIMARY KEY,
    bot_id BIGINT NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    content TEXT,
    keyboard JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE bot_states (
    bot_id BIGINT PRIMARY KEY REFERENCES bots(id) ON DELETE CASCADE,
    current_state JSONB,
    last_active TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_bots_owner_id ON bots(owner_id);
CREATE INDEX idx_bot_access_bot_id ON bot_access(bot_id);
CREATE INDEX idx_bot_templates_bot_id ON bot_templates(bot_id);

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_bots_timestamp
BEFORE UPDATE ON bots
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_bot_templates_timestamp
BEFORE UPDATE ON bot_templates
FOR EACH ROW EXECUTE FUNCTION update_timestamp();