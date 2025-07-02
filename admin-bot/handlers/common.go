package handlers

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

type HandlerConfig struct {
	Bot *tgbotapi.BotAPI
	DB  *gorm.DB
}

func InitHandlers(config HandlerConfig) {

}

func RouteUpdate(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}

	switch update.Message.Command() {
	case "start":
		handleStart(update.Message, bot)
	default:
		handleUnknownCommand(update.Message, bot)
	}
}

func handleStart(message *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Добро пожаловать в бота!")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func handleUnknownCommand(message *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Попробуйте /start.")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
