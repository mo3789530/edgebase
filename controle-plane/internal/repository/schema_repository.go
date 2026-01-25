package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"gorm.io/gorm"
)

type SchemaRepository interface {
	Create(ctx context.Context, schema *model.SchemaMigration) error
	GetLatestVersion(ctx context.Context) (int, error)
	GetByVersion(ctx context.Context, version int) (*model.SchemaMigration, error)
	ListSince(ctx context.Context, version int) ([]model.SchemaMigration, error)
	UpdateNodeStatus(ctx context.Context, status *model.NodeSchemaStatus) error
}

type schemaRepository struct {
	db *gorm.DB
}

func NewSchemaRepository(db *gorm.DB) SchemaRepository {
	return &schemaRepository{db: db}
}

func (r *schemaRepository) Create(ctx context.Context, schema *model.SchemaMigration) error {
	return r.db.WithContext(ctx).Create(schema).Error
}

func (r *schemaRepository) GetLatestVersion(ctx context.Context) (int, error) {
	var schema model.SchemaMigration
	if err := r.db.WithContext(ctx).Order("version desc").First(&schema).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return schema.Version, nil
}

func (r *schemaRepository) GetByVersion(ctx context.Context, version int) (*model.SchemaMigration, error) {
	var schema model.SchemaMigration
	if err := r.db.WithContext(ctx).Where("version = ?", version).First(&schema).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &schema, nil
}

func (r *schemaRepository) ListSince(ctx context.Context, version int) ([]model.SchemaMigration, error) {
	var schemas []model.SchemaMigration
	if err := r.db.WithContext(ctx).Where("version > ?", version).Order("version asc").Find(&schemas).Error; err != nil {
		return nil, err
	}
	return schemas, nil
}

func (r *schemaRepository) UpdateNodeStatus(ctx context.Context, status *model.NodeSchemaStatus) error {
	// Upsert based on NodeID? But NodeSchemaStatus has ID.
	// We want one status per node? Or history? The requirements say "Current version distribution".
	// Usually we keep the latest status.
	// Let's assume one entry per node for now, or log history? "NodeSchemaStatus" sounds like current state.
	// But ID is uuid.
	// Let's check if there is an existing record for NodeID.
	
	var existing model.NodeSchemaStatus
	err := r.db.WithContext(ctx).Where("node_id = ?", status.NodeID).First(&existing).Error
	if err == nil {
		status.ID = existing.ID
		return r.db.WithContext(ctx).Save(status).Error
	} else if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(status).Error
	}
	return err
}
