package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type NodeService interface {
	RegisterNode(ctx context.Context, name, region string) (*model.Node, string, error)
	Heartbeat(ctx context.Context, nodeID uuid.UUID) error
	GetNode(ctx context.Context, nodeID uuid.UUID) (*model.Node, error)
}

type nodeService struct {
	repo repository.NodeRepository
}

func NewNodeService(repo repository.NodeRepository) NodeService {
	return &nodeService{repo: repo}
}

func (s *nodeService) RegisterNode(ctx context.Context, name, region string) (*model.Node, string, error) {
	// Generate a random token
	token := uuid.New().String()
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	node := &model.Node{
		Name:          name,
		Region:        region,
		Status:        model.NodeStatusOnline,
		AuthTokenHash: tokenHash,
	}

	if err := s.repo.Create(ctx, node); err != nil {
		return nil, "", err
	}

	return node, token, nil
}

func (s *nodeService) Heartbeat(ctx context.Context, nodeID uuid.UUID) error {
	return s.repo.UpdateHeartbeat(ctx, nodeID)
}

func (s *nodeService) GetNode(ctx context.Context, nodeID uuid.UUID) (*model.Node, error) {
	return s.repo.GetByID(ctx, nodeID)
}
