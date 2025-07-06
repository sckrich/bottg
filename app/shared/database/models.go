package database

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Bot struct {
	ID           uint           `gorm:"primaryKey"`
	OwnerID      uint           `gorm:"index"`
	Token        string         `gorm:"uniqueIndex;size:46"`
	Username     string         `gorm:"size:32"`
	WebhookURL   string         `gorm:"size:512"`
	CurrentState datatypes.JSON `gorm:"type:jsonb"`
	IsActive     bool           `gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BotTemplate struct {
	ID        uint            `gorm:"primaryKey"`
	UserID    int64           `db:"user_id" json:"user_id"`
	BotID     uint            `gorm:"index"`
	Name      string          `gorm:"size:255;index"`
	Content   string          `gorm:"type:text"`
	Keyboard  json.RawMessage `db:"keyboard" json:"keyboard"` // Используем RawMessage
	IsActive  bool            `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BotAccess struct {
	ID          uint   `gorm:"primaryKey"`
	BotID       uint   `gorm:"index"`
	UserID      uint   `gorm:"index"`
	AccessLevel string `gorm:"size:20;index"`
	CreatedAt   time.Time
}

type User struct {
	ID          uint   `gorm:"primaryKey"`
	TelegramID  int64  `gorm:"uniqueIndex;not null"`
	Username    string `gorm:"size:255"`
	Phone       string `gorm:"size:20;unique"`
	Role        string `gorm:"size:10;not null;check:role IN ('admin', 'owner', 'client')"`
	IsActive    bool   `gorm:"default:true"`
	SessionData []byte `gorm:"type:bytea"`
	LastActive  time.Time
	CreatedAt   time.Time `gorm:"default:now()"`
}

type ChatState struct {
	ID          uint           `gorm:"primaryKey"`
	ChatID      int64          `gorm:"index"`
	BotID       uint           `gorm:"index"`
	CurrentNode string         `gorm:"size:100"`
	StateData   datatypes.JSON `gorm:"type:jsonb"`
	LastActive  time.Time      `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ChatState) TableName() string {
	return "chat_states"
}

func (b *Bot) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now
	return nil
}

func (b *Bot) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = time.Now()
	return nil
}
