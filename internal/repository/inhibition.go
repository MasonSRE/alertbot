package repository

import (
	"context"
	"fmt"

	"alertbot/internal/models"
	"gorm.io/gorm"
)

type inhibitionRepository struct {
	db *gorm.DB
}

func NewInhibitionRepository(db *gorm.DB) InhibitionRepository {
	return &inhibitionRepository{db: db}
}

func (r *inhibitionRepository) Create(rule *models.InhibitionRule) error {
	return r.db.Create(rule).Error
}

func (r *inhibitionRepository) GetByID(id uint) (*models.InhibitionRule, error) {
	var rule models.InhibitionRule
	err := r.db.First(&rule, id).Error
	return &rule, err
}

func (r *inhibitionRepository) List() ([]models.InhibitionRule, error) {
	var rules []models.InhibitionRule
	err := r.db.Order("priority DESC, created_at DESC").Find(&rules).Error
	return rules, err
}

func (r *inhibitionRepository) Update(rule *models.InhibitionRule) error {
	return r.db.Save(rule).Error
}

func (r *inhibitionRepository) Delete(id uint) error {
	return r.db.Delete(&models.InhibitionRule{}, id).Error
}

// Inhibition Rules

func (r *inhibitionRepository) ListInhibitionRules(ctx context.Context) ([]*models.InhibitionRule, error) {
	var rules []*models.InhibitionRule
	err := r.db.WithContext(ctx).Order("priority DESC, created_at ASC").Find(&rules).Error
	return rules, err
}

func (r *inhibitionRepository) GetInhibitionRule(ctx context.Context, id uint) (*models.InhibitionRule, error) {
	var rule models.InhibitionRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *inhibitionRepository) CreateInhibitionRule(ctx context.Context, rule *models.InhibitionRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *inhibitionRepository) UpdateInhibitionRule(ctx context.Context, rule *models.InhibitionRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *inhibitionRepository) DeleteInhibitionRule(ctx context.Context, id uint) error {
	// First delete all related inhibition statuses
	if err := r.db.WithContext(ctx).Where("rule_id = ?", id).Delete(&models.InhibitionStatus{}).Error; err != nil {
		return fmt.Errorf("failed to delete related inhibition statuses: %w", err)
	}
	
	// Then delete the rule
	return r.db.WithContext(ctx).Delete(&models.InhibitionRule{}, id).Error
}

func (r *inhibitionRepository) GetActiveInhibitionRules(ctx context.Context) ([]*models.InhibitionRule, error) {
	var rules []*models.InhibitionRule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("priority DESC, created_at ASC").Find(&rules).Error
	return rules, err
}

// Inhibition Status

func (r *inhibitionRepository) CreateInhibitionStatus(ctx context.Context, status *models.InhibitionStatus) error {
	// Check if this inhibition already exists
	var existing models.InhibitionStatus
	err := r.db.WithContext(ctx).Where(
		"source_fingerprint = ? AND target_fingerprint = ? AND rule_id = ?",
		status.SourceFingerprint, status.TargetFingerprint, status.RuleID,
	).First(&existing).Error
	
	if err == nil {
		// Already exists, update the existing one
		existing.InhibitedAt = status.InhibitedAt
		existing.ExpiresAt = status.ExpiresAt
		return r.db.WithContext(ctx).Save(&existing).Error
	}
	
	if err != gorm.ErrRecordNotFound {
		return err
	}
	
	// Create new inhibition status
	return r.db.WithContext(ctx).Create(status).Error
}

func (r *inhibitionRepository) DeleteInhibitionStatus(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.InhibitionStatus{}, id).Error
}

func (r *inhibitionRepository) GetInhibitionsByTarget(ctx context.Context, targetFingerprint string) ([]*models.InhibitionStatus, error) {
	var inhibitions []*models.InhibitionStatus
	query := r.db.WithContext(ctx).Where("target_fingerprint = ?", targetFingerprint)
	
	// Only get active inhibitions (not expired)
	query = query.Where("expires_at IS NULL OR expires_at > NOW()")
	
	err := query.Find(&inhibitions).Error
	return inhibitions, err
}

func (r *inhibitionRepository) GetInhibitionsBySource(ctx context.Context, sourceFingerprint string) ([]*models.InhibitionStatus, error) {
	var inhibitions []*models.InhibitionStatus
	query := r.db.WithContext(ctx).Where("source_fingerprint = ?", sourceFingerprint)
	
	// Only get active inhibitions (not expired)
	query = query.Where("expires_at IS NULL OR expires_at > NOW()")
	
	err := query.Find(&inhibitions).Error
	return inhibitions, err
}

func (r *inhibitionRepository) CleanupExpiredInhibitions(ctx context.Context) error {
	// Delete expired inhibitions
	return r.db.WithContext(ctx).Where("expires_at IS NOT NULL AND expires_at <= NOW()").Delete(&models.InhibitionStatus{}).Error
}

func (r *inhibitionRepository) GetActiveInhibitions(ctx context.Context) ([]*models.InhibitionStatus, error) {
	var inhibitions []*models.InhibitionStatus
	query := r.db.WithContext(ctx).Where("expires_at IS NULL OR expires_at > NOW()")
	err := query.Find(&inhibitions).Error
	return inhibitions, err
}