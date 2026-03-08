package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"gorm.io/gorm"
)

type ClusterSyncRepository interface {
	CreateRecord(ctx context.Context, record *model.ClusterSyncRecord) error
}

type clusterSyncRepository struct {
	db *gorm.DB
}

func NewClusterSyncRepository(db *gorm.DB) ClusterSyncRepository {
	return &clusterSyncRepository{db: db}
}

func (r *clusterSyncRepository) CreateRecord(ctx context.Context, record *model.ClusterSyncRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}
