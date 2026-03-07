package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type ClusterInventoryInput struct {
	ClusterID         uuid.UUID       `json:"cluster_id"`
	ObservedAt        time.Time       `json:"observed_at"`
	KubernetesVersion string          `json:"kubernetes_version"`
	Nodes             json.RawMessage `json:"nodes"`
	Deployments       json.RawMessage `json:"deployments"`
	Services          json.RawMessage `json:"services"`
	Pods              json.RawMessage `json:"pods"`
}

type InventoryService interface {
	SaveSnapshot(ctx context.Context, clusterID uuid.UUID, in ClusterInventoryInput) error
}

type inventoryService struct {
	repo repository.InventoryRepository
}

func NewInventoryService(repo repository.InventoryRepository) InventoryService {
	return &inventoryService{repo: repo}
}

func (s *inventoryService) SaveSnapshot(ctx context.Context, clusterID uuid.UUID, in ClusterInventoryInput) error {
	if in.ClusterID != uuid.Nil && in.ClusterID != clusterID {
		return fmt.Errorf("cluster_id mismatch")
	}

	if in.ObservedAt.IsZero() {
		in.ObservedAt = time.Now().UTC()
	}

	payloadObj := map[string]json.RawMessage{
		"nodes":       normalizedRaw(in.Nodes),
		"deployments": normalizedRaw(in.Deployments),
		"services":    normalizedRaw(in.Services),
		"pods":        normalizedRaw(in.Pods),
	}
	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return fmt.Errorf("marshal inventory payload: %w", err)
	}

	snapshot := &model.ClusterInventorySnapshot{
		ClusterID:         clusterID,
		ObservedAt:        in.ObservedAt,
		KubernetesVersion: in.KubernetesVersion,
		NodesCount:        countJSONArray(in.Nodes),
		DeploymentsCount:  countJSONArray(in.Deployments),
		ServicesCount:     countJSONArray(in.Services),
		PodsCount:         countJSONArray(in.Pods),
		Payload:           string(payloadBytes),
	}

	if err := s.repo.CreateSnapshot(ctx, snapshot); err != nil {
		return fmt.Errorf("create inventory snapshot: %w", err)
	}

	return nil
}

func normalizedRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("[]")
	}
	return raw
}

func countJSONArray(raw json.RawMessage) int {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return 0
	}
	return len(items)
}
