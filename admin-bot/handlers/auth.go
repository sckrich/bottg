package handlers

import (
	"admin-bot/models"
	"log"

	"shared/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleAdminAuth(update tgbotapi.Update) bool {
	telegramID := update.Message.From.ID

	var admin models.Admin
	if err := database.DB.Where("telegram_id = ?", telegramID).First(&admin).Error; err != nil {
		log.Printf("Unauthorized access attempt by %d", telegramID)
		return false
	}

	return admin.IsActive
}

func RegisterAdmin(telegramID int64) error {
	admin := models.Admin{
		TelegramID: telegramID,
		IsActive:   true,
	}
	return database.DB.Create(&admin).Error
}
