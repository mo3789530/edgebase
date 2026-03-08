package repository

import (
	"context"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvocationRepository interface {
	Create(ctx context.Context, invocation *model.Invocation) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Invocation, error)
	UpdateCompletion(ctx context.Context, id uuid.UUID, completedAt time.Time, finalStatus string, clientStatusCode *int) error
}

type invocationRepository struct {
	db *gorm.DB
}

func NewInvocationRepository(db *gorm.DB) InvocationRepository {
	return &invocationRepository{db: db}
}

func (r *invocationRepository) Create(ctx context.Context, invocation *model.Invocation) error {
	return r.db.WithContext(ctx).Create(invocation).Error
}

func (r *invocationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Invocation, error) {
	var invocation model.Invocation
	if err := r.db.WithContext(ctx).First(&invocation, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &invocation, nil
}

func (r *invocationRepository) UpdateCompletion(ctx context.Context, id uuid.UUID, completedAt time.Time, finalStatus string, clientStatusCode *int) error {
	updates := map[string]interface{}{
		"completed_at": completedAt,
		"final_status": finalStatus,
		"updated_at":   time.Now().UTC(),
	}
	if clientStatusCode != nil {
		updates["client_status_code"] = *clientStatusCode
	}
	return r.db.WithContext(ctx).Model(&model.Invocation{}).Where("id = ?", id).Updates(updates).Error
}
