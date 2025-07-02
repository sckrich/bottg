package main

import (
	"log"
	"os"
	"shared/database"

	"admin-bot/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	err := database.Init()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer database.Close()

	// Создание бота
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// Инициализация обработчиков
	handlers.InitHandlers(handlers.HandlerConfig{
		Bot: bot,
		DB:  database.DB,
	})

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		handlers.RouteUpdate(update, bot) // Передаем объект бота
	}
}
