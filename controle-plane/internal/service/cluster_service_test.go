package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClusterRepository struct {
	mock.Mock
}

func (m *MockClusterRepository) Create(ctx context.Context, cluster *model.Cluster) error {
	args := m.Called(ctx, cluster)
	return args.Error(0)
}
func (m *MockClusterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Cluster, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Cluster), args.Error(1)
}
func (m *MockClusterRepository) List(ctx context.Context) ([]model.Cluster, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Cluster), args.Error(1)
}
func (m *MockClusterRepository) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockClusterRepository) UpdateInventoryAt(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockClusterRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ClusterStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func TestRegisterCluster(t *testing.T) {
	mockRepo := new(MockClusterRepository)
	svc := NewClusterService(mockRepo)
	ctx := context.Background()

	mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Cluster")).Return(nil).Once()

	cluster, token, err := svc.RegisterCluster(ctx, RegisterClusterInput{
		Name:        "tokyo-edge-1",
		Region:      "ap-northeast-1",
		Environment: "prod",
		APIEndpoint: "https://10.0.0.10:6443",
		Labels:      map[string]string{"tier": "edge"},
	})

	assert.NoError(t, err)
	assert.NotNil(t, cluster)
	assert.NotEmpty(t, token)
	assert.Equal(t, "tokyo-edge-1", cluster.Name)
	assert.Equal(t, model.ClusterStatusOnline, cluster.Status)
	assert.NotEmpty(t, cluster.AuthTokenHash)
	assert.NotEqual(t, token, cluster.AuthTokenHash)
	mockRepo.AssertExpectations(t)
}
