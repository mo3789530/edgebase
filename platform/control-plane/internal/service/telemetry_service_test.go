package service

import (
	"context"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTelemetryRepository struct {
	mock.Mock
}

func (m *MockTelemetryRepository) InsertBatch(ctx context.Context, batch []model.TelemetryData) (int, error) {
	args := m.Called(ctx, batch)
	return args.Int(0), args.Error(1)
}

func (m *MockTelemetryRepository) GetPendingCommands(ctx context.Context, deviceID uuid.UUID) ([]model.Command, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Command), args.Error(1)
}

func (m *MockTelemetryRepository) UpdateCommandStatus(ctx context.Context, commandID uuid.UUID, success bool) error {
	args := m.Called(ctx, commandID, success)
	return args.Error(0)
}

func (m *MockTelemetryRepository) GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (*model.SyncStatus, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SyncStatus), args.Error(1)
}

func (m *MockTelemetryRepository) RegisterDevice(ctx context.Context, name, deviceType, location string) (uuid.UUID, error) {
	args := m.Called(ctx, name, deviceType, location)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockTelemetryRepository) UpdateDeviceLastSeen(ctx context.Context, deviceID uuid.UUID) error {
	args := m.Called(ctx, deviceID)
	return args.Error(0)
}

func TestSyncTelemetry(t *testing.T) {
	mockRepo := new(MockTelemetryRepository)
	deviceID := uuid.New()
	batch := []model.TelemetryData{
		{
			ID:       uuid.New(),
			DeviceID: deviceID,
			SensorID: "sensor1",
			DataType: "temperature",
			Value:    25.5,
		},
	}

	mockRepo.On("InsertBatch", mock.Anything, batch).Return(1, nil).Once()
	mockRepo.On("UpdateDeviceLastSeen", mock.Anything, deviceID).Return(nil).Once()

	svc := NewTelemetryService(mockRepo)
	inserted, err := svc.SyncTelemetry(context.Background(), batch)

	assert.NoError(t, err)
	assert.Equal(t, 1, inserted)
	mockRepo.AssertExpectations(t)
}

func TestGetCommands(t *testing.T) {
	mockRepo := new(MockTelemetryRepository)
	deviceID := uuid.New()
	commands := []model.Command{
		{
			ID:       uuid.New(),
			DeviceID: deviceID,
			Type:     "restart",
			Status:   "pending",
		},
	}

	mockRepo.On("GetPendingCommands", mock.Anything, deviceID).Return(commands, nil).Once()

	svc := NewTelemetryService(mockRepo)
	result, err := svc.GetCommands(context.Background(), deviceID)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	mockRepo.AssertExpectations(t)
}

func TestRegisterDevice(t *testing.T) {
	mockRepo := new(MockTelemetryRepository)
	deviceID := uuid.New()

	mockRepo.On("RegisterDevice", mock.Anything, "device1", "sensor", "location1").Return(deviceID, nil).Once()

	svc := NewTelemetryService(mockRepo)
	result, err := svc.RegisterDevice(context.Background(), "device1", "sensor", "location1")

	assert.NoError(t, err)
	assert.Equal(t, deviceID, result)
	mockRepo.AssertExpectations(t)
}
