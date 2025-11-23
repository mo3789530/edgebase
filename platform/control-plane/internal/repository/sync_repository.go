package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SyncRepository interface {
	CreateRecord(ctx context.Context, record *model.SyncRecord) error
	UpdateRecord(ctx context.Context, record *model.SyncRecord) error
	GetNodeDeployments(ctx context.Context, nodeID uuid.UUID) ([]model.NodeFunctionDeployment, error)
	UpdateNodeDeployments(ctx context.Context, nodeID uuid.UUID, deployments []model.NodeFunctionDeployment) error
}

type syncRepository struct {
	db *gorm.DB
}

func NewSyncRepository(db *gorm.DB) SyncRepository {
	return &syncRepository{db: db}
}

func (r *syncRepository) CreateRecord(ctx context.Context, record *model.SyncRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *syncRepository) UpdateRecord(ctx context.Context, record *model.SyncRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *syncRepository) GetNodeDeployments(ctx context.Context, nodeID uuid.UUID) ([]model.NodeFunctionDeployment, error) {
	var deployments []model.NodeFunctionDeployment
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}

func (r *syncRepository) UpdateNodeDeployments(ctx context.Context, nodeID uuid.UUID, deployments []model.NodeFunctionDeployment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing deployments for this node
		// Note: In a real scenario, we might want to be more selective, but for full sync this works.
		// Or we can use upsert if we want to keep history, but here we just want current state.
		// The design says "NodeFunctionDeployment" tracks current state.
		if err := tx.Where("node_id = ?", nodeID).Delete(&model.NodeFunctionDeployment{}).Error; err != nil {
			return err
		}
		if len(deployments) > 0 {
			if err := tx.Create(&deployments).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
