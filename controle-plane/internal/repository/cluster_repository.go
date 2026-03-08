package repository

import (
	"context"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClusterRepository interface {
	Create(ctx context.Context, cluster *model.Cluster) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Cluster, error)
	List(ctx context.Context) ([]model.Cluster, error)
	UpdateHeartbeat(ctx context.Context, id uuid.UUID) error
	UpdateInventoryAt(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.ClusterStatus) error
}

type clusterRepository struct {
	db *gorm.DB
}

func NewClusterRepository(db *gorm.DB) ClusterRepository {
	return &clusterRepository{db: db}
}

func (r *clusterRepository) Create(ctx context.Context, cluster *model.Cluster) error {
	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *clusterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Cluster, error) {
	var cluster model.Cluster
	if err := r.db.WithContext(ctx).First(&cluster, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *clusterRepository) List(ctx context.Context) ([]model.Cluster, error) {
	var clusters []model.Cluster
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

func (r *clusterRepository) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Cluster{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_heartbeat_at": now,
		"status":            model.ClusterStatusOnline,
	}).Error
}

func (r *clusterRepository) UpdateInventoryAt(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Cluster{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_inventory_at": now,
		"status":            model.ClusterStatusOnline,
	}).Error
}

func (r *clusterRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ClusterStatus) error {
	return r.db.WithContext(ctx).Model(&model.Cluster{}).Where("id = ?", id).Update("status", status).Error
}
