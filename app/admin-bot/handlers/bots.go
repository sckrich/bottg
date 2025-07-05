package handlers

import (
	"log"
	"shared/database"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func HandleAddBot(update tgbotapi.Update, bot *tgbotapi.BotAPI, db *gorm.DB) {

	if !isAdmin(update.Message.From.ID) {
		sendErrorMessage(bot, update.Message.Chat.ID, "❌ Доступ запрещен: недостаточно прав")
		return
	}

	token := update.Message.Text
	if !isValidBotToken(token) {
		sendErrorMessage(bot, update.Message.Chat.ID, "❌ Неверный формат токена бота")
		return
	}

	newBot := database.Bot{
		Token:      token,
		OwnerID:    getAdminID(update.Message.From.ID),
		IsActive:   true,
		WebhookURL: generateWebhookURL(token),
	}

	if err := db.Create(&newBot).Error; err != nil {
		log.Printf("Ошибка создания бота: %v", err)
		sendErrorMessage(bot, update.Message.Chat.ID, "❌ Ошибка при создании бота")
		return
	}
	botAccess := database.BotAccess{
		BotID:       newBot.ID,
		UserID:      newBot.OwnerID,
		AccessLevel: "owner",
	}

	if err := db.Create(&botAccess).Error; err != nil {
		log.Printf("Ошибка создания прав доступа: %v", err)

		db.Delete(&newBot)
		sendErrorMessage(bot, update.Message.Chat.ID, "❌ Ошибка при настройке прав доступа")
		return
	}

	response := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"✅ Бот успешно добавлен!\n"+
			"ID: "+strconv.FormatUint(uint64(newBot.ID), 10)+"\n"+
			"Username: @"+newBot.Username+"\n"+
			"Webhook: "+newBot.WebhookURL,
	)

	if _, err := bot.Send(response); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func isAdmin(userID int64) bool {
	var admin database.Admin
	err := database.DB.Where("telegram_id = ?", userID).First(&admin).Error
	return err == nil && admin.IsActive
}

func getAdminID(telegramID int64) uint {
	var admin database.Admin
	if err := database.DB.Where("telegram_id = ?", telegramID).First(&admin).Error; err != nil {
		return 0
	}
	return admin.ID
}

func isValidBotToken(token string) bool {

	return len(token) > 30 && strings.Contains(token, ":")
}

func generateWebhookURL(token string) string {
	return "https://yourdomain.com/webhook/" + token[:20]
}

func sendErrorMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
