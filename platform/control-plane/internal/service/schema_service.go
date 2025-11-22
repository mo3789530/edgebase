package service

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
)

type SchemaService interface {
	RegisterSchema(ctx context.Context, version int, upSQL, downSQL, description string) error
	ListSchemas(ctx context.Context) ([]model.SchemaMigration, error)
}

type schemaService struct {
	repo repository.SchemaRepository
}

func NewSchemaService(repo repository.SchemaRepository) SchemaService {
	return &schemaService{repo: repo}
}

func (s *schemaService) RegisterSchema(ctx context.Context, version int, upSQL, downSQL, description string) error {
	schema := &model.SchemaMigration{
		Version:     version,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
		Description: description,
	}
	return s.repo.Create(ctx, schema)
}

func (s *schemaService) ListSchemas(ctx context.Context) ([]model.SchemaMigration, error) {
	// List all schemas. The repository has ListSince, let's use that with version 0
	return s.repo.ListSince(ctx, 0)
}
