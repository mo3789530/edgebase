package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FunctionDefinitionRepository interface {
	Create(ctx context.Context, definition *model.FunctionDefinition) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.FunctionDefinition, error)
	GetByName(ctx context.Context, name string) (*model.FunctionDefinition, error)
	List(ctx context.Context) ([]model.FunctionDefinition, error)
}

type functionDefinitionRepository struct {
	db *gorm.DB
}

func NewFunctionDefinitionRepository(db *gorm.DB) FunctionDefinitionRepository {
	return &functionDefinitionRepository{db: db}
}

func (r *functionDefinitionRepository) Create(ctx context.Context, definition *model.FunctionDefinition) error {
	return r.db.WithContext(ctx).Create(definition).Error
}

func (r *functionDefinitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.FunctionDefinition, error) {
	var definition model.FunctionDefinition
	if err := r.db.WithContext(ctx).First(&definition, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &definition, nil
}

func (r *functionDefinitionRepository) GetByName(ctx context.Context, name string) (*model.FunctionDefinition, error) {
	var definition model.FunctionDefinition
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&definition).Error; err != nil {
		return nil, err
	}
	return &definition, nil
}

func (r *functionDefinitionRepository) List(ctx context.Context) ([]model.FunctionDefinition, error) {
	var definitions []model.FunctionDefinition
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&definitions).Error; err != nil {
		return nil, err
	}
	return definitions, nil
}
