package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
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

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("connection string error: %v", err)
	}

	// Важные настройки пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверка подключения с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping failed: %v", err)
	}

	return db, nil
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
			normalizedInput, err := normalizeJSONInput(message.Text)
			if err != nil {
				sendJSONError(message.Chat.ID)
				return
			}

			var keyboard [][]string
			if err := json.Unmarshal([]byte(normalizedInput), &keyboard); err != nil {
				// Показываем пользователю где именно ошибка
				errorPos := strings.Index(err.Error(), "offset ")
				if errorPos > 0 {
					posStr := err.Error()[errorPos+7:]
					if pos, e := strconv.Atoi(posStr); e == nil {
						excerpt := normalizedInput[max(0, pos-10):min(len(normalizedInput), pos+10)]
						sendMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка в позиции ~%d: ...%s...", pos, excerpt))
					}
				}
				sendJSONError(message.Chat.ID)
				return
			}

			// Дополнительная проверка структуры
			if len(keyboard) == 0 {
				sendMessage(message.Chat.ID, "❌ Клавиатура не может быть пустой")
				return
			}

			for _, row := range keyboard {
				if len(row) == 0 {
					sendMessage(message.Chat.ID, "❌ Строка клавиатуры не может быть пустой")
					return
				}
				for _, button := range row {
					if strings.TrimSpace(button) == "" {
						sendMessage(message.Chat.ID, "❌ Текст кнопки не может быть пустым")
						return
					}
				}
			}

			state.TempData["keyboard"] = keyboard

			if err := saveTemplate(message.From.ID, state.TempData); err != nil {
				log.Printf("Full save error: %v\nTemplate data: %+v", err, state.TempData)

				detailedMsg := "❌ Ошибка сохранения:\n"

				switch {
				case strings.Contains(err.Error(), "invalid template name"):
					detailedMsg += "Некорректное имя шаблона"
				case strings.Contains(err.Error(), "invalid template content"):
					detailedMsg += "Некорректное содержание шаблона"
				case strings.Contains(err.Error(), "invalid keyboard"):
					detailedMsg += "Некорректный формат клавиатуры"
				case strings.Contains(err.Error(), "database"):
					detailedMsg += "Проблема с базой данных"
				default:
					detailedMsg += "Техническая ошибка"
				}

				detailedMsg += "\n\nПопробуйте ещё раз или обратитесь в поддержку"

				msg := tgbotapi.NewMessage(message.Chat.ID, detailedMsg)
				msg.ReplyMarkup = getCancelKeyboard()
				send(msg)
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
func normalizeJSONInput(input string) (string, error) {
	input = strings.TrimSpace(input)

	// Удаляем все пробелы между кавычками и текстом
	re := regexp.MustCompile(`"\s*([^"]+?)\s*"`)
	input = re.ReplaceAllString(input, `"$1"`)

	// Проверяем баланс скобок
	if strings.Count(input, "[") != strings.Count(input, "]") {
		return "", fmt.Errorf("unbalanced brackets")
	}

	return input, nil
}
func sendJSONError(chatID int64) {
	example := `Пример правильного формата JSON для клавиатуры:
    
    [
        ["Да", "Нет"],
        ["Может быть"]
    ]

Или для одной кнопки в строке:
    
    [
        ["Да"],
        ["Нет"]
    ]`

	msg := tgbotapi.NewMessage(chatID, "❌ Неверный формат JSON. Пожалуйста, используйте один из следующих форматов:\n"+example)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = getCancelKeyboard()
	send(msg)
}
func sendJSONFormatError(chatID int64) {
	example := `Пример правильного формата:
[
    ["Да", "Нет"],
    ["Может быть"]
]`

	msg := tgbotapi.NewMessage(chatID, "❌ Неверный формат JSON. Пожалуйста, используйте следующий формат:\n"+example)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = getCancelKeyboard()
	send(msg)
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
	log.Printf("Attempting to save template for user %d\nData: %+v", userID, data)

	// 1. Извлечение данных с проверкой типов
	name, _ := data["name"].(string)
	content, _ := data["content"].(string)
	keyboard, ok := data["keyboard"].([][]string)
	if !ok {
		return fmt.Errorf("keyboard type assertion failed")
	}

	// 2. Преобразование клавиатуры в JSON
	keyboardJSON, err := json.Marshal(keyboard)
	if err != nil {
		log.Printf("Keyboard marshal error: %v\nKeyboard: %v", err, keyboard)
		return fmt.Errorf("keyboard marshal error")
	}

	// 3. Проверка существования пользователя
	var userExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&userExists)
	if err != nil || !userExists {
		return fmt.Errorf("user verification failed")
	}

	// 4. Выполнение запроса с транзакцией
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Transaction begin error: %v", err)
		return fmt.Errorf("transaction error")
	}
	defer tx.Rollback()

	// 5. Логируем полный SQL-запрос
	query := `INSERT INTO bot_templates 
             (user_id, name, content, keyboard, is_active, created_at)
             VALUES ($1, $2, $3, $4, $5, $6)`

	log.Printf("Executing query: %s\nParams: %d, %s, %s, %s, %v, %v",
		query, userID, name, content, string(keyboardJSON), true, time.Now())

	_, err = tx.Exec(query, userID, name, content, keyboardJSON, true, time.Now())
	if err != nil {
		log.Printf("Database error: %v", err)
		return fmt.Errorf("database execution error")
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Commit error: %v", err)
		return fmt.Errorf("transaction commit error")
	}

	return nil
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
        SELECT id, user_id, name, content, keyboard, is_active, created_at, updated_at
        FROM bot_templates 
        WHERE user_id = $1`, userID)
	if err != nil {
		log.Printf("Database query error: %v", err)
		return nil
	}
	defer rows.Close()

	var templates []models.BotTemplate
	for rows.Next() {
		var t models.BotTemplate
		var keyboardJSON []byte

		err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.Name,
			&t.Content,
			&keyboardJSON,
			&t.IsActive,
			&t.CreatedAt,
			&t.UpdatedAt,
		)

		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}

		t.Keyboard = keyboardJSON
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
