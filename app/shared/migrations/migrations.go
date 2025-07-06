package database

import (
	"admin-bot/models"
	"log"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.BotTemplate{},
	)
	if err != nil {
		return err
	}
	log.Println("Миграции успешно выполнены")
	return nil
}
