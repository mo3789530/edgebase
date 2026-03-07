package service

import (
	"context"
	"fmt"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FunctionCatalogService interface {
	CreateDefinition(ctx context.Context, name, description, runtimeKind string, defaultTimeoutSeconds, defaultMemoryMB, defaultCPUMillis int32) (*model.FunctionDefinition, error)
	GetDefinition(ctx context.Context, id uuid.UUID) (*model.FunctionDefinition, error)
	ListDefinitions(ctx context.Context) ([]model.FunctionDefinition, error)
	CreateRevision(ctx context.Context, functionDefinitionID uuid.UUID, input CreateFunctionRevisionInput) (*model.FunctionRevision, error)
}

type CreateFunctionRevisionInput struct {
	Version         string
	Image           string
	ImageDigest     string
	Command         string
	Args            string
	Env             string
	Port            int32
	HealthcheckPath string
}

type functionCatalogService struct {
	definitionRepo repository.FunctionDefinitionRepository
	revisionRepo   repository.FunctionRevisionRepository
}

func NewFunctionCatalogService(
	definitionRepo repository.FunctionDefinitionRepository,
	revisionRepo repository.FunctionRevisionRepository,
) FunctionCatalogService {
	return &functionCatalogService{
		definitionRepo: definitionRepo,
		revisionRepo:   revisionRepo,
	}
}

func (s *functionCatalogService) CreateDefinition(ctx context.Context, name, description, runtimeKind string, defaultTimeoutSeconds, defaultMemoryMB, defaultCPUMillis int32) (*model.FunctionDefinition, error) {
	existing, err := s.definitionRepo.GetByName(ctx, name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("function definition %s already exists", name)
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if runtimeKind == "" {
		runtimeKind = "container"
	}
	if defaultTimeoutSeconds <= 0 {
		defaultTimeoutSeconds = 3
	}
	if defaultMemoryMB <= 0 {
		defaultMemoryMB = 128
	}
	if defaultCPUMillis <= 0 {
		defaultCPUMillis = 250
	}

	definition := &model.FunctionDefinition{
		ID:                    uuid.New(),
		Name:                  name,
		Description:           description,
		RuntimeKind:           runtimeKind,
		DefaultTimeoutSeconds: defaultTimeoutSeconds,
		DefaultMemoryMB:       defaultMemoryMB,
		DefaultCPUMillis:      defaultCPUMillis,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.definitionRepo.Create(ctx, definition); err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *functionCatalogService) GetDefinition(ctx context.Context, id uuid.UUID) (*model.FunctionDefinition, error) {
	return s.definitionRepo.GetByID(ctx, id)
}

func (s *functionCatalogService) ListDefinitions(ctx context.Context) ([]model.FunctionDefinition, error) {
	return s.definitionRepo.List(ctx)
}

func (s *functionCatalogService) CreateRevision(ctx context.Context, functionDefinitionID uuid.UUID, input CreateFunctionRevisionInput) (*model.FunctionRevision, error) {
	if _, err := s.definitionRepo.GetByID(ctx, functionDefinitionID); err != nil {
		return nil, err
	}

	existing, err := s.revisionRepo.GetByDefinitionAndVersion(ctx, functionDefinitionID, input.Version)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("function revision %s already exists", input.Version)
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if input.Port == 0 {
		input.Port = 8080
	}
	if input.HealthcheckPath == "" {
		input.HealthcheckPath = "/health"
	}

	revision := &model.FunctionRevision{
		ID:                   uuid.New(),
		FunctionDefinitionID: functionDefinitionID,
		Version:              input.Version,
		Image:                input.Image,
		ImageDigest:          input.ImageDigest,
		Command:              input.Command,
		Args:                 input.Args,
		Env:                  input.Env,
		Port:                 input.Port,
		HealthcheckPath:      input.HealthcheckPath,
		CreatedAt:            time.Now(),
	}

	if err := s.revisionRepo.Create(ctx, revision); err != nil {
		return nil, err
	}

	return revision, nil
}
