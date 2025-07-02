package models

import (
	"time"
)

type Admin struct {
	ID         uint   `gorm:"primaryKey"`
	TelegramID int64  `gorm:"uniqueIndex"`
	Username   string `gorm:"size:255"`
	IsActive   bool   `gorm:"default:true"`
	CreatedAt  time.Time
}

type BotUser struct {
	ID          uint   `gorm:"primaryKey"`
	Phone       string `gorm:"uniqueIndex;size:20"`
	TelegramID  int64  `gorm:"index"`
	SessionData []byte `gorm:"type:bytea"`
	LastActive  time.Time
	CreatedAt   time.Time
}
