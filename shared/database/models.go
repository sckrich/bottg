package database

import (
	"time"

	"gorm.io/datatypes"
)

// Admin - администратор системы
type Admin struct {
	ID         uint   `gorm:"primaryKey"`
	TelegramID int64  `gorm:"uniqueIndex"`
	Username   string `gorm:"size:255"`
	IsActive   bool   `gorm:"default:true"`
	CreatedAt  time.Time
}

// Bot - зарегистрированный бот
type Bot struct {
	ID         uint   `gorm:"primaryKey"`
	Token      string `gorm:"uniqueIndex;size:255"`
	WebhookURL string `gorm:"size:512"`
	AdminID    uint   `gorm:"index"` // Связь с Admin
	IsActive   bool   `gorm:"default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// BotTemplate - шаблон ответов (граф)
type BotTemplate struct {
	ID        uint           `gorm:"primaryKey"`
	BotID     uint           `gorm:"index"` // Связь с Bot
	Name      string         `gorm:"size:255"`
	Graph     datatypes.JSON `gorm:"type:jsonb"` // JSON-структура графа
	CreatedAt time.Time
}

// BotUser - пользователь бота (для 2FA)
type BotUser struct {
	ID          uint   `gorm:"primaryKey"`
	BotID       uint   `gorm:"index"` // Связь с Bot
	TelegramID  int64  `gorm:"index"`
	Phone       string `gorm:"size:20"`
	SessionData []byte `gorm:"type:bytea"` // Сериализованная сессия MTProto
	LastActive  time.Time
	CreatedAt   time.Time
}

// ChatState - состояние диалога (кеш в Redis)
// NOTE: Эта модель не будет сохраняться в PostgreSQL,
// а только использоваться для документации структуры
type ChatState struct {
	ChatID      int64     `gorm:"-"` // Игнорировать для GORM
	CurrentNode string    `gorm:"-"`
	LastEvent   time.Time `gorm:"-"`
	Data        string    `gorm:"-"` // Дополнительные данные
}
