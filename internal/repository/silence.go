package repository

import (
	"alertbot/internal/models"
	"time"

	"gorm.io/gorm"
)

type silenceRepository struct {
	db *gorm.DB
}

func NewSilenceRepository(db *gorm.DB) SilenceRepository {
	return &silenceRepository{db: db}
}

func (r *silenceRepository) Create(silence *models.Silence) error {
	return r.db.Create(silence).Error
}

func (r *silenceRepository) GetByID(id uint) (*models.Silence, error) {
	var silence models.Silence
	err := r.db.First(&silence, id).Error
	if err != nil {
		return nil, err
	}
	return &silence, nil
}

func (r *silenceRepository) List() ([]models.Silence, error) {
	var silences []models.Silence
	err := r.db.Order("created_at DESC").Find(&silences).Error
	return silences, err
}

func (r *silenceRepository) Delete(id uint) error {
	return r.db.Delete(&models.Silence{}, id).Error
}

func (r *silenceRepository) GetActiveSilences() ([]models.Silence, error) {
	var silences []models.Silence
	now := time.Now()
	err := r.db.Where("starts_at <= ? AND ends_at > ?", now, now).Find(&silences).Error
	return silences, err
}