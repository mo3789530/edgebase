package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FunctionDeploymentTargetRepository interface {
	Create(ctx context.Context, target *model.FunctionDeploymentTarget) error
	ListByClusterID(ctx context.Context, clusterID uuid.UUID) ([]model.FunctionDeploymentTarget, error)
	UpdateStatusByClusterID(ctx context.Context, clusterID uuid.UUID, status string) error
}

type functionDeploymentTargetRepository struct {
	db *gorm.DB
}

func NewFunctionDeploymentTargetRepository(db *gorm.DB) FunctionDeploymentTargetRepository {
	return &functionDeploymentTargetRepository{db: db}
}

func (r *functionDeploymentTargetRepository) Create(ctx context.Context, target *model.FunctionDeploymentTarget) error {
	return r.db.WithContext(ctx).Create(target).Error
}

func (r *functionDeploymentTargetRepository) ListByClusterID(ctx context.Context, clusterID uuid.UUID) ([]model.FunctionDeploymentTarget, error) {
	var targets []model.FunctionDeploymentTarget
	if err := r.db.WithContext(ctx).
		Where("cluster_id = ? AND enabled = ?", clusterID, true).
		Order("created_at asc").
		Find(&targets).Error; err != nil {
		return nil, err
	}
	return targets, nil
}

func (r *functionDeploymentTargetRepository) UpdateStatusByClusterID(ctx context.Context, clusterID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.FunctionDeploymentTarget{}).
		Where("cluster_id = ?", clusterID).
		Update("status", status).Error
}
