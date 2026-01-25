package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/mqtt"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type SchemaService interface {
	RegisterSchema(ctx context.Context, version int, upSQL, downSQL, description string) error
	ListSchemas(ctx context.Context) ([]model.SchemaMigration, error)
	GetSchema(ctx context.Context, version int) (*model.SchemaMigration, error)
	UpdateNodeStatus(ctx context.Context, nodeID uuid.UUID, version int, status, errorMessage string) error
}

type schemaService struct {
	repo       repository.SchemaRepository
	mqttClient *mqtt.Client
}

func NewSchemaService(repo repository.SchemaRepository, mqttClient *mqtt.Client) SchemaService {
	return &schemaService{
		repo:       repo,
		mqttClient: mqttClient,
	}
}

func (s *schemaService) RegisterSchema(ctx context.Context, version int, upSQL, downSQL, description string) error {
	// 1. Check version sequence
	latestVersion, err := s.repo.GetLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}
	if version <= latestVersion {
		return fmt.Errorf("version must be greater than latest version %d", latestVersion)
	}

	// 2. Calculate checksum (SHA256 of UpSQL)
	hash := sha256.Sum256([]byte(upSQL))
	checksum := hex.EncodeToString(hash[:])

	schema := &model.SchemaMigration{
		Version:     version,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
		Description: description,
		Checksum:    checksum,
	}

	// 3. Save to DB
	if err := s.repo.Create(ctx, schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// 4. Send Notification via MQTT
	if s.mqttClient != nil {
		// Topic: sys/schema/update
		// Payload: {"version": 2, "checksum": "..."}
		payload := fmt.Sprintf(`{"version": %d, "checksum": "%s"}`, version, checksum)
		if err := s.mqttClient.Publish("sys/schema/update", payload); err != nil {
			// Log error but don't fail the request as schema is already saved
			// In a real system, we might want to have a reliable delivery mechanism
			fmt.Printf("failed to publish schema update notification: %v\n", err)
		}
	}

	return nil
}

func (s *schemaService) ListSchemas(ctx context.Context) ([]model.SchemaMigration, error) {
	// List all schemas. The repository has ListSince, let's use that with version 0
	return s.repo.ListSince(ctx, 0)
}

func (s *schemaService) GetSchema(ctx context.Context, version int) (*model.SchemaMigration, error) {
	return s.repo.GetByVersion(ctx, version)
}

func (s *schemaService) UpdateNodeStatus(ctx context.Context, nodeID uuid.UUID, version int, status, errorMessage string) error {
	nodeStatus := &model.NodeSchemaStatus{
		NodeID:         nodeID,
		CurrentVersion: version,
		Status:         status,
		ErrorMessage:   errorMessage,
		LastUpdated:    time.Now(),
	}
	return s.repo.UpdateNodeStatus(ctx, nodeStatus)
}
