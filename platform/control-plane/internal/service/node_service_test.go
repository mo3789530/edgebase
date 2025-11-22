package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterNode(t *testing.T) {
	mockRepo := new(MockNodeRepository)
	svc := NewNodeService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Node")).Return(nil).Once()

		node, token, err := svc.RegisterNode(ctx, "test-node", "us-east-1")

		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.NotEmpty(t, token)
		assert.Equal(t, "test-node", node.Name)
		assert.Equal(t, "us-east-1", node.Region)
		assert.Equal(t, model.NodeStatusOnline, node.Status)
		assert.NotEmpty(t, node.AuthTokenHash)
		assert.NotEqual(t, token, node.AuthTokenHash) // Hash should be different from raw token

		mockRepo.AssertExpectations(t)
	})
}

func TestHeartbeat(t *testing.T) {
	mockRepo := new(MockNodeRepository)
	svc := NewNodeService(mockRepo)
	ctx := context.Background()
	id := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("UpdateHeartbeat", ctx, id).Return(nil).Once()

		err := svc.Heartbeat(ctx, id)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetNode(t *testing.T) {
	mockRepo := new(MockNodeRepository)
	svc := NewNodeService(mockRepo)
	ctx := context.Background()
	id := uuid.New()

	t.Run("Found", func(t *testing.T) {
		expectedNode := &model.Node{ID: id, Name: "test"}
		mockRepo.On("GetByID", ctx, id).Return(expectedNode, nil).Once()

		node, err := svc.GetNode(ctx, id)

		assert.NoError(t, err)
		assert.Equal(t, expectedNode, node)
		mockRepo.AssertExpectations(t)
	})
}
