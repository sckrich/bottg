package handlers

import (
	"admin-bot/models"
	"encoding/json"
	"log"
	"shared/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleCreateTemplate(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if !HandleAdminAuth(update) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Доступ запрещен")
		bot.Send(msg)
		return
	}

	var template models.BotTemplate
	if err := json.Unmarshal([]byte(update.Message.Text), &template); err != nil {
		log.Printf("Template parse error: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка формата шаблона")
		bot.Send(msg)
		return
	}

	if err := database.DB.Create(&template).Error; err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка сохранения: "+err.Error())
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ Шаблон сохранен")
	bot.Send(msg)
}

func GetTemplatesList(adminID int) ([]models.BotTemplate, error) {
	var templates []models.BotTemplate
	err := database.DB.Where("admin_id = ?", adminID).Find(&templates).Error
	return templates, err
}
