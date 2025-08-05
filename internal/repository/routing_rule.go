package repository

import (
	"alertbot/internal/models"

	"gorm.io/gorm"
)

type routingRuleRepository struct {
	db *gorm.DB
}

func NewRoutingRuleRepository(db *gorm.DB) RoutingRuleRepository {
	return &routingRuleRepository{db: db}
}

func (r *routingRuleRepository) Create(rule *models.RoutingRule) error {
	return r.db.Create(rule).Error
}

func (r *routingRuleRepository) GetByID(id uint) (*models.RoutingRule, error) {
	var rule models.RoutingRule
	err := r.db.First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *routingRuleRepository) List() ([]models.RoutingRule, error) {
	var rules []models.RoutingRule
	err := r.db.Order("priority DESC, created_at DESC").Find(&rules).Error
	return rules, err
}

func (r *routingRuleRepository) Update(rule *models.RoutingRule) error {
	return r.db.Save(rule).Error
}

func (r *routingRuleRepository) Delete(id uint) error {
	return r.db.Delete(&models.RoutingRule{}, id).Error
}

func (r *routingRuleRepository) GetActiveRulesByPriority() ([]models.RoutingRule, error) {
	var rules []models.RoutingRule
	err := r.db.Where("enabled = ?", true).Order("priority DESC").Find(&rules).Error
	return rules, err
}