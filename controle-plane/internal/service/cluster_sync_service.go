package service

import (
	"context"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type ClusterFunctionState struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type ClusterState struct {
	SchemaVersion int                    `json:"schema_version"`
	Functions     []ClusterFunctionState `json:"functions"`
}

type ClusterDesiredState struct {
	SchemaVersion int             `json:"schema_version"`
	Functions     []FunctionState `json:"functions"`
}

type ClusterSyncPlan struct {
	SyncID       uuid.UUID           `json:"sync_id"`
	DesiredState ClusterDesiredState `json:"desired_state"`
}

type ClusterSyncResult struct {
	Success        bool   `json:"success"`
	ErrorMessage   string `json:"error_message"`
	ChangesSummary string `json:"changes_summary"`
}

type ClusterSyncService interface {
	GetSyncPlan(ctx context.Context, clusterID uuid.UUID, currentState ClusterState) (*ClusterSyncPlan, error)
	AcknowledgeSync(ctx context.Context, clusterID uuid.UUID, syncID uuid.UUID, result ClusterSyncResult) error
}

type clusterSyncService struct {
	syncRepo   repository.ClusterSyncRepository
	funcRepo   repository.FunctionRepository
	schemaRepo repository.SchemaRepository
}

func NewClusterSyncService(
	syncRepo repository.ClusterSyncRepository,
	funcRepo repository.FunctionRepository,
	schemaRepo repository.SchemaRepository,
) ClusterSyncService {
	return &clusterSyncService{
		syncRepo:   syncRepo,
		funcRepo:   funcRepo,
		schemaRepo: schemaRepo,
	}
}

func (s *clusterSyncService) GetSyncPlan(ctx context.Context, clusterID uuid.UUID, currentState ClusterState) (*ClusterSyncPlan, error) {
	latestVersion, err := s.schemaRepo.GetLatestVersion(ctx)
	if err != nil {
		return nil, err
	}

	allFunctions, err := s.funcRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	latestFunctionsMap := make(map[string]*model.Function)
	for i := range allFunctions {
		fn := &allFunctions[i]
		if existing, ok := latestFunctionsMap[fn.Name]; !ok || fn.CreatedAt.After(existing.CreatedAt) {
			latestFunctionsMap[fn.Name] = fn
		}
	}

	functions := make([]FunctionState, 0, len(latestFunctionsMap))
	for _, fn := range latestFunctionsMap {
		functions = append(functions, FunctionState{
			Name:    fn.Name,
			Version: fn.Version,
			Hash:    fn.Hash,
		})
	}

	return &ClusterSyncPlan{
		SyncID: uuid.New(),
		DesiredState: ClusterDesiredState{
			SchemaVersion: latestVersion,
			Functions:     functions,
		},
	}, nil
}

func (s *clusterSyncService) AcknowledgeSync(ctx context.Context, clusterID uuid.UUID, syncID uuid.UUID, result ClusterSyncResult) error {
	status := "success"
	if !result.Success {
		status = "failed"
	}
	now := time.Now()
	record := &model.ClusterSyncRecord{
		ID:             syncID,
		ClusterID:      clusterID,
		SyncType:       "incremental",
		Status:         status,
		StartedAt:      now,
		CompletedAt:    &now,
		ErrorMessage:   result.ErrorMessage,
		ChangesSummary: result.ChangesSummary,
	}
	return s.syncRepo.CreateRecord(ctx, record)
}
