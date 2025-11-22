package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"gorm.io/gorm"
)

type SchemaRepository interface {
	Create(ctx context.Context, schema *model.SchemaMigration) error
	GetLatestVersion(ctx context.Context) (int, error)
	ListSince(ctx context.Context, version int) ([]model.SchemaMigration, error)
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

func (r *schemaRepository) ListSince(ctx context.Context, version int) ([]model.SchemaMigration, error) {
	var schemas []model.SchemaMigration
	if err := r.db.WithContext(ctx).Where("version > ?", version).Order("version asc").Find(&schemas).Error; err != nil {
		return nil, err
	}
	return schemas, nil
}
