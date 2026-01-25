package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VMRepository interface {
	Create(ctx context.Context, vm *model.VM) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.VM, error)
	ListByNodeID(ctx context.Context, nodeID uuid.UUID) ([]model.VM, error)
	Update(ctx context.Context, vm *model.VM) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type vmRepository struct {
	db *gorm.DB
}

func NewVMRepository(db *gorm.DB) VMRepository {
	return &vmRepository{db: db}
}

func (r *vmRepository) Create(ctx context.Context, vm *model.VM) error {
	return r.db.WithContext(ctx).Create(vm).Error
}

func (r *vmRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.VM, error) {
	var vm model.VM
	if err := r.db.WithContext(ctx).First(&vm, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &vm, nil
}

func (r *vmRepository) ListByNodeID(ctx context.Context, nodeID uuid.UUID) ([]model.VM, error) {
	var vms []model.VM
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Find(&vms).Error; err != nil {
		return nil, err
	}
	return vms, nil
}

func (r *vmRepository) Update(ctx context.Context, vm *model.VM) error {
	return r.db.WithContext(ctx).Save(vm).Error
}

func (r *vmRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.VM{}, "id = ?", id).Error
}
