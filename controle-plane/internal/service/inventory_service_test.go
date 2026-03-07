package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) CreateSnapshot(ctx context.Context, snapshot *model.ClusterInventorySnapshot) error {
	args := m.Called(ctx, snapshot)
	return args.Error(0)
}

func TestInventoryService_SaveSnapshot(t *testing.T) {
	repo := new(MockInventoryRepository)
	svc := NewInventoryService(repo)
	clusterID := uuid.New()

	nodesRaw, _ := json.Marshal([]map[string]interface{}{{"name": "n1"}})
	deployRaw, _ := json.Marshal([]map[string]interface{}{{"name": "d1"}, {"name": "d2"}})
	svcRaw, _ := json.Marshal([]map[string]interface{}{})
	podsRaw, _ := json.Marshal([]map[string]interface{}{{"name": "p1"}})

	repo.On("CreateSnapshot", mock.Anything, mock.MatchedBy(func(s *model.ClusterInventorySnapshot) bool {
		return s.ClusterID == clusterID && s.NodesCount == 1 && s.DeploymentsCount == 2 && s.ServicesCount == 0 && s.PodsCount == 1
	})).Return(nil).Once()

	err := svc.SaveSnapshot(context.Background(), clusterID, ClusterInventoryInput{
		ClusterID:         clusterID,
		ObservedAt:        time.Now().UTC(),
		KubernetesVersion: "v1.31.2+k3s1",
		Nodes:             nodesRaw,
		Deployments:       deployRaw,
		Services:          svcRaw,
		Pods:              podsRaw,
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
