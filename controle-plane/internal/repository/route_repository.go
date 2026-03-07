package repository

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"gorm.io/gorm"
)

type RouteRepository interface {
	Create(ctx context.Context, route *model.RouteDefinition) error
	List(ctx context.Context) ([]model.RouteDefinition, error)
}

type routeRepository struct {
	db *gorm.DB
}

func NewRouteRepository(db *gorm.DB) RouteRepository {
	return &routeRepository{db: db}
}

func (r *routeRepository) Create(ctx context.Context, route *model.RouteDefinition) error {
	return r.db.WithContext(ctx).Create(route).Error
}

func (r *routeRepository) List(ctx context.Context) ([]model.RouteDefinition, error) {
	var routes []model.RouteDefinition
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("created_at asc").Find(&routes).Error; err != nil {
		return nil, err
	}
	return routes, nil
}

