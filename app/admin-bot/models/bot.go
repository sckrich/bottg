package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Bot struct {
	ID        int64     `db:"id" json:"id"`
	OwnerID   int64     `db:"owner_id" json:"owner_id"`
	Token     string    `gorm:"unique"`
	Username  string    `db:"username" json:"username"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type BotAccess struct {
	UserID      int64     `db:"user_id" json:"user_id"`
	BotID       int64     `db:"bot_id" json:"bot_id"`
	AccessLevel string    `db:"access_level" json:"access_level"`
	GrantedAt   time.Time `db:"granted_at" json:"granted_at"`
}
type BotTemplate struct {
	BotID     int64           `db:"bot_id" json:"bot_id"`
	ID        int64           `db:"id" json:"id"`
	UserID    int64           `db:"user_id" json:"user_id"`
	Name      string          `db:"name" json:"name"`
	Content   string          `db:"content" json:"content"`
	Keyboard  json.RawMessage `db:"keyboard" json:"keyboard"`
	IsActive  bool            `db:"is_active" json:"is_active"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

type BotState struct {
	BotID        int64     `db:"bot_id" json:"bot_id"`
	CurrentState StateData `db:"current_state" json:"current_state"`
	LastActive   time.Time `db:"last_active" json:"last_active"`
}

type StateData map[string]interface{}

func (s StateData) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StateData) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, &s)
}
