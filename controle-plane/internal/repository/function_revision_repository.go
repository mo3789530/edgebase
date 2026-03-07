package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FunctionRevisionRepository interface {
	Create(ctx context.Context, revision *model.FunctionRevision) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.FunctionRevision, error)
	GetByDefinitionAndVersion(ctx context.Context, functionDefinitionID uuid.UUID, version string) (*model.FunctionRevision, error)
	ListByDefinitionID(ctx context.Context, functionDefinitionID uuid.UUID) ([]model.FunctionRevision, error)
}

type functionRevisionRepository struct {
	db *gorm.DB
}

func NewFunctionRevisionRepository(db *gorm.DB) FunctionRevisionRepository {
	return &functionRevisionRepository{db: db}
}

func (r *functionRevisionRepository) Create(ctx context.Context, revision *model.FunctionRevision) error {
	return r.db.WithContext(ctx).Create(revision).Error
}

func (r *functionRevisionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.FunctionRevision, error) {
	var revision model.FunctionRevision
	if err := r.db.WithContext(ctx).First(&revision, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &revision, nil
}

func (r *functionRevisionRepository) GetByDefinitionAndVersion(ctx context.Context, functionDefinitionID uuid.UUID, version string) (*model.FunctionRevision, error) {
	var revision model.FunctionRevision
	if err := r.db.WithContext(ctx).
		Where("function_definition_id = ? AND version = ?", functionDefinitionID, version).
		First(&revision).Error; err != nil {
		return nil, err
	}
	return &revision, nil
}

func (r *functionRevisionRepository) ListByDefinitionID(ctx context.Context, functionDefinitionID uuid.UUID) ([]model.FunctionRevision, error) {
	var revisions []model.FunctionRevision
	if err := r.db.WithContext(ctx).
		Where("function_definition_id = ?", functionDefinitionID).
		Order("created_at desc").
		Find(&revisions).Error; err != nil {
		return nil, err
	}
	return revisions, nil
}
