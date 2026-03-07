package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InventoryRepository interface {
	CreateSnapshot(ctx context.Context, snapshot *model.ClusterInventorySnapshot) error
	GetLatestSnapshot(ctx context.Context, clusterID uuid.UUID) (*model.ClusterInventorySnapshot, error)
}

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) CreateSnapshot(ctx context.Context, snapshot *model.ClusterInventorySnapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}

func (r *inventoryRepository) GetLatestSnapshot(ctx context.Context, clusterID uuid.UUID) (*model.ClusterInventorySnapshot, error) {
	var snapshot model.ClusterInventorySnapshot
	if err := r.db.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		Order("observed_at desc").
		First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}
