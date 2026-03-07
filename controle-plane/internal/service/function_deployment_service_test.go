package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFunctionDeploymentService_CreateTargets(t *testing.T) {
	ctx := context.Background()
	definitionRepo := new(MockFunctionDefinitionRepository)
	revisionRepo := new(MockFunctionRevisionRepository)
	targetRepo := new(MockFunctionDeploymentTargetRepository)
	svc := NewFunctionDeploymentService(definitionRepo, revisionRepo, targetRepo)

	functionID := uuid.New()
	revisionID := uuid.New()
	clusterID := uuid.New()

	definitionRepo.On("GetByID", ctx, functionID).Return(&model.FunctionDefinition{ID: functionID}, nil).Once()
	revisionRepo.On("GetByID", ctx, revisionID).Return(&model.FunctionRevision{ID: revisionID}, nil).Once()
	targetRepo.On("Create", ctx, mock.MatchedBy(func(target *model.FunctionDeploymentTarget) bool {
		return target.FunctionDefinitionID == functionID &&
			target.ClusterID == clusterID &&
			target.DesiredRevisionID == revisionID &&
			target.Namespace == "edge-functions" &&
			target.Replicas == 1
	})).Return(nil).Once()

	targets, err := svc.CreateTargets(ctx, functionID, CreateDeploymentTargetsInput{
		ClusterIDs:        []uuid.UUID{clusterID},
		DesiredRevisionID: revisionID,
	})

	assert.NoError(t, err)
	assert.Len(t, targets, 1)
	assert.Equal(t, clusterID, targets[0].ClusterID)
	targetRepo.AssertExpectations(t)
}
