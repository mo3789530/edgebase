package service

import (
	"context"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type TelemetryService interface {
	SyncTelemetry(ctx context.Context, batch []model.TelemetryData) (int, error)
	GetCommands(ctx context.Context, deviceID uuid.UUID) ([]model.Command, error)
	AckCommand(ctx context.Context, commandID uuid.UUID, success bool) error
	GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (*model.SyncStatus, error)
	RegisterDevice(ctx context.Context, name, deviceType, location string) (uuid.UUID, error)
}

type telemetryService struct {
	repo repository.TelemetryRepository
}

func NewTelemetryService(repo repository.TelemetryRepository) TelemetryService {
	return &telemetryService{repo: repo}
}

func (s *telemetryService) SyncTelemetry(ctx context.Context, batch []model.TelemetryData) (int, error) {
	if len(batch) == 0 {
		return 0, nil
	}

	inserted, err := s.repo.InsertBatch(ctx, batch)
	if err != nil {
		return 0, err
	}

	if err := s.repo.UpdateDeviceLastSeen(ctx, batch[0].DeviceID); err != nil {
		return inserted, err
	}

	return inserted, nil
}

func (s *telemetryService) GetCommands(ctx context.Context, deviceID uuid.UUID) ([]model.Command, error) {
	return s.repo.GetPendingCommands(ctx, deviceID)
}

func (s *telemetryService) AckCommand(ctx context.Context, commandID uuid.UUID, success bool) error {
	return s.repo.UpdateCommandStatus(ctx, commandID, success)
}

func (s *telemetryService) GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (*model.SyncStatus, error) {
	return s.repo.GetSyncStatus(ctx, deviceID)
}

func (s *telemetryService) RegisterDevice(ctx context.Context, name, deviceType, location string) (uuid.UUID, error) {
	return s.repo.RegisterDevice(ctx, name, deviceType, location)
}
