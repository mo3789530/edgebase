package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFunctionDeploymentTargetRepository struct {
	mock.Mock
}

func (m *MockFunctionDeploymentTargetRepository) Create(ctx context.Context, target *model.FunctionDeploymentTarget) error {
	args := m.Called(ctx, target)
	return args.Error(0)
}

func (m *MockFunctionDeploymentTargetRepository) ListByClusterID(ctx context.Context, clusterID uuid.UUID) ([]model.FunctionDeploymentTarget, error) {
	args := m.Called(ctx, clusterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FunctionDeploymentTarget), args.Error(1)
}

func (m *MockFunctionDeploymentTargetRepository) UpdateStatusByClusterID(ctx context.Context, clusterID uuid.UUID, status string) error {
	args := m.Called(ctx, clusterID, status)
	return args.Error(0)
}

type MockControllerInventoryRepository struct {
	mock.Mock
}

func (m *MockControllerInventoryRepository) CreateSnapshot(ctx context.Context, snapshot *model.ClusterInventorySnapshot) error {
	args := m.Called(ctx, snapshot)
	return args.Error(0)
}

func (m *MockControllerInventoryRepository) GetLatestSnapshot(ctx context.Context, clusterID uuid.UUID) (*model.ClusterInventorySnapshot, error) {
	args := m.Called(ctx, clusterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClusterInventorySnapshot), args.Error(1)
}

func TestFunctionControllerService_GetClusterSyncPlan(t *testing.T) {
	ctx := context.Background()
	targetRepo := new(MockFunctionDeploymentTargetRepository)
	definitionRepo := new(MockFunctionDefinitionRepository)
	revisionRepo := new(MockFunctionRevisionRepository)
	inventoryRepo := new(MockControllerInventoryRepository)
	svc := NewFunctionControllerService(targetRepo, definitionRepo, revisionRepo, inventoryRepo)

	clusterID := uuid.New()
	functionID := uuid.New()
	revisionID := uuid.New()
	targetRepo.On("ListByClusterID", ctx, clusterID).Return([]model.FunctionDeploymentTarget{{
		ID:                   uuid.New(),
		FunctionDefinitionID: functionID,
		ClusterID:            clusterID,
		Namespace:            "edge-functions",
		DesiredRevisionID:    revisionID,
		Replicas:             1,
	}}, nil).Once()
	inventoryRepo.On("GetLatestSnapshot", ctx, clusterID).Return(&model.ClusterInventorySnapshot{
		ClusterID: clusterID,
		Payload:   `{"deployments":[{"namespace":"edge-functions","name":"fn-old"}],"services":[{"namespace":"edge-functions","name":"fn-old"}]}`,
	}, nil).Once()
	definitionRepo.On("GetByID", ctx, functionID).Return(&model.FunctionDefinition{ID: functionID, Name: "telemetry-normalizer"}, nil).Once()
	revisionRepo.On("GetByID", ctx, revisionID).Return(&model.FunctionRevision{
		ID:              revisionID,
		Version:         "v1",
		Image:           "registry.local/telemetry-normalizer:v1",
		ImageDigest:     "sha256:abcd",
		Port:            8080,
		HealthcheckPath: "/health",
	}, nil).Once()

	plan, err := svc.GetClusterSyncPlan(ctx, clusterID)

	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Len(t, plan.Actions, 4)
	assert.Equal(t, clusterActionApplyDeployment, plan.Actions[0].Type)
	assert.Equal(t, clusterActionApplyService, plan.Actions[1].Type)
	assert.Equal(t, clusterActionDeleteDeployment, plan.Actions[2].Type)
	assert.Equal(t, clusterActionDeleteService, plan.Actions[3].Type)
}

func TestFunctionControllerService_AcknowledgeClusterSync(t *testing.T) {
	ctx := context.Background()
	targetRepo := new(MockFunctionDeploymentTargetRepository)
	definitionRepo := new(MockFunctionDefinitionRepository)
	revisionRepo := new(MockFunctionRevisionRepository)
	inventoryRepo := new(MockControllerInventoryRepository)
	svc := NewFunctionControllerService(targetRepo, definitionRepo, revisionRepo, inventoryRepo)

	clusterID := uuid.New()
	targetRepo.On("UpdateStatusByClusterID", ctx, clusterID, "applied").Return(nil).Once()

	err := svc.AcknowledgeClusterSync(ctx, clusterID, uuid.New(), SyncResult{Success: true})

	assert.NoError(t, err)
	targetRepo.AssertExpectations(t)
}
