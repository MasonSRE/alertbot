package repository

import (
	"alertbot/internal/models"

	"gorm.io/gorm"
)

type alertHistoryRepository struct {
	db *gorm.DB
}

func NewAlertHistoryRepository(db *gorm.DB) AlertHistoryRepository {
	return &alertHistoryRepository{db: db}
}

func (r *alertHistoryRepository) Create(history *models.AlertHistory) error {
	return r.db.Create(history).Error
}

func (r *alertHistoryRepository) GetByAlertFingerprint(fingerprint string) ([]models.AlertHistory, error) {
	var histories []models.AlertHistory
	err := r.db.Where("alert_fingerprint = ?", fingerprint).Order("created_at DESC").Find(&histories).Error
	return histories, err
}