package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

type UserState struct {
	CurrentAction string
	TempData      map[string]interface{}
	CreatedAt     time.Time
	LastActivity  time.Time
}

var (
	bot        *tgbotapi.BotAPI
	db         *sql.DB
	userStates = make(map[int64]*UserState)
	stateMutex sync.RWMutex
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
	ID        int64           `db:"id" json:"id"`
	UserID    int64           `db:"user_id" json:"user_id"`
	Name      string          `db:"name" json:"name"`
	Content   string          `db:"content" json:"content"`
	Keyboard  json.RawMessage `db:"keyboard" json:"keyboard"` // Используем RawMessage
	IsActive  bool            `db:"is_active" json:"is_active"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

// Методы для работы с базой
func (t *BotTemplate) Save() error {
	query := `
        INSERT INTO bot_templates 
        (user_id, name, content, keyboard, is_active)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at`

	return db.QueryRow(
		query,
		t.UserID,
		t.Name,
		t.Content,
		t.Keyboard,
		t.IsActive,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
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
