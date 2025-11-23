package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks for Services
type MockNodeService struct {
	mock.Mock
}

func (m *MockNodeService) RegisterNode(ctx context.Context, name, region string) (*model.Node, string, error) {
	args := m.Called(ctx, name, region)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*model.Node), args.String(1), args.Error(2)
}
func (m *MockNodeService) Heartbeat(ctx context.Context, nodeID uuid.UUID) error {
	args := m.Called(ctx, nodeID)
	return args.Error(0)
}
func (m *MockNodeService) GetNode(ctx context.Context, nodeID uuid.UUID) (*model.Node, error) {
	args := m.Called(ctx, nodeID)
	return args.Get(0).(*model.Node), args.Error(1)
}

type MockSyncService struct {
	mock.Mock
}

func (m *MockSyncService) GetSyncPlan(ctx context.Context, nodeID uuid.UUID, currentState service.NodeState) (*service.SyncPlan, error) {
	args := m.Called(ctx, nodeID, currentState)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SyncPlan), args.Error(1)
}
func (m *MockSyncService) AcknowledgeSync(ctx context.Context, nodeID uuid.UUID, syncID uuid.UUID, result service.SyncResult) error {
	args := m.Called(ctx, nodeID, syncID, result)
	return args.Error(0)
}
func (m *MockSyncService) QueueDeployment(ctx context.Context, nodeID, functionID uuid.UUID) error {
	args := m.Called(ctx, nodeID, functionID)
	return args.Error(0)
}
func (m *MockSyncService) CreateRoute(ctx context.Context, host, path, functionID string, methods []string, priority int32, popSelector *string) (interface{}, error) {
	args := m.Called(ctx, host, path, functionID, methods, priority, popSelector)
	return args.Get(0), args.Error(1)
}
func (m *MockSyncService) ListRoutes(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

type MockArtifactService struct {
	mock.Mock
}

func (m *MockArtifactService) UploadFunction(ctx context.Context, name, version string, binary []byte) (*model.Function, error) {
	args := m.Called(ctx, name, version, binary)
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockArtifactService) GetFunction(ctx context.Context, id uuid.UUID) (*model.Function, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockArtifactService) GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}
func (m *MockArtifactService) DeleteFunction(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockArtifactService) CreateFunction(ctx context.Context, name, entrypoint, runtime string, memoryPages, maxExecutionMs int32) (*model.Function, error) {
	args := m.Called(ctx, name, entrypoint, runtime, memoryPages, maxExecutionMs)
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockArtifactService) UploadArtifact(ctx context.Context, id uuid.UUID, binary []byte) (*model.Function, error) {
	args := m.Called(ctx, id, binary)
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockArtifactService) GetArtifactData(ctx context.Context, id, version string) ([]byte, error) {
	args := m.Called(ctx, id, version)
	return args.Get(0).([]byte), args.Error(1)
}

type MockSchemaService struct {
	mock.Mock
}

func (m *MockSchemaService) RegisterSchema(ctx context.Context, version int, upSQL, downSQL, description string) error {
	args := m.Called(ctx, version, upSQL, downSQL, description)
	return args.Error(0)
}
func (m *MockSchemaService) ListSchemas(ctx context.Context) ([]model.SchemaMigration, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.SchemaMigration), args.Error(1)
}

type MockTelemetryService struct {
	mock.Mock
}

func (m *MockTelemetryService) SyncTelemetry(ctx context.Context, batch []model.TelemetryData) (int, error) {
	args := m.Called(ctx, batch)
	return args.Int(0), args.Error(1)
}
func (m *MockTelemetryService) GetCommands(ctx context.Context, deviceID uuid.UUID) ([]model.Command, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Command), args.Error(1)
}
func (m *MockTelemetryService) AckCommand(ctx context.Context, commandID uuid.UUID, success bool) error {
	args := m.Called(ctx, commandID, success)
	return args.Error(0)
}
func (m *MockTelemetryService) GetSyncStatus(ctx context.Context, deviceID uuid.UUID) (*model.SyncStatus, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SyncStatus), args.Error(1)
}
func (m *MockTelemetryService) RegisterDevice(ctx context.Context, name, deviceType, location string) (uuid.UUID, error) {
	args := m.Called(ctx, name, deviceType, location)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func TestRegisterNode(t *testing.T) {
	mockNodeSvc := new(MockNodeService)
	mockSyncSvc := new(MockSyncService)
	mockArtifactSvc := new(MockArtifactService)
	mockSchemaSvc := new(MockSchemaService)
	mockTelemetrySvc := new(MockTelemetryService)

	h := NewHandler(mockNodeSvc, mockSyncSvc, mockArtifactSvc, mockSchemaSvc, mockTelemetrySvc)
	app := fiber.New()
	h.RegisterRoutes(app)

	t.Run("Success", func(t *testing.T) {
		node := &model.Node{Name: "test-node", Region: "us-east-1"}
		token := "secret-token"
		mockNodeSvc.On("RegisterNode", mock.Anything, "test-node", "us-east-1").Return(node, token, nil).Once()

		reqBody := map[string]string{
			"name":   "test-node",
			"region": "us-east-1",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/nodes/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var respBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&respBody)
		assert.Equal(t, token, respBody["token"])

		mockNodeSvc.AssertExpectations(t)
	})
}
