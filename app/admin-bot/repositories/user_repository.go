package repositories

import (
	"errors"
	"time"

	"admin-bot/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetOrCreate(telegramID int64, username string, role string) (*models.User, error) {
	var user models.User
	err := r.db.Where("telegram_id = ?", telegramID).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = models.User{
			ID:         uint(telegramID),
			TelegramID: telegramID,
			Username:   username,
			Role:       role,
			IsActive:   true,
			LastActive: time.Now(),
			CreatedAt:  time.Now(),
		}

		if err := r.db.Create(&user).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Обновляем последнюю активность
	user.LastActive = time.Now()
	user.Username = username // Обновляем username если изменился
	r.db.Save(&user)

	return &user, nil
}

func (r *UserRepository) IsOwner(telegramID int64) (bool, error) {
	var user models.User
	err := r.db.Where("telegram_id = ? AND role = ?", telegramID, "owner").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}
