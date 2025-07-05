CREATE TABLE bot_users (
    id SERIAL PRIMARY KEY,
    phone TEXT UNIQUE NOT NULL,
    session BYTEA,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE bots (
    id SERIAL PRIMARY KEY,
    token TEXT UNIQUE NOT NULL,
    admin_id INTEGER REFERENCES admins(id),
    config JSONB NOT NULL
);