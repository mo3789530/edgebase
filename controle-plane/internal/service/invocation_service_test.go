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

type MockInvocationRepository struct {
	mock.Mock
}

func (m *MockInvocationRepository) Create(ctx context.Context, invocation *model.Invocation) error {
	args := m.Called(ctx, invocation)
	return args.Error(0)
}

func (m *MockInvocationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Invocation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Invocation), args.Error(1)
}

func (m *MockInvocationRepository) UpdateCompletion(ctx context.Context, id uuid.UUID, completedAt time.Time, finalStatus string, clientStatusCode *int) error {
	args := m.Called(ctx, id, completedAt, finalStatus, clientStatusCode)
	return args.Error(0)
}

type MockInvocationAttemptRepository struct {
	mock.Mock
}

func (m *MockInvocationAttemptRepository) Create(ctx context.Context, attempt *model.InvocationAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *MockInvocationAttemptRepository) ListByInvocationID(ctx context.Context, invocationID uuid.UUID) ([]model.InvocationAttempt, error) {
	args := m.Called(ctx, invocationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.InvocationAttempt), args.Error(1)
}

func (m *MockInvocationAttemptRepository) UpdateCompletion(ctx context.Context, id uuid.UUID, completedAt time.Time, status string, statusCode *int, errorMessage string) error {
	args := m.Called(ctx, id, completedAt, status, statusCode, errorMessage)
	return args.Error(0)
}

func TestInvocationService_StartAndComplete(t *testing.T) {
	ctx := context.Background()
	invocationRepo := new(MockInvocationRepository)
	attemptRepo := new(MockInvocationAttemptRepository)
	svc := NewInvocationService(invocationRepo, attemptRepo)

	functionID := uuid.New()
	requestID := "req-1"
	startedAt := time.Now().UTC()

	invocationRepo.On("Create", ctx, mock.MatchedBy(func(inv *model.Invocation) bool {
		return inv.FunctionDefinitionID == functionID &&
			inv.TriggerType == "http" &&
			inv.RequestID == requestID &&
			inv.FinalStatus == "started"
	})).Return(nil).Once()

	invocation, err := svc.StartInvocation(ctx, StartInvocationInput{
		FunctionDefinitionID: functionID,
		TriggerType:          "http",
		RequestID:            requestID,
		StartedAt:            startedAt,
	})

	assert.NoError(t, err)
	assert.Equal(t, functionID, invocation.FunctionDefinitionID)

	completedAt := startedAt.Add(time.Second)
	statusCode := 200
	invocationRepo.On("UpdateCompletion", ctx, invocation.ID, completedAt, "succeeded", &statusCode).Return(nil).Once()

	err = svc.CompleteInvocation(ctx, invocation.ID, CompleteInvocationInput{
		CompletedAt:      completedAt,
		FinalStatus:      "succeeded",
		ClientStatusCode: &statusCode,
	})

	assert.NoError(t, err)
	invocationRepo.AssertExpectations(t)
}

func TestInvocationService_RecordAttemptAndGet(t *testing.T) {
	ctx := context.Background()
	invocationRepo := new(MockInvocationRepository)
	attemptRepo := new(MockInvocationAttemptRepository)
	svc := NewInvocationService(invocationRepo, attemptRepo)

	invocationID := uuid.New()
	clusterID := uuid.New()
	startedAt := time.Now().UTC()

	attemptRepo.On("Create", ctx, mock.MatchedBy(func(attempt *model.InvocationAttempt) bool {
		return attempt.InvocationID == invocationID &&
			attempt.ClusterID == clusterID &&
			attempt.KnativeService == "telemetry-normalizer" &&
			attempt.AttemptNo == 1
	})).Return(nil).Once()

	attempt, err := svc.RecordAttempt(ctx, invocationID, RecordAttemptInput{
		ClusterID:      clusterID,
		KnativeService: "telemetry-normalizer",
		AttemptNo:      1,
		StartedAt:      startedAt,
	})

	assert.NoError(t, err)
	assert.Equal(t, invocationID, attempt.InvocationID)

	invocationRepo.On("GetByID", ctx, invocationID).Return(&model.Invocation{
		ID:                   invocationID,
		FunctionDefinitionID: uuid.New(),
		TriggerType:          "http",
		RequestID:            "req-2",
		StartedAt:            startedAt,
		FinalStatus:          "started",
	}, nil).Once()
	attemptRepo.On("ListByInvocationID", ctx, invocationID).Return([]model.InvocationAttempt{*attempt}, nil).Once()

	detail, err := svc.GetInvocation(ctx, invocationID)

	assert.NoError(t, err)
	assert.Len(t, detail.Attempts, 1)
	attemptRepo.AssertExpectations(t)
}
