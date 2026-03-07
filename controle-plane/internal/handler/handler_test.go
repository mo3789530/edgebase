package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edgebase/platform/control-plane/internal/auth"
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

type MockFunctionCatalogService struct {
	mock.Mock
}

func (m *MockFunctionCatalogService) CreateDefinition(ctx context.Context, name, description, runtimeKind string, defaultTimeoutSeconds, defaultMemoryMB, defaultCPUMillis int32) (*model.FunctionDefinition, error) {
	args := m.Called(ctx, name, description, runtimeKind, defaultTimeoutSeconds, defaultMemoryMB, defaultCPUMillis)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FunctionDefinition), args.Error(1)
}

func (m *MockFunctionCatalogService) GetDefinition(ctx context.Context, id uuid.UUID) (*model.FunctionDefinition, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FunctionDefinition), args.Error(1)
}

func (m *MockFunctionCatalogService) ListDefinitions(ctx context.Context) ([]model.FunctionDefinition, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FunctionDefinition), args.Error(1)
}

func (m *MockFunctionCatalogService) CreateRevision(ctx context.Context, functionDefinitionID uuid.UUID, input service.CreateFunctionRevisionInput) (*model.FunctionRevision, error) {
	args := m.Called(ctx, functionDefinitionID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FunctionRevision), args.Error(1)
}

type MockFunctionDeploymentService struct {
	mock.Mock
}

func (m *MockFunctionDeploymentService) CreateTargets(ctx context.Context, functionDefinitionID uuid.UUID, input service.CreateDeploymentTargetsInput) ([]model.FunctionDeploymentTarget, error) {
	args := m.Called(ctx, functionDefinitionID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FunctionDeploymentTarget), args.Error(1)
}

type MockFunctionControllerService struct {
	mock.Mock
}

func (m *MockFunctionControllerService) GetClusterSyncPlan(ctx context.Context, clusterID uuid.UUID) (*service.SyncPlan, error) {
	args := m.Called(ctx, clusterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SyncPlan), args.Error(1)
}

func (m *MockFunctionControllerService) AcknowledgeClusterSync(ctx context.Context, clusterID, syncID uuid.UUID, result service.SyncResult) error {
	args := m.Called(ctx, clusterID, syncID, result)
	return args.Error(0)
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
func (m *MockSchemaService) GetSchema(ctx context.Context, version int) (*model.SchemaMigration, error) {
	args := m.Called(ctx, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SchemaMigration), args.Error(1)
}
func (m *MockSchemaService) UpdateNodeStatus(ctx context.Context, nodeID uuid.UUID, version int, status, errorMessage string) error {
	args := m.Called(ctx, nodeID, version, status, errorMessage)
	return args.Error(0)
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

type MockInventoryService struct {
	mock.Mock
}

func (m *MockInventoryService) SaveSnapshot(ctx context.Context, clusterID uuid.UUID, in service.ClusterInventoryInput) error {
	args := m.Called(ctx, clusterID, in)
	return args.Error(0)
}

func TestRegisterNode(t *testing.T) {
	mockNodeSvc := new(MockNodeService)
	mockSyncSvc := new(MockSyncService)
	mockArtifactSvc := new(MockArtifactService)
	mockFunctionSvc := new(MockFunctionCatalogService)
	mockDeploymentSvc := new(MockFunctionDeploymentService)
	mockControllerSvc := new(MockFunctionControllerService)
	mockSchemaSvc := new(MockSchemaService)
	mockTelemetrySvc := new(MockTelemetryService)
	mockInventorySvc := new(MockInventoryService)

	authMgr := auth.NewManager("test-secret")
	h := NewHandler(mockNodeSvc, mockSyncSvc, mockArtifactSvc, mockFunctionSvc, mockDeploymentSvc, mockControllerSvc, mockSchemaSvc, mockTelemetrySvc, mockInventorySvc, authMgr, time.Hour, nil, nil)
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
		assert.NotEmpty(t, respBody["token"])

		tokenStr, ok := respBody["token"].(string)
		assert.True(t, ok)
		_, err = authMgr.VerifyToken(tokenStr)
		assert.NoError(t, err)

		mockNodeSvc.AssertExpectations(t)
	})
}

func TestFunctionRoutes(t *testing.T) {
	mockNodeSvc := new(MockNodeService)
	mockSyncSvc := new(MockSyncService)
	mockArtifactSvc := new(MockArtifactService)
	mockFunctionSvc := new(MockFunctionCatalogService)
	mockDeploymentSvc := new(MockFunctionDeploymentService)
	mockControllerSvc := new(MockFunctionControllerService)
	mockSchemaSvc := new(MockSchemaService)
	mockTelemetrySvc := new(MockTelemetryService)
	mockInventorySvc := new(MockInventoryService)

	authMgr := auth.NewManager("test-secret")
	h := NewHandler(mockNodeSvc, mockSyncSvc, mockArtifactSvc, mockFunctionSvc, mockDeploymentSvc, mockControllerSvc, mockSchemaSvc, mockTelemetrySvc, mockInventorySvc, authMgr, time.Hour, nil, nil)
	app := fiber.New()
	h.RegisterRoutes(app)

	token, err := authMgr.GenerateToken(uuid.New(), time.Hour)
	assert.NoError(t, err)

	t.Run("CreateFunctionDefinition", func(t *testing.T) {
		definition := &model.FunctionDefinition{
			ID:                    uuid.New(),
			Name:                  "telemetry-normalizer",
			RuntimeKind:           "container",
			DefaultTimeoutSeconds: 3,
			DefaultMemoryMB:       128,
			DefaultCPUMillis:      250,
		}
		mockFunctionSvc.On("CreateDefinition", mock.Anything, "telemetry-normalizer", "normalize telemetry", "container", int32(3), int32(128), int32(250)).Return(definition, nil).Once()

		body, _ := json.Marshal(map[string]interface{}{
			"name":                    "telemetry-normalizer",
			"description":             "normalize telemetry",
			"runtime_kind":            "container",
			"default_timeout_seconds": 3,
			"default_memory_mb":       128,
			"default_cpu_millis":      250,
		})
		req := httptest.NewRequest("POST", "/api/v1/functions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		mockFunctionSvc.AssertExpectations(t)
	})

}

func TestClusterCompatibilityRoutes(t *testing.T) {
	newApp := func(clusterID uuid.UUID) (*fiber.App, *auth.Manager, *MockNodeService, *MockFunctionControllerService, *MockInventoryService) {
		mockNodeSvc := new(MockNodeService)
		mockSyncSvc := new(MockSyncService)
		mockArtifactSvc := new(MockArtifactService)
		mockFunctionSvc := new(MockFunctionCatalogService)
		mockDeploymentSvc := new(MockFunctionDeploymentService)
		mockControllerSvc := new(MockFunctionControllerService)
		mockSchemaSvc := new(MockSchemaService)
		mockTelemetrySvc := new(MockTelemetryService)
		mockInventorySvc := new(MockInventoryService)

		authMgr := auth.NewManager("test-secret")
		h := NewHandler(mockNodeSvc, mockSyncSvc, mockArtifactSvc, mockFunctionSvc, mockDeploymentSvc, mockControllerSvc, mockSchemaSvc, mockTelemetrySvc, mockInventorySvc, authMgr, time.Hour, nil, nil)
		app := fiber.New()
		h.RegisterRoutes(app)
		return app, authMgr, mockNodeSvc, mockControllerSvc, mockInventorySvc
	}

	t.Run("ClusterHeartbeat delegates to node heartbeat", func(t *testing.T) {
		clusterID := uuid.New()
		app, authMgr, mockNodeSvc, _, _ := newApp(clusterID)
		token, err := authMgr.GenerateToken(clusterID, time.Hour)
		assert.NoError(t, err)

		mockNodeSvc.On("Heartbeat", mock.Anything, clusterID).Return(nil).Once()

		req := httptest.NewRequest("POST", "/api/v1/clusters/"+clusterID.String()+"/heartbeat", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockNodeSvc.AssertExpectations(t)
	})

	t.Run("ClusterSync delegates to controller service", func(t *testing.T) {
		clusterID := uuid.New()
		app, authMgr, _, mockControllerSvc, _ := newApp(clusterID)
		token, err := authMgr.GenerateToken(clusterID, time.Hour)
		assert.NoError(t, err)

		mockControllerSvc.On("GetClusterSyncPlan", mock.Anything, clusterID).Return(&service.SyncPlan{
			SyncID:     uuid.New(),
			Generation: 1,
			Actions:    []service.SyncAction{},
		}, nil).Once()

		req := httptest.NewRequest("GET", "/api/v1/clusters/"+clusterID.String()+"/sync", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockControllerSvc.AssertExpectations(t)
	})

	t.Run("ClusterInventory accepts payload", func(t *testing.T) {
		clusterID := uuid.New()
		app, authMgr, _, _, mockInventorySvc := newApp(clusterID)
		token, err := authMgr.GenerateToken(clusterID, time.Hour)
		assert.NoError(t, err)
		mockInventorySvc.On("SaveSnapshot", mock.Anything, clusterID, mock.AnythingOfType("service.ClusterInventoryInput")).Return(nil).Once()

		body, _ := json.Marshal(map[string]interface{}{
			"cluster_id":         clusterID.String(),
			"observed_at":        time.Now().UTC().Format(time.RFC3339),
			"kubernetes_version": "v1.31.2+k3s1",
			"nodes":              []map[string]interface{}{},
			"deployments":        []map[string]interface{}{},
			"services":           []map[string]interface{}{},
			"pods":               []map[string]interface{}{},
		})
		req := httptest.NewRequest("POST", "/api/v1/clusters/"+clusterID.String()+"/inventory", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		mockInventorySvc.AssertExpectations(t)
	})
}
