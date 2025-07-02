package handlers

import (
	"log"
	"shared/database"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleAddBot(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	adminID := uint(update.Message.From.ID)

	newBot := database.Bot{
		Token:    update.Message.Text,
		AdminID:  adminID,
		IsActive: true,
	}

	if err := database.DB.Create(&newBot).Error; err != nil {
		log.Printf("Ошибка создания бота: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Ошибка при создании бота: "+err.Error())
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ Бот успешно добавлен!\nID: "+strconv.FormatUint(uint64(newBot.ID), 10))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
