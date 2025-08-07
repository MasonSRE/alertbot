package repository

import (
	"context"
	"fmt"

	"alertbot/internal/models"

	"gorm.io/gorm"
)

type AlertGroupRepository interface {
	// Alert Groups
	ListAlertGroups(ctx context.Context, filters *models.AlertFilters) ([]*models.AlertGroup, error)
	GetAlertGroup(ctx context.Context, id uint) (*models.AlertGroup, error)
	GetAlertGroupByKey(ctx context.Context, groupKey string) (*models.AlertGroup, error)
	CreateAlertGroup(ctx context.Context, group *models.AlertGroup) error
	UpdateAlertGroup(ctx context.Context, group *models.AlertGroup) error
	DeleteAlertGroup(ctx context.Context, id uint) error

	// Alert Group Rules
	ListAlertGroupRules(ctx context.Context) ([]*models.AlertGroupRule, error)
	GetAlertGroupRule(ctx context.Context, id uint) (*models.AlertGroupRule, error)
	CreateAlertGroupRule(ctx context.Context, rule *models.AlertGroupRule) error
	UpdateAlertGroupRule(ctx context.Context, rule *models.AlertGroupRule) error
	DeleteAlertGroupRule(ctx context.Context, id uint) error

	// Helper methods
	GetActiveAlertGroupRules(ctx context.Context) ([]*models.AlertGroupRule, error)
}

type alertGroupRepository struct {
	db *gorm.DB
}

func NewAlertGroupRepository(db *gorm.DB) AlertGroupRepository {
	return &alertGroupRepository{db: db}
}

// Alert Groups Implementation

func (r *alertGroupRepository) ListAlertGroups(ctx context.Context, filters *models.AlertFilters) ([]*models.AlertGroup, error) {
	var groups []*models.AlertGroup
	
	query := r.db.WithContext(ctx).Model(&models.AlertGroup{})
	
	// Apply filters
	if filters != nil {
		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}
		if filters.Severity != "" {
			query = query.Where("severity = ?", filters.Severity)
		}
		
		// Pagination
		if filters.Page > 0 && filters.Size > 0 {
			offset := (filters.Page - 1) * filters.Size
			query = query.Offset(offset).Limit(filters.Size)
		}
		
		// Sorting
		if filters.Sort != "" {
			order := "ASC"
			if filters.Order == "desc" {
				order = "DESC"
			}
			query = query.Order(fmt.Sprintf("%s %s", filters.Sort, order))
		} else {
			query = query.Order("updated_at DESC")
		}
	}
	
	err := query.Find(&groups).Error
	return groups, err
}

func (r *alertGroupRepository) GetAlertGroup(ctx context.Context, id uint) (*models.AlertGroup, error) {
	var group models.AlertGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *alertGroupRepository) GetAlertGroupByKey(ctx context.Context, groupKey string) (*models.AlertGroup, error) {
	var group models.AlertGroup
	err := r.db.WithContext(ctx).Where("group_key = ?", groupKey).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *alertGroupRepository) CreateAlertGroup(ctx context.Context, group *models.AlertGroup) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *alertGroupRepository) UpdateAlertGroup(ctx context.Context, group *models.AlertGroup) error {
	return r.db.WithContext(ctx).Save(group).Error
}

func (r *alertGroupRepository) DeleteAlertGroup(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AlertGroup{}, id).Error
}

// Alert Group Rules Implementation

func (r *alertGroupRepository) ListAlertGroupRules(ctx context.Context) ([]*models.AlertGroupRule, error) {
	var rules []*models.AlertGroupRule
	err := r.db.WithContext(ctx).Order("priority DESC, created_at ASC").Find(&rules).Error
	return rules, err
}

func (r *alertGroupRepository) GetAlertGroupRule(ctx context.Context, id uint) (*models.AlertGroupRule, error) {
	var rule models.AlertGroupRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *alertGroupRepository) CreateAlertGroupRule(ctx context.Context, rule *models.AlertGroupRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *alertGroupRepository) UpdateAlertGroupRule(ctx context.Context, rule *models.AlertGroupRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *alertGroupRepository) DeleteAlertGroupRule(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AlertGroupRule{}, id).Error
}

func (r *alertGroupRepository) GetActiveAlertGroupRules(ctx context.Context) ([]*models.AlertGroupRule, error) {
	var rules []*models.AlertGroupRule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("priority DESC, created_at ASC").Find(&rules).Error
	return rules, err
}