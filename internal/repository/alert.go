package repository

import (
	"alertbot/internal/models"
	"fmt"

	"gorm.io/gorm"
)

type alertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) AlertRepository {
	return &alertRepository{db: db}
}

func (r *alertRepository) Create(alert *models.Alert) error {
	return r.db.Create(alert).Error
}

func (r *alertRepository) GetByFingerprint(fingerprint string) (*models.Alert, error) {
	var alert models.Alert
	err := r.db.Where("fingerprint = ?", fingerprint).First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *alertRepository) List(filters models.AlertFilters) ([]models.Alert, int64, error) {
	var alerts []models.Alert
	var total int64

	query := r.db.Model(&models.Alert{})

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Severity != "" {
		query = query.Where("severity = ?", filters.Severity)
	}
	if filters.AlertName != "" {
		query = query.Where("labels->>'alertname' ILIKE ?", "%"+filters.AlertName+"%")
	}
	if filters.Instance != "" {
		query = query.Where("labels->>'instance' ILIKE ?", "%"+filters.Instance+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Size == 0 {
		filters.Size = 20
	}
	if filters.Size > 100 {
		filters.Size = 100
	}

	offset := (filters.Page - 1) * filters.Size

	if filters.Sort == "" {
		filters.Sort = "created_at"
	}
	if filters.Order == "" {
		filters.Order = "desc"
	}

	orderBy := fmt.Sprintf("%s %s", filters.Sort, filters.Order)
	
	err := query.Order(orderBy).Offset(offset).Limit(filters.Size).Find(&alerts).Error
	if err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

func (r *alertRepository) Update(alert *models.Alert) error {
	return r.db.Save(alert).Error
}

func (r *alertRepository) Delete(fingerprint string) error {
	return r.db.Where("fingerprint = ?", fingerprint).Delete(&models.Alert{}).Error
}