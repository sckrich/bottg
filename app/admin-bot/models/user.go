package models

import (
	"time"
)

type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleOwner  UserRole = "owner"
	RoleClient UserRole = "client"
)

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
