package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"admin-bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

type UserState struct {
	CurrentAction string
	TempData      map[string]interface{}
}

var (
	bot        *tgbotapi.BotAPI
	db         *sql.DB
	userStates = make(map[int64]*UserState)
	stateMutex sync.RWMutex
)

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	db, err = initDB()
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{tgbotapi.UpdateTypeMessage, tgbotapi.UpdateTypeCallbackQuery}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			handleCallback(update.CallbackQuery)
		}
	}
}

func initDB() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	return sql.Open("postgres", connStr)
}
func handleMessage(message *tgbotapi.Message) {
	state := getUserState(message.From.ID)

	if state != nil {
		switch state.CurrentAction {
		case "awaiting_template_name":
			state.TempData["name"] = message.Text
			state.CurrentAction = "awaiting_template_content"
			msg := tgbotapi.NewMessage(message.Chat.ID, "Введите содержание шаблона:")
			msg.ReplyMarkup = getCancelKeyboard()
			send(msg)
			return

		case "awaiting_template_content":
			state.TempData["content"] = message.Text
			state.CurrentAction = "awaiting_template_keyboard"
			msg := tgbotapi.NewMessage(message.Chat.ID, "Введите клавиатуру в JSON формате (пример: [[\"Да\"], [\"Нет\"]]):")
			msg.ReplyMarkup = getCancelKeyboard()
			send(msg)
			return

		case "awaiting_template_keyboard":
			cleanedInput := strings.TrimSpace(message.Text)
			cleanedInput = strings.ReplaceAll(cleanedInput, "“", `"`)
			cleanedInput = strings.ReplaceAll(cleanedInput, "”", `"`)

			// Пробуем распарсить JSON
			var keyboard models.StateData
			if err := json.Unmarshal([]byte(cleanedInput), &keyboard); err != nil {
				log.Printf("JSON parse error: %v\nInput was: %s", err, cleanedInput)

				// Создаем информативное сообщение об ошибке
				errorMsg := tgbotapi.NewMessage(
					message.Chat.ID,
					"❌ Неверный формат JSON. Пожалуйста, введите клавиатуру в формате:\n"+
						"```\n[[\"Кнопка 1\"], [\"Кнопка 2\", \"Кнопка 3\"]]\n```\n"+
						"Обязательно используйте двойные кавычки (\") и правильные скобки.",
				)
				errorMsg.ParseMode = "Markdown"
				errorMsg.ReplyMarkup = getCancelKeyboard()
				send(errorMsg)
				return
			}
			state.TempData["keyboard"] = keyboard
			if err := saveTemplate(message.From.ID, state.TempData); err != nil {
				log.Printf("Error saving template: %v", err)
				sendMessage(message.Chat.ID, "❌ Ошибка при сохранении шаблона. Попробуйте позже.")
				return
			}

			clearUserState(message.From.ID)
			sendMessage(message.Chat.ID, "✅ Шаблон успешно создан!")
			return
		}
	}

	if message.IsCommand() {
		switch message.Command() {
		case "start":
			sendMainMenu(message.Chat.ID)
			return
		}
	}

	sendMessage(message.Chat.ID, "Используйте кнопки меню")
}
func handleCallback(callback *tgbotapi.CallbackQuery) {
	callbackCfg := tgbotapi.NewCallback(callback.ID, "")
	bot.Request(callbackCfg)

	parts := strings.Split(callback.Data, ":")
	action := parts[0]

	switch action {

	case "add_bot":
		if len(parts) < 2 {
			sendMessage(callback.Message.Chat.ID, "Ошибка выбора шаблона")
			return
		}
		templateID, _ := strconv.ParseInt(parts[1], 10, 64)
		setUserState(callback.From.ID, &UserState{
			CurrentAction: "awaiting_bot_token",
			TempData:      map[string]interface{}{"template_id": templateID},
		})
		sendMessage(callback.Message.Chat.ID, "Введите токен бота:")

	case "view_templates":
		templates := getUserTemplates(callback.From.ID)
		if len(templates) == 0 {
			sendMessage(callback.Message.Chat.ID, "У вас нет шаблонов")
			return
		}
		msg := "📁 Ваши шаблоны:\n"
		for _, t := range templates {
			msg += fmt.Sprintf("\n🔹 %s (ID: %d)", t.Name, t.ID)
		}
		sendMessage(callback.Message.Chat.ID, msg)
	case "add_template":
		setUserState(callback.From.ID, &UserState{
			CurrentAction: "awaiting_template_name",
			TempData:      make(map[string]interface{}),
		})
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Введите название шаблона:")
		msg.ReplyMarkup = getCancelKeyboard()
		send(msg)
	case "cancel":
		clearUserState(callback.From.ID)
		sendMessage(callback.Message.Chat.ID, "Действие отменено")
		sendMainMenu(callback.Message.Chat.ID)
	}
}

func sendMainMenu(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "📱 Главное меню")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить бота", "add_bot_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Создать шаблон", "add_template"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👀 Мои шаблоны", "view_templates"),
		),
	)
	msg.ReplyMarkup = keyboard
	send(msg)
}

func saveTemplate(userID int64, data map[string]interface{}) error {
	keyboardJSON, _ := json.Marshal(data["keyboard"])
	_, err := db.Exec(`
		INSERT INTO bot_templates 
		(user_id, name, content, keyboard, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, data["name"], data["content"], keyboardJSON, true, time.Now())
	return err
}
func saveBot(userID int64, token string, templateID int64) error {
	_, err := db.Exec(`
		INSERT INTO bots 
		(user_id, token, template_id, created_at)
		VALUES ($1, $2, $3, $4)`,
		userID, token, templateID, time.Now())
	return err
}
func getCancelKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)
}
func getUserTemplates(userID int64) []models.BotTemplate {
	rows, err := db.Query(`
		SELECT id, name, content, keyboard 
		FROM bot_templates 
		WHERE user_id = $1`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var templates []models.BotTemplate
	for rows.Next() {
		var t models.BotTemplate
		var keyboardJSON []byte
		rows.Scan(&t.ID, &t.Name, &t.Content, &keyboardJSON)
		json.Unmarshal(keyboardJSON, &t.Keyboard)
		templates = append(templates, t)
	}
	return templates
}
func parseBotID(text string) (int64, error) {
	var botID int64
	_, err := fmt.Sscanf(text, "%d", &botID)
	return botID, err
}

func sendAdminPanel(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🛠 Админ-панель управления шаблонами")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Создать шаблон", "add_template"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_template"),
		),
	)
	send(msg)
}

func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	send(msg)
}

func send(msg tgbotapi.Chattable) {
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// Управление состояниями
func setUserState(userID int64, state *UserState) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	userStates[userID] = state
}

func getUserState(userID int64) *UserState {
	stateMutex.RLock()
	defer stateMutex.RUnlock()
	return userStates[userID]
}

func clearUserState(userID int64) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	delete(userStates, userID)
}
