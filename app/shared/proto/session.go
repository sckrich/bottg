package proto

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"admin-bot/models"
	"shared/database"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
)

type SessionStorage struct {
	UserID int
}

func (s SessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	var user models.BotUser
	if err := database.DB.WithContext(ctx).
		Where("id = ?", s.UserID).
		First(&user).Error; err != nil {
		return nil, err
	}

	if len(user.SessionData) == 0 {
		return nil, session.ErrNotFound
	}

	return user.SessionData, nil
}

func (s SessionStorage) StoreSession(ctx context.Context, data []byte) error {
	return database.DB.WithContext(ctx).
		Model(&models.BotUser{}).
		Where("id = ?", s.UserID).
		Update("session_data", data).Error
}

func CreateClient(userID int) (*telegram.Client, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	storage := SessionStorage{UserID: userID}

	return telegram.NewClient(
		telegram.TestAppID,
		telegram.TestAppHash,
		telegram.Options{
			SessionStorage: storage,
		},
	), nil
}

func SaveAuthSession(ctx context.Context, userID int, data *session.Data) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return database.DB.WithContext(ctx).
		Model(&models.BotUser{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"session_data": bytes,
			"last_active":  time.Now(),
		}).Error
}
