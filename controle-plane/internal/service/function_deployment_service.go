package service

import (
	"context"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type FunctionDeploymentService interface {
	CreateTargets(ctx context.Context, functionDefinitionID uuid.UUID, input CreateDeploymentTargetsInput) ([]model.FunctionDeploymentTarget, error)
}

type CreateDeploymentTargetsInput struct {
	ClusterIDs        []uuid.UUID
	Namespace         string
	DesiredRevisionID uuid.UUID
	Replicas          int32
	RolloutStrategy   string
}

type functionDeploymentService struct {
	definitionRepo repository.FunctionDefinitionRepository
	revisionRepo   repository.FunctionRevisionRepository
	targetRepo     repository.FunctionDeploymentTargetRepository
}

func NewFunctionDeploymentService(
	definitionRepo repository.FunctionDefinitionRepository,
	revisionRepo repository.FunctionRevisionRepository,
	targetRepo repository.FunctionDeploymentTargetRepository,
) FunctionDeploymentService {
	return &functionDeploymentService{
		definitionRepo: definitionRepo,
		revisionRepo:   revisionRepo,
		targetRepo:     targetRepo,
	}
}

func (s *functionDeploymentService) CreateTargets(ctx context.Context, functionDefinitionID uuid.UUID, input CreateDeploymentTargetsInput) ([]model.FunctionDeploymentTarget, error) {
	if _, err := s.definitionRepo.GetByID(ctx, functionDefinitionID); err != nil {
		return nil, err
	}
	if _, err := s.revisionRepo.GetByID(ctx, input.DesiredRevisionID); err != nil {
		return nil, err
	}

	if input.Namespace == "" {
		input.Namespace = "edge-functions"
	}
	if input.Replicas <= 0 {
		input.Replicas = 1
	}
	if input.RolloutStrategy == "" {
		input.RolloutStrategy = "rolling"
	}

	targets := make([]model.FunctionDeploymentTarget, 0, len(input.ClusterIDs))
	for _, clusterID := range input.ClusterIDs {
		target := model.FunctionDeploymentTarget{
			ID:                   uuid.New(),
			FunctionDefinitionID: functionDefinitionID,
			ClusterID:            clusterID,
			Namespace:            input.Namespace,
			DesiredRevisionID:    input.DesiredRevisionID,
			Replicas:             input.Replicas,
			RolloutStrategy:      input.RolloutStrategy,
			Enabled:              true,
			Status:               "pending",
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		if err := s.targetRepo.Create(ctx, &target); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, nil
}
