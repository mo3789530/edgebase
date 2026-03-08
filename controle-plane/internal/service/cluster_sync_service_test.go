package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClusterSyncRepository struct {
	mock.Mock
}

func (m *MockClusterSyncRepository) CreateRecord(ctx context.Context, record *model.ClusterSyncRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func TestGetClusterSyncPlan(t *testing.T) {
	ctx := context.Background()
	clusterID := uuid.New()
	mockSyncRepo := new(MockClusterSyncRepository)
	mockFuncRepo := new(MockFunctionRepository)
	mockSchemaRepo := new(MockSchemaRepository)
	svc := NewClusterSyncService(mockSyncRepo, mockFuncRepo, mockSchemaRepo)

	mockSchemaRepo.On("GetLatestVersion", ctx).Return(3, nil).Once()
	mockFuncRepo.On("List", ctx).Return([]model.Function{
		{Name: "fn-a", Version: "v1.0.0", Hash: "h1"},
	}, nil).Once()

	plan, err := svc.GetSyncPlan(ctx, clusterID, ClusterState{})
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, 3, plan.DesiredState.SchemaVersion)
	assert.Len(t, plan.DesiredState.Functions, 1)
	assert.Equal(t, "fn-a", plan.DesiredState.Functions[0].Name)
}

func TestAckClusterSync(t *testing.T) {
	ctx := context.Background()
	clusterID := uuid.New()
	syncID := uuid.New()
	mockSyncRepo := new(MockClusterSyncRepository)
	mockFuncRepo := new(MockFunctionRepository)
	mockSchemaRepo := new(MockSchemaRepository)
	svc := NewClusterSyncService(mockSyncRepo, mockFuncRepo, mockSchemaRepo)

	mockSyncRepo.On("CreateRecord", ctx, mock.MatchedBy(func(record *model.ClusterSyncRecord) bool {
		return record.ID == syncID && record.ClusterID == clusterID && record.Status == "success"
	})).Return(nil).Once()

	err := svc.AcknowledgeSync(ctx, clusterID, syncID, ClusterSyncResult{
		Success:        true,
		ChangesSummary: "applied: fn-a",
	})
	assert.NoError(t, err)
	mockSyncRepo.AssertExpectations(t)
}
