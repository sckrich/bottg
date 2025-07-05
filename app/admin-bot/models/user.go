package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleOwner  UserRole = "owner"
	RoleClient UserRole = "client"
)

type Admin struct {
	ID         uint   `gorm:"primaryKey"`
	TelegramID int64  `gorm:"uniqueIndex"`
	Username   string `gorm:"size:255"`
	IsActive   bool   `gorm:"default:true"`
	CreatedAt  time.Time
}

type Owner struct {
	ID          uint   `gorm:"primaryKey"`
	TelegramID  int64  `gorm:"uniqueIndex"`
	Username    string `gorm:"size:255"`
	CompanyName string `gorm:"size:255"`
	TaxID       string `gorm:"size:100"`
	IsVerified  bool   `gorm:"default:false"`
	BillingInfo JSONB  `gorm:"type:jsonb"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BotUser struct {
	ID          uint   `gorm:"primaryKey"`
	Phone       string `gorm:"uniqueIndex;size:20"`
	TelegramID  int64  `gorm:"index"`
	SessionData []byte `gorm:"type:bytea"`
	LastActive  time.Time
	CreatedAt   time.Time
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("неподдерживаемый тип для JSONB")
	}

	if len(data) == 0 {
		*j = nil
		return nil
	}
	return json.Unmarshal(data, j)
}
