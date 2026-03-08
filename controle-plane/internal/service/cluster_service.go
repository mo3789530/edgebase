package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type RegisterClusterInput struct {
	Name        string
	Region      string
	Environment string
	APIEndpoint string
	Labels      map[string]string
}

type ClusterService interface {
	RegisterCluster(ctx context.Context, in RegisterClusterInput) (*model.Cluster, string, error)
	Heartbeat(ctx context.Context, clusterID uuid.UUID) error
	GetCluster(ctx context.Context, clusterID uuid.UUID) (*model.Cluster, error)
	ListClusters(ctx context.Context) ([]model.Cluster, error)
}

type clusterService struct {
	repo repository.ClusterRepository
}

func NewClusterService(repo repository.ClusterRepository) ClusterService {
	return &clusterService{repo: repo}
}

func (s *clusterService) RegisterCluster(ctx context.Context, in RegisterClusterInput) (*model.Cluster, string, error) {
	token := uuid.New().String()
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	cluster := &model.Cluster{
		Name:          in.Name,
		Region:        in.Region,
		Environment:   in.Environment,
		APIEndpoint:   in.APIEndpoint,
		Labels:        in.Labels,
		Status:        model.ClusterStatusOnline,
		AuthTokenHash: tokenHash,
	}

	if err := s.repo.Create(ctx, cluster); err != nil {
		return nil, "", err
	}

	return cluster, token, nil
}

func (s *clusterService) Heartbeat(ctx context.Context, clusterID uuid.UUID) error {
	return s.repo.UpdateHeartbeat(ctx, clusterID)
}

func (s *clusterService) GetCluster(ctx context.Context, clusterID uuid.UUID) (*model.Cluster, error) {
	return s.repo.GetByID(ctx, clusterID)
}

func (s *clusterService) ListClusters(ctx context.Context) ([]model.Cluster, error) {
	return s.repo.List(ctx)
}
