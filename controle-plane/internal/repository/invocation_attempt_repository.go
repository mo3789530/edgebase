package repository

import (
	"context"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvocationAttemptRepository interface {
	Create(ctx context.Context, attempt *model.InvocationAttempt) error
	ListByInvocationID(ctx context.Context, invocationID uuid.UUID) ([]model.InvocationAttempt, error)
	UpdateCompletion(ctx context.Context, id uuid.UUID, completedAt time.Time, status string, statusCode *int, errorMessage string) error
}

type invocationAttemptRepository struct {
	db *gorm.DB
}

func NewInvocationAttemptRepository(db *gorm.DB) InvocationAttemptRepository {
	return &invocationAttemptRepository{db: db}
}

func (r *invocationAttemptRepository) Create(ctx context.Context, attempt *model.InvocationAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *invocationAttemptRepository) ListByInvocationID(ctx context.Context, invocationID uuid.UUID) ([]model.InvocationAttempt, error) {
	var attempts []model.InvocationAttempt
	if err := r.db.WithContext(ctx).Where("invocation_id = ?", invocationID).Order("attempt_no asc").Find(&attempts).Error; err != nil {
		return nil, err
	}
	return attempts, nil
}

func (r *invocationAttemptRepository) UpdateCompletion(ctx context.Context, id uuid.UUID, completedAt time.Time, status string, statusCode *int, errorMessage string) error {
	updates := map[string]interface{}{
		"completed_at":  completedAt,
		"status":        status,
		"error_message": errorMessage,
		"updated_at":    time.Now().UTC(),
	}
	if statusCode != nil {
		updates["status_code"] = *statusCode
	}
	return r.db.WithContext(ctx).Model(&model.InvocationAttempt{}).Where("id = ?", id).Updates(updates).Error
}
