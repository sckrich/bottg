package models

import (
	"time"

	"gorm.io/datatypes"
)

type BotTemplate struct {
	ID        uint `gorm:"primaryKey"`
	BotID     uint
	Name      string         `gorm:"size:255"`
	Graph     datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time
}
