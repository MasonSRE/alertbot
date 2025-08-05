package repository

import (
	"alertbot/internal/models"

	"gorm.io/gorm"
)

type notificationChannelRepository struct {
	db *gorm.DB
}

func NewNotificationChannelRepository(db *gorm.DB) NotificationChannelRepository {
	return &notificationChannelRepository{db: db}
}

func (r *notificationChannelRepository) Create(channel *models.NotificationChannel) error {
	return r.db.Create(channel).Error
}

func (r *notificationChannelRepository) GetByID(id uint) (*models.NotificationChannel, error) {
	var channel models.NotificationChannel
	err := r.db.First(&channel, id).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (r *notificationChannelRepository) List() ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	err := r.db.Order("created_at DESC").Find(&channels).Error
	return channels, err
}

func (r *notificationChannelRepository) Update(channel *models.NotificationChannel) error {
	return r.db.Save(channel).Error
}

func (r *notificationChannelRepository) Delete(id uint) error {
	return r.db.Delete(&models.NotificationChannel{}, id).Error
}

func (r *notificationChannelRepository) GetActiveChannels() ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	err := r.db.Where("enabled = ?", true).Find(&channels).Error
	return channels, err
}