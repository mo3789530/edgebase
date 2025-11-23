package service

import (
	"context"
	"testing"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
type MockSyncRepository struct {
	mock.Mock
}

func (m *MockSyncRepository) CreateRecord(ctx context.Context, record *model.SyncRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockSyncRepository) UpdateRecord(ctx context.Context, record *model.SyncRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockSyncRepository) GetNodeDeployments(ctx context.Context, nodeID uuid.UUID) ([]model.NodeFunctionDeployment, error) {
	args := m.Called(ctx, nodeID)
	return args.Get(0).([]model.NodeFunctionDeployment), args.Error(1)
}

func (m *MockSyncRepository) UpdateNodeDeployments(ctx context.Context, nodeID uuid.UUID, deployments []model.NodeFunctionDeployment) error {
	args := m.Called(ctx, nodeID, deployments)
	return args.Error(0)
}

type MockNodeRepository struct {
	mock.Mock
}

func (m *MockNodeRepository) Create(ctx context.Context, node *model.Node) error {
	args := m.Called(ctx, node)
	return args.Error(0)
}
func (m *MockNodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Node, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Node), args.Error(1)
}
func (m *MockNodeRepository) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockNodeRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NodeStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
func (m *MockNodeRepository) List(ctx context.Context) ([]model.Node, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Node), args.Error(1)
}

type MockFunctionRepository struct {
	mock.Mock
}

func (m *MockFunctionRepository) Create(ctx context.Context, function *model.Function) error {
	args := m.Called(ctx, function)
	return args.Error(0)
}
func (m *MockFunctionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Function, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockFunctionRepository) GetByNameAndVersion(ctx context.Context, name, version string) (*model.Function, error) {
	args := m.Called(ctx, name, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockFunctionRepository) List(ctx context.Context) ([]model.Function, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Function), args.Error(1)
}
func (m *MockFunctionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockFunctionRepository) Update(ctx context.Context, function *model.Function) error {
	args := m.Called(ctx, function)
	return args.Error(0)
}

type MockSchemaRepository struct {
	mock.Mock
}

func (m *MockSchemaRepository) Create(ctx context.Context, schema *model.SchemaMigration) error {
	args := m.Called(ctx, schema)
	return args.Error(0)
}
func (m *MockSchemaRepository) GetLatestVersion(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}
func (m *MockSchemaRepository) ListSince(ctx context.Context, version int) ([]model.SchemaMigration, error) {
	args := m.Called(ctx, version)
	return args.Get(0).([]model.SchemaMigration), args.Error(1)
}

type MockArtifactService struct {
	mock.Mock
}

func (m *MockArtifactService) UploadFunction(ctx context.Context, name, version string, binary []byte) (*model.Function, error) {
	args := m.Called(ctx, name, version, binary)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockArtifactService) GetFunction(ctx context.Context, id uuid.UUID) (*model.Function, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockArtifactService) UploadArtifact(ctx context.Context, id uuid.UUID, binary []byte) (*model.Function, error) {
	args := m.Called(ctx, id, binary)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Function), args.Error(1)
}
func (m *MockArtifactService) GetArtifactData(ctx context.Context, id, version string) ([]byte, error) {
	args := m.Called(ctx, id, version)
	return args.Get(0).([]byte), args.Error(1)
}

func TestGetSyncPlan(t *testing.T) {
	mockSyncRepo := new(MockSyncRepository)
	mockNodeRepo := new(MockNodeRepository)
	mockFuncRepo := new(MockFunctionRepository)
	mockSchemaRepo := new(MockSchemaRepository)
	mockArtifactSvc := new(MockArtifactService)

	svc := NewSyncService(mockSyncRepo, mockNodeRepo, mockFuncRepo, mockSchemaRepo, mockArtifactSvc)
	ctx := context.Background()
	nodeID := uuid.New()

	t.Run("No changes needed", func(t *testing.T) {
		// Setup
		mockSchemaRepo.On("GetLatestVersion", ctx).Return(1, nil).Once()
		mockFuncRepo.On("List", ctx).Return([]model.Function{}, nil).Once()

		currentState := NodeState{
			SchemaVersion: 1,
			Functions:     []FunctionState{},
		}

		// Execute
		plan, err := svc.GetSyncPlan(ctx, nodeID, currentState)

		// Verify
		assert.NoError(t, err)
		assert.NotNil(t, plan)
		assert.Empty(t, plan.Actions)
	})

	t.Run("Schema update needed", func(t *testing.T) {
		// Setup
		mockSchemaRepo.On("GetLatestVersion", ctx).Return(2, nil).Once()
		mockFuncRepo.On("List", ctx).Return([]model.Function{}, nil).Once()

		migrations := []model.SchemaMigration{
			{Version: 2, UpSQL: "CREATE TABLE test..."},
		}
		mockSchemaRepo.On("ListSince", ctx, 1).Return(migrations, nil).Once()

		currentState := NodeState{
			SchemaVersion: 1,
			Functions:     []FunctionState{},
		}

		// Execute
		plan, err := svc.GetSyncPlan(ctx, nodeID, currentState)

		// Verify
		assert.NoError(t, err)
		assert.Len(t, plan.Actions, 1)
		assert.Equal(t, ActionTypeApplySchema, plan.Actions[0].Type)
		assert.Equal(t, 2, plan.Actions[0].Payload.(model.SchemaMigration).Version)
	})

	t.Run("Function add needed", func(t *testing.T) {
		// Setup
		mockSchemaRepo.On("GetLatestVersion", ctx).Return(1, nil).Once()

		fnID := uuid.New()
		targetFn := model.Function{
			ID:        fnID,
			Name:      "func1",
			Version:   "v1",
			Hash:      "hash1",
			CreatedAt: time.Now(),
		}
		mockFuncRepo.On("List", ctx).Return([]model.Function{targetFn}, nil).Once()
		mockArtifactSvc.On("GetDownloadURL", ctx, fnID).Return("http://minio/func1", nil).Once()

		currentState := NodeState{
			SchemaVersion: 1,
			Functions:     []FunctionState{},
		}

		// Execute
		plan, err := svc.GetSyncPlan(ctx, nodeID, currentState)

		// Verify
		assert.NoError(t, err)
		assert.Len(t, plan.Actions, 1)
		assert.Equal(t, ActionTypeAddFunction, plan.Actions[0].Type)
		payload := plan.Actions[0].Payload.(map[string]interface{})
		assert.Equal(t, "http://minio/func1", payload["url"])
	})

	t.Run("Function remove needed", func(t *testing.T) {
		// Setup
		mockSchemaRepo.On("GetLatestVersion", ctx).Return(1, nil).Once()
		mockFuncRepo.On("List", ctx).Return([]model.Function{}, nil).Once()

		currentState := NodeState{
			SchemaVersion: 1,
			Functions: []FunctionState{
				{Name: "old-func", Version: "v1", Hash: "old-hash"},
			},
		}

		// Execute
		plan, err := svc.GetSyncPlan(ctx, nodeID, currentState)

		// Verify
		assert.NoError(t, err)
		assert.Len(t, plan.Actions, 1)
		assert.Equal(t, ActionTypeRemoveFunction, plan.Actions[0].Type)
		payload := plan.Actions[0].Payload.(map[string]string)
		assert.Equal(t, "old-func", payload["name"])
	})
}

func TestAcknowledgeSync(t *testing.T) {
	mockSyncRepo := new(MockSyncRepository)
	mockNodeRepo := new(MockNodeRepository)
	mockFuncRepo := new(MockFunctionRepository)
	mockSchemaRepo := new(MockSchemaRepository)
	mockArtifactSvc := new(MockArtifactService)

	svc := NewSyncService(mockSyncRepo, mockNodeRepo, mockFuncRepo, mockSchemaRepo, mockArtifactSvc)
	ctx := context.Background()
	nodeID := uuid.New()
	syncID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockSyncRepo.On("CreateRecord", ctx, mock.AnythingOfType("*model.SyncRecord")).Return(nil).Once()

		result := SyncResult{Success: true}
		err := svc.AcknowledgeSync(ctx, nodeID, syncID, result)

		assert.NoError(t, err)
		mockSyncRepo.AssertExpectations(t)
	})
}
