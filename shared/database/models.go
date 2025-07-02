package database

import (
	"time"

	"gorm.io/datatypes"
)

type Admin struct {
	ID         uint   `gorm:"primaryKey"`
	TelegramID int64  `gorm:"uniqueIndex"`
	Username   string `gorm:"size:255"`
	IsActive   bool   `gorm:"default:true"`
	CreatedAt  time.Time
}

type Bot struct {
	ID         uint   `gorm:"primaryKey"`
	Token      string `gorm:"uniqueIndex;size:255"`
	WebhookURL string `gorm:"size:512"`
	AdminID    uint   `gorm:"index"`
	IsActive   bool   `gorm:"default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type BotTemplate struct {
	ID        uint           `gorm:"primaryKey"`
	BotID     uint           `gorm:"index"`
	Name      string         `gorm:"size:255"`
	Graph     datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time
}

type BotUser struct {
	ID          uint   `gorm:"primaryKey"`
	BotID       uint   `gorm:"index"`
	TelegramID  int64  `gorm:"index"`
	Phone       string `gorm:"size:20"`
	SessionData []byte `gorm:"type:bytea"`
	LastActive  time.Time
	CreatedAt   time.Time
}

type ChatState struct {
	ChatID      int64     `gorm:"-"`
	CurrentNode string    `gorm:"-"`
	LastEvent   time.Time `gorm:"-"`
	Data        string    `gorm:"-"`
}
