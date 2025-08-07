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

func (r *alertHistoryRepository) GetByFingerprint(fingerprint string) ([]models.AlertHistory, error) {
	return r.GetByAlertFingerprint(fingerprint)
}

func (r *alertHistoryRepository) List(filters models.AlertHistoryFilters) ([]models.AlertHistory, int64, error) {
	var histories []models.AlertHistory
	var total int64
	
	query := r.db.Model(&models.AlertHistory{})
	
	// Apply filters
	if filters.AlertFingerprint != "" {
		query = query.Where("alert_fingerprint = ?", filters.AlertFingerprint)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	
	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Apply pagination
	if filters.Page > 0 && filters.Size > 0 {
		offset := (filters.Page - 1) * filters.Size
		query = query.Offset(offset).Limit(filters.Size)
	}
	
	// Apply sorting
	sortField := "created_at"
	if filters.Sort != "" {
		sortField = filters.Sort
	}
	sortOrder := "DESC"
	if filters.Order != "" {
		sortOrder = filters.Order
	}
	query = query.Order(sortField + " " + sortOrder)
	
	err := query.Find(&histories).Error
	return histories, total, err
}