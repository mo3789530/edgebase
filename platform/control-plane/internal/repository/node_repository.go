package repository

import (
	"context"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NodeRepository interface {
	Create(ctx context.Context, node *model.Node) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Node, error)
	UpdateHeartbeat(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.NodeStatus) error
	List(ctx context.Context) ([]model.Node, error)
}

type nodeRepository struct {
	db *gorm.DB
}

func NewNodeRepository(db *gorm.DB) NodeRepository {
	return &nodeRepository{db: db}
}

func (r *nodeRepository) Create(ctx context.Context, node *model.Node) error {
	return r.db.WithContext(ctx).Create(node).Error
}

func (r *nodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Node, error) {
	var node model.Node
	if err := r.db.WithContext(ctx).First(&node, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *nodeRepository) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Node{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_heartbeat_at": now,
		"status":            model.NodeStatusOnline,
	}).Error
}

func (r *nodeRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NodeStatus) error {
	return r.db.WithContext(ctx).Model(&model.Node{}).Where("id = ?", id).Update("status", status).Error
}

func (r *nodeRepository) List(ctx context.Context) ([]model.Node, error) {
	var nodes []model.Node
	if err := r.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}
