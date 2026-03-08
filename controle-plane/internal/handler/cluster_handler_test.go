package handler

import (
	"bytes"
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

func newClusterTestApp() (*fiber.App, *MockClusterService, *MockClusterInventoryService, *MockClusterSyncService, *auth.Manager) {
	mockNodeSvc := new(MockNodeService)
	mockClusterSvc := new(MockClusterService)
	mockClusterInvSvc := new(MockClusterInventoryService)
	mockClusterSyncSvc := new(MockClusterSyncService)
	mockSyncSvc := new(MockSyncService)
	mockArtifactSvc := new(MockArtifactService)
	mockSchemaSvc := new(MockSchemaService)
	mockTelemetrySvc := new(MockTelemetryService)

	authMgr := auth.NewManager("test-secret")
	h := NewHandler(mockNodeSvc, mockClusterSvc, mockClusterInvSvc, mockClusterSyncSvc, mockSyncSvc, mockArtifactSvc, mockSchemaSvc, mockTelemetrySvc, authMgr, time.Hour, nil, nil)
	app := fiber.New()
	h.RegisterRoutes(app)
	return app, mockClusterSvc, mockClusterInvSvc, mockClusterSyncSvc, authMgr
}

func TestRegisterCluster(t *testing.T) {
	app, mockClusterSvc, _, _, _ := newClusterTestApp()
	cluster := &model.Cluster{
		ID:          uuid.New(),
		Name:        "tokyo-edge-1",
		Region:      "ap-northeast-1",
		Environment: "prod",
		APIEndpoint: "https://10.0.0.10:6443",
		Status:      model.ClusterStatusOnline,
	}

	mockClusterSvc.On("RegisterCluster", mock.Anything, service.RegisterClusterInput{
		Name:        "tokyo-edge-1",
		Region:      "ap-northeast-1",
		Environment: "prod",
		APIEndpoint: "https://10.0.0.10:6443",
		Labels:      map[string]string{"tier": "edge"},
	}).Return(cluster, "raw-token", nil).Once()

	reqBody := map[string]interface{}{
		"name":         "tokyo-edge-1",
		"region":       "ap-northeast-1",
		"environment":  "prod",
		"api_endpoint": "https://10.0.0.10:6443",
		"labels":       map[string]string{"tier": "edge"},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/clusters/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var payload map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	assert.NotEmpty(t, payload["token"])
	mockClusterSvc.AssertExpectations(t)
}

func TestClusterInventory(t *testing.T) {
	app, _, mockClusterInvSvc, _, authMgr := newClusterTestApp()
	clusterID := uuid.New()
	token, _ := authMgr.GenerateToken(clusterID, time.Hour)

	mockClusterInvSvc.On("UpdateInventory", mock.Anything, clusterID, mock.MatchedBy(func(in service.ClusterInventoryInput) bool {
		return len(in.Nodes) == 1 && in.Nodes[0].NodeName == "node-a"
	})).Return(nil).Once()

	reqBody := map[string]interface{}{
		"nodes": []map[string]string{
			{
				"node_name":   "node-a",
				"role":        "worker",
				"status":      "ready",
				"internal_ip": "10.0.0.11",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	url := "/api/v1/clusters/" + clusterID.String() + "/inventory"
	req := httptest.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockClusterInvSvc.AssertExpectations(t)
}

func TestClusterSync(t *testing.T) {
	app, _, _, mockClusterSyncSvc, authMgr := newClusterTestApp()
	clusterID := uuid.New()
	token, _ := authMgr.GenerateToken(clusterID, time.Hour)

	mockClusterSyncSvc.On("GetSyncPlan", mock.Anything, clusterID, service.ClusterState{}).Return(&service.ClusterSyncPlan{
		SyncID: uuid.New(),
		DesiredState: service.ClusterDesiredState{
			SchemaVersion: 2,
		},
	}, nil).Once()

	url := "/api/v1/clusters/" + clusterID.String() + "/sync"
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockClusterSyncSvc.AssertExpectations(t)
}
