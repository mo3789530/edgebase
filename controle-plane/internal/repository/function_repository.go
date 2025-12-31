package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FunctionRepository interface {
	Create(ctx context.Context, function *model.Function) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Function, error)
	GetByNameAndVersion(ctx context.Context, name, version string) (*model.Function, error)
	List(ctx context.Context) ([]model.Function, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, function *model.Function) error
}

type functionRepository struct {
	db *gorm.DB
}

func NewFunctionRepository(db *gorm.DB) FunctionRepository {
	return &functionRepository{db: db}
}

func (r *functionRepository) Create(ctx context.Context, function *model.Function) error {
	return r.db.WithContext(ctx).Create(function).Error
}

func (r *functionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Function, error) {
	var function model.Function
	if err := r.db.WithContext(ctx).First(&function, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &function, nil
}

func (r *functionRepository) GetByNameAndVersion(ctx context.Context, name, version string) (*model.Function, error) {
	var function model.Function
	if err := r.db.WithContext(ctx).Where("name = ? AND version = ?", name, version).First(&function).Error; err != nil {
		return nil, err
	}
	return &function, nil
}

func (r *functionRepository) List(ctx context.Context) ([]model.Function, error) {
	var functions []model.Function
	if err := r.db.WithContext(ctx).Find(&functions).Error; err != nil {
		return nil, err
	}
	return functions, nil
}

func (r *functionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Function{}, "id = ?", id).Error
}

func (r *functionRepository) Update(ctx context.Context, function *model.Function) error {
	return r.db.WithContext(ctx).Save(function).Error
}
