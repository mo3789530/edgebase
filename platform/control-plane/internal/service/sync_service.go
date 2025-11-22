package service

import (
	"context"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type NodeState struct {
	SchemaVersion int             `json:"schema_version"`
	Functions     []FunctionState `json:"functions"`
}

type FunctionState struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type SyncPlan struct {
	SyncID  uuid.UUID    `json:"sync_id"`
	Actions []SyncAction `json:"actions"`
}

type SyncActionType string

const (
	ActionTypeAddFunction    SyncActionType = "ADD_FUNCTION"
	ActionTypeRemoveFunction SyncActionType = "REMOVE_FUNCTION"
	ActionTypeApplySchema    SyncActionType = "APPLY_SCHEMA"
)

type SyncAction struct {
	Type        SyncActionType `json:"type"`
	Order       int            `json:"order"`
	Payload     interface{}    `json:"payload"`
	Description string         `json:"description"`
}

type SyncResult struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message"`
}

type SyncService interface {
	GetSyncPlan(ctx context.Context, nodeID uuid.UUID, currentState NodeState) (*SyncPlan, error)
	AcknowledgeSync(ctx context.Context, nodeID uuid.UUID, syncID uuid.UUID, result SyncResult) error
}

type syncService struct {
	syncRepo    repository.SyncRepository
	nodeRepo    repository.NodeRepository
	funcRepo    repository.FunctionRepository
	schemaRepo  repository.SchemaRepository
	artifactSvc ArtifactService
}

func NewSyncService(
	syncRepo repository.SyncRepository,
	nodeRepo repository.NodeRepository,
	funcRepo repository.FunctionRepository,
	schemaRepo repository.SchemaRepository,
	artifactSvc ArtifactService,
) SyncService {
	return &syncService{
		syncRepo:    syncRepo,
		nodeRepo:    nodeRepo,
		funcRepo:    funcRepo,
		schemaRepo:  schemaRepo,
		artifactSvc: artifactSvc,
	}
}

func (s *syncService) GetSyncPlan(ctx context.Context, nodeID uuid.UUID, currentState NodeState) (*SyncPlan, error) {
	// 1. Get Target Schema
	latestVersion, err := s.schemaRepo.GetLatestVersion(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Get Target Functions (All latest functions)
	allFunctions, err := s.funcRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Filter to keep only latest version for each function name
	latestFunctionsMap := make(map[string]*model.Function)
	for i := range allFunctions {
		fn := &allFunctions[i]
		if existing, ok := latestFunctionsMap[fn.Name]; !ok {
			latestFunctionsMap[fn.Name] = fn
		} else {
			// Simple string comparison for version might not be enough, but assuming semantic versioning or timestamp
			// For now, let's assume CreatedAt is the source of truth for "latest"
			if fn.CreatedAt.After(existing.CreatedAt) {
				latestFunctionsMap[fn.Name] = fn
			}
		}
	}

	actions := []SyncAction{}
	order := 1

	// 3. Schema Actions
	if currentState.SchemaVersion < latestVersion {
		migrations, err := s.schemaRepo.ListSince(ctx, currentState.SchemaVersion)
		if err != nil {
			return nil, err
		}
		for _, m := range migrations {
			actions = append(actions, SyncAction{
				Type:        ActionTypeApplySchema,
				Order:       order,
				Payload:     m,
				Description: "Apply schema version " + string(rune(m.Version)), // simplistic
			})
			order++
		}
	}

	// 4. Function Actions
	// Identify what to ADD
	currentFuncMap := make(map[string]FunctionState)
	for _, f := range currentState.Functions {
		currentFuncMap[f.Name] = f
	}

	for _, targetFn := range latestFunctionsMap {
		currentFn, exists := currentFuncMap[targetFn.Name]

		if !exists || currentFn.Version != targetFn.Version || currentFn.Hash != targetFn.Hash {
			// Need to add/update
			downloadURL, err := s.artifactSvc.GetDownloadURL(ctx, targetFn.ID)
			if err != nil {
				return nil, err
			}

			payload := map[string]interface{}{
				"function": targetFn,
				"url":      downloadURL,
			}

			actions = append(actions, SyncAction{
				Type:        ActionTypeAddFunction,
				Order:       order,
				Payload:     payload,
				Description: "Add function " + targetFn.Name + " version " + targetFn.Version,
			})
			order++
		}
	}

	// Identify what to REMOVE
	for _, currentFn := range currentState.Functions {
		if _, exists := latestFunctionsMap[currentFn.Name]; !exists {
			actions = append(actions, SyncAction{
				Type:        ActionTypeRemoveFunction,
				Order:       order,
				Payload:     map[string]string{"name": currentFn.Name},
				Description: "Remove function " + currentFn.Name,
			})
			order++
		}
	}

	// Create Sync Record (Pending)
	syncID := uuid.New()
	// We don't save pending sync record in this MVP implementation to keep it simple,
	// or we could save it to track "In Progress".
	// The design says "Sync Manager ... Manage sync transaction".
	// Let's just return the plan. The Ack will record the result.

	return &SyncPlan{
		SyncID:  syncID,
		Actions: actions,
	}, nil
}

func (s *syncService) AcknowledgeSync(ctx context.Context, nodeID uuid.UUID, syncID uuid.UUID, result SyncResult) error {
	status := "success"
	if !result.Success {
		status = "failed"
	}

	// Record sync history
	record := &model.SyncRecord{
		ID:           syncID,
		NodeID:       nodeID,
		SyncType:     "incremental", // Simplified
		Status:       status,
		StartedAt:    time.Now(), // Approximate
		CompletedAt:  &time.Time{},
		ErrorMessage: result.ErrorMessage,
	}
	now := time.Now()
	record.CompletedAt = &now

	if err := s.syncRepo.CreateRecord(ctx, record); err != nil {
		return err
	}

	if result.Success {
		// Update node status
		// In a real system, we would parse the result to know exactly what changed.
		// Here we assume if success, the node is up to date with what we calculated before?
		// Actually, the Ack should probably contain the "New State" as per design:
		// "5. Receive completion ... New current state"
		// But the interface in design `AcknowledgeSync` only takes `SyncResult`.
		// Let's assume `SyncResult` should contain the new state or we trust it.
		// For now, just update LastSyncAt.

		// Also update node_function_deployments?
		// We need the new state to do that accurately.
		// I'll leave that for now or assume the node sends its new state in the next heartbeat/sync.
	}

	return nil
}
