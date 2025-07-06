package main

import (
	"admin-bot/models"
	"admin-bot/repositories"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

type StateData struct {
	Buttons [][]string `json:"buttons"`
}

func (s *StateData) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StateData) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &s)
}

func main() {
	// Инициализация бота
	var err error
	bot, err = tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// Инициализация базы данных
	db, err = initDB()
	if err != nil {
		log.Panicf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Проверка таблиц (только после инициализации db)
	if err := checkDatabase(); err != nil {
		log.Panicf("Database check failed: %v", err)
	}
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

func checkDatabase() error {
	requiredTables := []string{"bot_templates", "users"}
	for _, table := range requiredTables {
		var exists bool
		err := db.QueryRow(`
            SELECT EXISTS (
                SELECT FROM information_schema.tables 
                WHERE table_name = $1
            )`, table).Scan(&exists)

		if err != nil || !exists {
			return fmt.Errorf("table %s check failed", table)
		}
	}
	return nil
}

func initDB() (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)

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
			ShowOwnerPanel(bot, message.Chat.ID)
			return
		}
	}

	if message.IsCommand() {
		switch message.Command() {
		case "start":
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: db,
			}), &gorm.Config{})
			if err != nil {
				log.Printf("Failed to create gorm DB: %v", err)
				sendMessage(message.Chat.ID, "❌ Ошибка инициализации")
				return
			}

			update := tgbotapi.Update{
				Message: message,
			}

			HandleStart(bot, gormDB, update)
			return
		}
	}

	sendMessage(message.Chat.ID, "Используйте кнопки меню")
}

func AddTemplateHandler(bot *tgbotapi.BotAPI, db *sql.DB, userID int64, chatID int64) {
	setUserState(userID, &UserState{
		CurrentAction: "awaiting_template_name",
		TempData:      make(map[string]interface{}),
	})

	msg := tgbotapi.NewMessage(chatID, "📝 Создание нового шаблона\n\nВведите название шаблона:")
	msg.ReplyMarkup = getCancelKeyboard()
	send(msg)
}

func CompleteTemplateCreation(bot *tgbotapi.BotAPI, db *sql.DB, userID int64, chatID int64, templateData map[string]interface{}) error {
	name, ok := templateData["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		sendMessage(chatID, "❌ Неверное название шаблона")
		return fmt.Errorf("invalid template name")
	}

	content, ok := templateData["content"].(string)
	if !ok || strings.TrimSpace(content) == "" {
		sendMessage(chatID, "❌ Неверное содержание шаблона")
		return fmt.Errorf("invalid template content")
	}

	keyboard, ok := templateData["keyboard"].([][]string)
	if !ok || len(keyboard) == 0 {
		sendMessage(chatID, "❌ Неверный формат клавиатуры")
		return fmt.Errorf("invalid keyboard format")
	}

	keyboardJSON, err := json.Marshal(keyboard)
	if err != nil {
		sendMessage(chatID, "❌ Ошибка обработки клавиатуры")
		return fmt.Errorf("keyboard marshal error: %v", err)
	}

	query := `
        INSERT INTO bot_templates 
        (user_id, name, content, keyboard, is_active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id`

	var templateID int64
	err = db.QueryRow(query,
		userID,
		name,
		content,
		keyboardJSON,
		true,
		time.Now(),
		time.Now(),
	).Scan(&templateID)

	if err != nil {
		log.Printf("Ошибка при сохранении шаблона: %v", err)
		sendMessage(chatID, "❌ Ошибка при сохранении шаблона в БД")
		return fmt.Errorf("database error: %v", err)
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"✅ Шаблон успешно создан!\n\nID: %d\nНазвание: %s\n\nТеперь вы можете использовать его при создании ботов.",
		templateID, name))

	send(msg)
	return nil
}

func ShowTemplateDetails(bot *tgbotapi.BotAPI, chatID int64, template models.BotTemplate) {
	var keyboard [][]string
	if err := json.Unmarshal(template.Keyboard, &keyboard); err != nil {
		log.Printf("Ошибка разбора клавиатуры: %v", err)
		keyboard = [][]string{{"Ошибка отображения"}}
	}

	msgText := fmt.Sprintf(
		"📋 Шаблон: %s\n\nID: %d\nСодержание:\n%s\n\nКлавиатура:",
		template.Name, template.ID, template.Content)

	for _, row := range keyboard {
		msgText += "\n"
		for _, btn := range row {
			msgText += fmt.Sprintf("[%s] ", btn)
		}
	}

	msg := tgbotapi.NewMessage(chatID, msgText)

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("edit_template:%d", template.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Удалить", fmt.Sprintf("delete_template:%d", template.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "list_templates"),
		),
	)

	send(msg)
}

func normalizeJSONInput(input string) (string, error) {
	input = strings.TrimSpace(input)

	re := regexp.MustCompile(`"\s*([^"]+?)\s*"`)
	input = re.ReplaceAllString(input, `"$1"`)

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
func HandleStart(bot *tgbotapi.BotAPI, db *gorm.DB, update tgbotapi.Update) {
	userRepo := repositories.NewUserRepository(db)
	telegramID := update.Message.From.ID
	username := update.Message.From.UserName
	user, err := userRepo.GetOrCreate(telegramID, username, "owner")
	if err != nil {
		log.Printf("Ошибка создания пользователя: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Ошибка инициализации")
		bot.Send(msg)
		return
	}
	isOwner, err := userRepo.IsOwner(user.TelegramID)
	if err != nil {
		log.Printf("Ошибка проверки владельца: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Ошибка проверки доступа")
		bot.Send(msg)
		return
	}

	if !isOwner {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⛔ Доступ только для владельцев")
		bot.Send(msg)
		return
	}

	ShowOwnerPanel(bot, update.Message.Chat.ID)
}

func ShowOwnerPanel(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "👑 Панель владельца")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 Мои боты", "my_bots"),
			tgbotapi.NewInlineKeyboardButtonData("📝 Шаблоны", "templates"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать шаблон", "add_template"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
			tgbotapi.NewInlineKeyboardButtonData("💳 Тарифы", "billing"),
		),
	)

	msg.ReplyMarkup = keyboard
	bot.Send(msg)
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

	case "add_template":
		AddTemplateHandler(bot, db, callback.From.ID, callback.Message.Chat.ID)

	case "list_templates":
		templates := getUserTemplates(callback.From.ID)
		if len(templates) == 0 {
			sendMessage(callback.Message.Chat.ID, "У вас пока нет шаблонов.")
			return
		}
		ShowTemplatesList(bot, callback.Message.Chat.ID, templates)

	case "view_template":
		if len(parts) < 2 {
			sendMessage(callback.Message.Chat.ID, "Ошибка: не указан ID шаблона")
			return
		}

		templateID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			sendMessage(callback.Message.Chat.ID, "Ошибка: неверный ID шаблона")
			return
		}

		template := getTemplateByID(templateID)
		if template == nil {
			sendMessage(callback.Message.Chat.ID, "Шаблон не найден")
			return
		}

		ShowTemplateDetails(bot, callback.Message.Chat.ID, *template)

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

	case "templates":
		templates := getUserTemplates(callback.From.ID)
		if len(templates) == 0 {
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "У вас пока нет шаблонов. Хотите создать новый?")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("➕ Создать шаблон", "add_template"),
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "main_menu"),
				),
			)
			send(msg)
			return
		}
		ShowTemplatesList(bot, callback.Message.Chat.ID, templates)
	case "cancel":
		clearUserState(callback.From.ID)
		sendMessage(callback.Message.Chat.ID, "Действие отменено")
		ShowOwnerPanel(bot, callback.Message.Chat.ID)
	case "main_menu":
		clearUserState(callback.From.ID)
		ShowOwnerPanel(bot, callback.Message.Chat.ID)
	}
}

func ShowTemplatesList(bot *tgbotapi.BotAPI, chatID int64, templates []models.BotTemplate) {
	if len(templates) == 0 {
		sendMessage(chatID, "У вас пока нет шаблонов.")
		return
	}

	msg := tgbotapi.NewMessage(chatID, "📂 Ваши шаблоны:")
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, t := range templates {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s (ID: %d)", t.Name, t.ID),
			fmt.Sprintf("view_template:%d", t.ID),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// Добавляем кнопки управления
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Создать новый", "add_template"),
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "main_menu"),
	))

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	send(msg)
}

func getTemplateByID(templateID int64) *models.BotTemplate {
	row := db.QueryRow(`
        SELECT id, user_id, name, content, keyboard, is_active, created_at, updated_at
        FROM bot_templates WHERE id = $1`, templateID)

	var t models.BotTemplate
	var keyboardJSON []byte

	err := row.Scan(
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
		log.Printf("Ошибка при получении шаблона: %v", err)
		return nil
	}

	t.Keyboard = keyboardJSON
	return &t
}
func saveTemplate(userID int64, data map[string]interface{}) error {
	if err := db.Ping(); err != nil {
		log.Printf("Database ping failed: %v", err)
		return fmt.Errorf("database connection error")
	}

	name, ok := data["name"].(string)
	if !ok {
		return fmt.Errorf("invalid name data")
	}

	content, ok := data["content"].(string)
	if !ok {
		return fmt.Errorf("invalid content data")
	}

	keyboard, ok := data["keyboard"].([][]string)
	if !ok {
		return fmt.Errorf("invalid keyboard data")
	}

	keyboardJSON, err := json.Marshal(keyboard)
	if err != nil {
		log.Printf("Keyboard marshal error: %v", err)
		return fmt.Errorf("keyboard format error")
	}

	query := `
        INSERT INTO bot_templates 
        (user_id, name, content, keyboard, is_active, created_at) 
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id`

	var id int64
	err = db.QueryRow(query,
		userID,
		name,
		content,
		string(keyboardJSON),
		true,
		time.Now(),
	).Scan(&id)

	if err != nil {
		log.Printf("Database error: %v\nQuery: %s\nParams: %d, %s, %s, %s, %v, %v",
			err, query, userID, name, content, string(keyboardJSON), true, time.Now())
		return fmt.Errorf("database save error")
	}

	return nil
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

func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	send(msg)
}

func send(msg tgbotapi.Chattable) {
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
