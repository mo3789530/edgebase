package service

import (
	"context"
	"errors"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type InvocationService interface {
	StartInvocation(ctx context.Context, input StartInvocationInput) (*model.Invocation, error)
	CompleteInvocation(ctx context.Context, invocationID uuid.UUID, input CompleteInvocationInput) error
	RecordAttempt(ctx context.Context, invocationID uuid.UUID, input RecordAttemptInput) (*model.InvocationAttempt, error)
	CompleteAttempt(ctx context.Context, attemptID uuid.UUID, input CompleteAttemptInput) error
	GetInvocation(ctx context.Context, invocationID uuid.UUID) (*InvocationDetail, error)
}

type StartInvocationInput struct {
	RouteID              *uuid.UUID
	FunctionDefinitionID uuid.UUID
	TriggerType          string
	RequestID            string
	StartedAt            time.Time
}

type CompleteInvocationInput struct {
	CompletedAt      time.Time
	FinalStatus      string
	ClientStatusCode *int
}

type RecordAttemptInput struct {
	ClusterID       uuid.UUID
	KnativeService  string
	KnativeRevision string
	PodName         string
	AttemptNo       int
	StartedAt       time.Time
	CompletedAt     *time.Time
	Status          string
	StatusCode      *int
	ErrorMessage    string
}

type CompleteAttemptInput struct {
	CompletedAt  time.Time
	Status       string
	StatusCode   *int
	ErrorMessage string
}

type InvocationDetail struct {
	Invocation model.Invocation          `json:"invocation"`
	Attempts   []model.InvocationAttempt `json:"attempts"`
}

type invocationService struct {
	invocationRepo        repository.InvocationRepository
	invocationAttemptRepo repository.InvocationAttemptRepository
}

func NewInvocationService(
	invocationRepo repository.InvocationRepository,
	invocationAttemptRepo repository.InvocationAttemptRepository,
) InvocationService {
	return &invocationService{
		invocationRepo:        invocationRepo,
		invocationAttemptRepo: invocationAttemptRepo,
	}
}

func (s *invocationService) StartInvocation(ctx context.Context, input StartInvocationInput) (*model.Invocation, error) {
	if input.FunctionDefinitionID == uuid.Nil {
		return nil, errors.New("function_definition_id is required")
	}
	if input.TriggerType == "" {
		return nil, errors.New("trigger_type is required")
	}
	if input.RequestID == "" {
		return nil, errors.New("request_id is required")
	}
	if input.StartedAt.IsZero() {
		input.StartedAt = time.Now().UTC()
	}

	invocation := &model.Invocation{
		ID:                   uuid.New(),
		RouteID:              input.RouteID,
		FunctionDefinitionID: input.FunctionDefinitionID,
		TriggerType:          input.TriggerType,
		RequestID:            input.RequestID,
		StartedAt:            input.StartedAt,
		FinalStatus:          "started",
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	if err := s.invocationRepo.Create(ctx, invocation); err != nil {
		return nil, err
	}
	return invocation, nil
}

func (s *invocationService) CompleteInvocation(ctx context.Context, invocationID uuid.UUID, input CompleteInvocationInput) error {
	if invocationID == uuid.Nil {
		return errors.New("invocation_id is required")
	}
	if input.FinalStatus == "" {
		return errors.New("final_status is required")
	}
	if input.CompletedAt.IsZero() {
		input.CompletedAt = time.Now().UTC()
	}
	return s.invocationRepo.UpdateCompletion(ctx, invocationID, input.CompletedAt, input.FinalStatus, input.ClientStatusCode)
}

func (s *invocationService) RecordAttempt(ctx context.Context, invocationID uuid.UUID, input RecordAttemptInput) (*model.InvocationAttempt, error) {
	if invocationID == uuid.Nil {
		return nil, errors.New("invocation_id is required")
	}
	if input.ClusterID == uuid.Nil {
		return nil, errors.New("cluster_id is required")
	}
	if input.KnativeService == "" {
		return nil, errors.New("knative_service is required")
	}
	if input.AttemptNo <= 0 {
		input.AttemptNo = 1
	}
	if input.StartedAt.IsZero() {
		input.StartedAt = time.Now().UTC()
	}
	if input.Status == "" {
		input.Status = "started"
	}

	attempt := &model.InvocationAttempt{
		ID:              uuid.New(),
		InvocationID:    invocationID,
		ClusterID:       input.ClusterID,
		KnativeService:  input.KnativeService,
		KnativeRevision: input.KnativeRevision,
		PodName:         input.PodName,
		AttemptNo:       input.AttemptNo,
		StartedAt:       input.StartedAt,
		CompletedAt:     input.CompletedAt,
		Status:          input.Status,
		StatusCode:      input.StatusCode,
		ErrorMessage:    input.ErrorMessage,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	if err := s.invocationAttemptRepo.Create(ctx, attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *invocationService) CompleteAttempt(ctx context.Context, attemptID uuid.UUID, input CompleteAttemptInput) error {
	if attemptID == uuid.Nil {
		return errors.New("attempt_id is required")
	}
	if input.Status == "" {
		return errors.New("status is required")
	}
	if input.CompletedAt.IsZero() {
		input.CompletedAt = time.Now().UTC()
	}
	return s.invocationAttemptRepo.UpdateCompletion(ctx, attemptID, input.CompletedAt, input.Status, input.StatusCode, input.ErrorMessage)
}

func (s *invocationService) GetInvocation(ctx context.Context, invocationID uuid.UUID) (*InvocationDetail, error) {
	invocation, err := s.invocationRepo.GetByID(ctx, invocationID)
	if err != nil {
		return nil, err
	}
	attempts, err := s.invocationAttemptRepo.ListByInvocationID(ctx, invocationID)
	if err != nil {
		return nil, err
	}
	return &InvocationDetail{
		Invocation: *invocation,
		Attempts:   attempts,
	}, nil
}
