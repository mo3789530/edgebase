package service

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type ClusterInventoryNodeInput struct {
	NodeName         string
	Role             string
	InternalIP       string
	Status           string
	KubeletVersion   string
	OSImage          string
	ContainerRuntime string
}

type ClusterInventoryInput struct {
	Nodes []ClusterInventoryNodeInput
}

type ClusterInventoryService interface {
	UpdateInventory(ctx context.Context, clusterID uuid.UUID, in ClusterInventoryInput) error
}

type clusterInventoryService struct {
	clusterRepo   repository.ClusterRepository
	inventoryRepo repository.ClusterInventoryRepository
}

func NewClusterInventoryService(clusterRepo repository.ClusterRepository, inventoryRepo repository.ClusterInventoryRepository) ClusterInventoryService {
	return &clusterInventoryService{
		clusterRepo:   clusterRepo,
		inventoryRepo: inventoryRepo,
	}
}

func (s *clusterInventoryService) UpdateInventory(ctx context.Context, clusterID uuid.UUID, in ClusterInventoryInput) error {
	nodes := make([]model.ClusterNode, 0, len(in.Nodes))
	for _, n := range in.Nodes {
		nodes = append(nodes, model.ClusterNode{
			ClusterID:        clusterID,
			NodeName:         n.NodeName,
			Role:             n.Role,
			InternalIP:       n.InternalIP,
			Status:           n.Status,
			KubeletVersion:   n.KubeletVersion,
			OSImage:          n.OSImage,
			ContainerRuntime: n.ContainerRuntime,
		})
	}

	if err := s.inventoryRepo.ReplaceNodes(ctx, clusterID, nodes); err != nil {
		return err
	}
	return s.clusterRepo.UpdateInventoryAt(ctx, clusterID)
}
