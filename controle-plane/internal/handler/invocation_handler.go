package handler

import (
	"net/http"
	"time"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h *Handler) StartInvocation(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	var req struct {
		RouteID              *string `json:"route_id"`
		FunctionDefinitionID string  `json:"function_definition_id"`
		TriggerType          string  `json:"trigger_type"`
		RequestID            string  `json:"request_id"`
		StartedAt            string  `json:"started_at"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("function_definition_id", req.FunctionDefinitionID)
	v.Required("trigger_type", req.TriggerType)
	v.Required("request_id", req.RequestID)
	if !v.IsValid() {
		return errors.BadRequest(c, "validation failed", asInterfaceMap(v.ErrorMap()))
	}

	functionDefinitionID, err := uuid.Parse(req.FunctionDefinitionID)
	if err != nil {
		return errors.BadRequest(c, "invalid function_definition_id", nil)
	}

	var routeID *uuid.UUID
	if req.RouteID != nil && *req.RouteID != "" {
		parsed, err := uuid.Parse(*req.RouteID)
		if err != nil {
			return errors.BadRequest(c, "invalid route_id", nil)
		}
		routeID = &parsed
	}

	startedAt, err := parseOptionalTime(req.StartedAt)
	if err != nil {
		return errors.BadRequest(c, "invalid started_at", nil)
	}

	invocation, err := h.invocationSvc.StartInvocation(c.Context(), service.StartInvocationInput{
		RouteID:              routeID,
		FunctionDefinitionID: functionDefinitionID,
		TriggerType:          req.TriggerType,
		RequestID:            req.RequestID,
		StartedAt:            startedAt,
	})
	if err != nil {
		logger.Error(requestID, "start_invocation_failed", err)
		return errors.InternalError(c, "failed to start invocation")
	}

	return c.Status(http.StatusCreated).JSON(invocation)
}

func (h *Handler) CompleteInvocation(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	invocationID, err := h.parseUUID(c, "id")
	if err != nil {
		return nil
	}

	var req struct {
		CompletedAt      string `json:"completed_at"`
		FinalStatus      string `json:"final_status"`
		ClientStatusCode *int   `json:"client_status_code"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	completedAt, err := parseOptionalTime(req.CompletedAt)
	if err != nil {
		return errors.BadRequest(c, "invalid completed_at", nil)
	}

	if err := h.invocationSvc.CompleteInvocation(c.Context(), invocationID, service.CompleteInvocationInput{
		CompletedAt:      completedAt,
		FinalStatus:      req.FinalStatus,
		ClientStatusCode: req.ClientStatusCode,
	}); err != nil {
		logger.Error(requestID, "complete_invocation_failed", err)
		return errors.InternalError(c, "failed to complete invocation")
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) RecordInvocationAttempt(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	invocationID, err := h.parseUUID(c, "id")
	if err != nil {
		return nil
	}

	var req struct {
		ClusterID       string `json:"cluster_id"`
		KnativeService  string `json:"knative_service"`
		KnativeRevision string `json:"knative_revision"`
		PodName         string `json:"pod_name"`
		AttemptNo       int    `json:"attempt_no"`
		StartedAt       string `json:"started_at"`
		CompletedAt     string `json:"completed_at"`
		Status          string `json:"status"`
		StatusCode      *int   `json:"status_code"`
		ErrorMessage    string `json:"error_message"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	clusterID, err := uuid.Parse(req.ClusterID)
	if err != nil {
		return errors.BadRequest(c, "invalid cluster_id", nil)
	}

	startedAt, err := parseOptionalTime(req.StartedAt)
	if err != nil {
		return errors.BadRequest(c, "invalid started_at", nil)
	}

	var completedAt *time.Time
	if req.CompletedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.CompletedAt)
		if err != nil {
			return errors.BadRequest(c, "invalid completed_at", nil)
		}
		utc := parsed.UTC()
		completedAt = &utc
	}

	attempt, err := h.invocationSvc.RecordAttempt(c.Context(), invocationID, service.RecordAttemptInput{
		ClusterID:       clusterID,
		KnativeService:  req.KnativeService,
		KnativeRevision: req.KnativeRevision,
		PodName:         req.PodName,
		AttemptNo:       req.AttemptNo,
		StartedAt:       startedAt,
		CompletedAt:     completedAt,
		Status:          req.Status,
		StatusCode:      req.StatusCode,
		ErrorMessage:    req.ErrorMessage,
	})
	if err != nil {
		logger.Error(requestID, "record_invocation_attempt_failed", err)
		return errors.InternalError(c, "failed to record invocation attempt")
	}

	return c.Status(http.StatusCreated).JSON(attempt)
}

func (h *Handler) CompleteInvocationAttempt(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	attemptID, err := h.parseUUID(c, "attempt_id")
	if err != nil {
		return nil
	}

	var req struct {
		CompletedAt  string `json:"completed_at"`
		Status       string `json:"status"`
		StatusCode   *int   `json:"status_code"`
		ErrorMessage string `json:"error_message"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	completedAt, err := parseOptionalTime(req.CompletedAt)
	if err != nil {
		return errors.BadRequest(c, "invalid completed_at", nil)
	}

	if err := h.invocationSvc.CompleteAttempt(c.Context(), attemptID, service.CompleteAttemptInput{
		CompletedAt:  completedAt,
		Status:       req.Status,
		StatusCode:   req.StatusCode,
		ErrorMessage: req.ErrorMessage,
	}); err != nil {
		logger.Error(requestID, "complete_invocation_attempt_failed", err)
		return errors.InternalError(c, "failed to complete invocation attempt")
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) GetInvocation(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	invocationID, err := h.parseUUID(c, "id")
	if err != nil {
		return nil
	}

	detail, err := h.invocationSvc.GetInvocation(c.Context(), invocationID)
	if err != nil {
		logger.Error(requestID, "get_invocation_failed", err)
		return errors.InternalError(c, "failed to get invocation")
	}

	return c.JSON(detail)
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func asInterfaceMap(values map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
