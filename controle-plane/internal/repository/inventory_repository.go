package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"gorm.io/gorm"
)

type InventoryRepository interface {
	CreateSnapshot(ctx context.Context, snapshot *model.ClusterInventorySnapshot) error
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
