package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockFunctionDefinitionRepository struct {
	mock.Mock
}

func (m *MockFunctionDefinitionRepository) Create(ctx context.Context, definition *model.FunctionDefinition) error {
	args := m.Called(ctx, definition)
	return args.Error(0)
}

func (m *MockFunctionDefinitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.FunctionDefinition, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FunctionDefinition), args.Error(1)
}

func (m *MockFunctionDefinitionRepository) GetByName(ctx context.Context, name string) (*model.FunctionDefinition, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FunctionDefinition), args.Error(1)
}

func (m *MockFunctionDefinitionRepository) List(ctx context.Context) ([]model.FunctionDefinition, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FunctionDefinition), args.Error(1)
}

type MockFunctionRevisionRepository struct {
	mock.Mock
}

func (m *MockFunctionRevisionRepository) Create(ctx context.Context, revision *model.FunctionRevision) error {
	args := m.Called(ctx, revision)
	return args.Error(0)
}

func (m *MockFunctionRevisionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.FunctionRevision, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FunctionRevision), args.Error(1)
}

func (m *MockFunctionRevisionRepository) GetByDefinitionAndVersion(ctx context.Context, functionDefinitionID uuid.UUID, version string) (*model.FunctionRevision, error) {
	args := m.Called(ctx, functionDefinitionID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FunctionRevision), args.Error(1)
}

func (m *MockFunctionRevisionRepository) ListByDefinitionID(ctx context.Context, functionDefinitionID uuid.UUID) ([]model.FunctionRevision, error) {
	args := m.Called(ctx, functionDefinitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FunctionRevision), args.Error(1)
}

func TestFunctionCatalogService_CreateDefinition(t *testing.T) {
	ctx := context.Background()
	definitionRepo := new(MockFunctionDefinitionRepository)
	revisionRepo := new(MockFunctionRevisionRepository)
	svc := NewFunctionCatalogService(definitionRepo, revisionRepo)

	definitionRepo.On("GetByName", ctx, "telemetry-normalizer").Return(nil, gorm.ErrRecordNotFound).Once()
	definitionRepo.On("Create", ctx, mock.MatchedBy(func(def *model.FunctionDefinition) bool {
		return def.Name == "telemetry-normalizer" && def.RuntimeKind == "container" && def.DefaultTimeoutSeconds == 3
	})).Return(nil).Once()

	definition, err := svc.CreateDefinition(ctx, "telemetry-normalizer", "normalize telemetry", "container", 3, 128, 250)

	assert.NoError(t, err)
	assert.Equal(t, "telemetry-normalizer", definition.Name)
	definitionRepo.AssertExpectations(t)
}

func TestFunctionCatalogService_CreateRevision(t *testing.T) {
	ctx := context.Background()
	definitionRepo := new(MockFunctionDefinitionRepository)
	revisionRepo := new(MockFunctionRevisionRepository)
	svc := NewFunctionCatalogService(definitionRepo, revisionRepo)

	functionID := uuid.New()
	definitionRepo.On("GetByID", ctx, functionID).Return(&model.FunctionDefinition{ID: functionID, Name: "telemetry-normalizer"}, nil).Once()
	revisionRepo.On("GetByDefinitionAndVersion", ctx, functionID, "v1").Return(nil, gorm.ErrRecordNotFound).Once()
	revisionRepo.On("Create", ctx, mock.MatchedBy(func(rev *model.FunctionRevision) bool {
		return rev.FunctionDefinitionID == functionID && rev.Version == "v1" && rev.Port == 8080 && rev.HealthcheckPath == "/health"
	})).Return(nil).Once()

	revision, err := svc.CreateRevision(ctx, functionID, CreateFunctionRevisionInput{
		Version:     "v1",
		Image:       "registry.local/telemetry-normalizer:v1",
		ImageDigest: "sha256:abcd",
	})

	assert.NoError(t, err)
	assert.Equal(t, "v1", revision.Version)
	revisionRepo.AssertExpectations(t)
}

func TestFunctionCatalogService_ListDefinitions(t *testing.T) {
	ctx := context.Background()
	definitionRepo := new(MockFunctionDefinitionRepository)
	revisionRepo := new(MockFunctionRevisionRepository)
	svc := NewFunctionCatalogService(definitionRepo, revisionRepo)

	definitions := []model.FunctionDefinition{{ID: uuid.New(), Name: "telemetry-normalizer"}}
	definitionRepo.On("List", ctx).Return(definitions, nil).Once()

	got, err := svc.ListDefinitions(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "telemetry-normalizer", got[0].Name)
	definitionRepo.AssertExpectations(t)
}
