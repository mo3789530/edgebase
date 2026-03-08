package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClusterInventoryRepository struct {
	mock.Mock
}

func (m *MockClusterInventoryRepository) ReplaceNodes(ctx context.Context, clusterID uuid.UUID, nodes []model.ClusterNode) error {
	args := m.Called(ctx, clusterID, nodes)
	return args.Error(0)
}

func (m *MockClusterInventoryRepository) ListNodes(ctx context.Context, clusterID uuid.UUID) ([]model.ClusterNode, error) {
	args := m.Called(ctx, clusterID)
	return args.Get(0).([]model.ClusterNode), args.Error(1)
}

func TestUpdateInventory(t *testing.T) {
	clusterID := uuid.New()
	ctx := context.Background()

	mockClusterRepo := new(MockClusterRepository)
	mockInvRepo := new(MockClusterInventoryRepository)
	svc := NewClusterInventoryService(mockClusterRepo, mockInvRepo)

	mockInvRepo.On("ReplaceNodes", ctx, clusterID, mock.MatchedBy(func(nodes []model.ClusterNode) bool {
		return len(nodes) == 1 && nodes[0].NodeName == "node-a" && nodes[0].ClusterID == clusterID
	})).Return(nil).Once()
	mockClusterRepo.On("UpdateInventoryAt", ctx, clusterID).Return(nil).Once()

	err := svc.UpdateInventory(ctx, clusterID, ClusterInventoryInput{
		Nodes: []ClusterInventoryNodeInput{
			{
				NodeName:   "node-a",
				Role:       "worker",
				Status:     "ready",
				InternalIP: "10.0.0.11",
			},
		},
	})
	assert.NoError(t, err)

	mockInvRepo.AssertExpectations(t)
	mockClusterRepo.AssertExpectations(t)
}
