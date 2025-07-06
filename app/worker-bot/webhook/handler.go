package webhook

import (
	"context"
	"log"
	"net/http"
	"time"
	"worker-bot/models"
	mtproto "worker-bot/mt-proto"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type WebhookConfig struct {
	Token      string
	URL        string
	ListenAddr string
}

func Start(cfg WebhookConfig, redis *models.RedisClient, mtp *mtproto.Session) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	webhookURL := cfg.URL + "/webhook"
	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		log.Fatalf("Failed to create webhook: %v", err)
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatalf("Failed to set webhook: %v", err)
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		update, err := bot.HandleUpdate(r)
		if err != nil {
			log.Printf("Error handling update: %v", err)
			return
		}

		if update.Message != nil {
			handleMessage(bot, update.Message, redis, mtp)
		}
	})

	log.Printf("Starting server on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, redis *models.RedisClient, mtp *mtproto.Session) {
	ctx := context.Background()
	userID := msg.From.ID

	state, err := redis.GetBotState(ctx, userID)
	if err != nil {
		log.Printf("Error getting bot state: %v", err)
		return
	}
	if state == nil {
		state = &models.BotState{
			UserID:      userID,
			CurrentStep: "start",
			RefCode:     "ref_default",
			LastActive:  time.Now(),
			IsBlocked:   false,
		}
	}

	state.LastActive = time.Now()

	switch {
	case msg.IsCommand() && msg.Command() == "start":
		handleStartCommand(bot, msg.Chat.ID, state, redis)
	case msg.IsCommand() && msg.Command() == "auth":
		handleAuthCommand(bot, msg.Chat.ID, state, redis, mtp)
	default:
		handleRegularMessage(bot, msg, state, redis)
	}
	if err := redis.SaveBotState(ctx, state); err != nil {
		log.Printf("Error saving bot state: %v", err)
	}
}

func handleStartCommand(bot *tgbotapi.BotAPI, chatID int64, state *models.BotState, redis *models.RedisClient) {
	state.CurrentStep = "start"

	msg := tgbotapi.NewMessage(chatID, "Добро пожаловать! Ваш реферальный код: "+state.RefCode)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func handleAuthCommand(bot *tgbotapi.BotAPI, chatID int64, state *models.BotState, redis *models.RedisClient, mtp *mtproto.Session) {
	state.CurrentStep = "waiting_phone"

	msg := tgbotapi.NewMessage(chatID, "Введите номер телефона в формате +71234567890")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func handleRegularMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state *models.BotState, redis *models.RedisClient) {
	switch state.CurrentStep {
	case "waiting_phone":
		handlePhoneInput(bot, msg, state)
	case "waiting_code":
		handleCodeInput(bot, msg, state)
	default:
		handleUnknownCommand(bot, msg.Chat.ID)
	}
}

func handlePhoneInput(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state *models.BotState) {
	phone := msg.Text
	state.CurrentStep = "waiting_code"
	state.RefCode = "ref_" + phone // Просто пример, в реальном коде используйте нормальную генерацию

	reply := tgbotapi.NewMessage(msg.Chat.ID, "Номер принят. Введите код подтверждения")
	if _, err := bot.Send(reply); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func handleCodeInput(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state *models.BotState) {
	code := msg.Text
	state.CurrentStep = "authenticated"

	reply := tgbotapi.NewMessage(msg.Chat.ID, "Вы успешно авторизованы! Ваш код: "+code)
	if _, err := bot.Send(reply); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте /start или /auth")
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
