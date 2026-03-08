package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClusterInventoryRepository interface {
	ReplaceNodes(ctx context.Context, clusterID uuid.UUID, nodes []model.ClusterNode) error
	ListNodes(ctx context.Context, clusterID uuid.UUID) ([]model.ClusterNode, error)
}

type clusterInventoryRepository struct {
	db *gorm.DB
}

func NewClusterInventoryRepository(db *gorm.DB) ClusterInventoryRepository {
	return &clusterInventoryRepository{db: db}
}

func (r *clusterInventoryRepository) ReplaceNodes(ctx context.Context, clusterID uuid.UUID, nodes []model.ClusterNode) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cluster_id = ?", clusterID).Delete(&model.ClusterNode{}).Error; err != nil {
			return err
		}
		if len(nodes) == 0 {
			return nil
		}
		return tx.Create(&nodes).Error
	})
}

func (r *clusterInventoryRepository) ListNodes(ctx context.Context, clusterID uuid.UUID) ([]model.ClusterNode, error) {
	var nodes []model.ClusterNode
	if err := r.db.WithContext(ctx).Where("cluster_id = ?", clusterID).Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}
