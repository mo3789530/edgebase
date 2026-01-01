package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RegisterNodeRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

func (h *Handler) RegisterNode(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	var req RegisterNodeRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	// Validate input
	v := validator.New()
	v.Required("name", req.Name).MinLength("name", req.Name, 1).MaxLength("name", req.Name, 255)
	v.Required("region", req.Region).MinLength("region", req.Region, 1).MaxLength("region", req.Region, 255)

	if !v.IsValid() {
		logger.Warn(requestID, "validation_failed", nil, v.ErrorMap())
		errs := make(map[string]interface{})
		for k, v := range v.ErrorMap() {
			errs[k] = v
		}
		return errors.BadRequest(c, "validation failed", errs)
	}

	node, _, err := h.nodeSvc.RegisterNode(c.Context(), req.Name, req.Region)
	if err != nil {
		logger.Error(requestID, "register_node_failed", err)
		return errors.InternalError(c, "failed to register node")
	}

	token, err := h.authMgr.GenerateToken(node.ID, h.tokenExpiry)
	if err != nil {
		logger.Error(requestID, "token_generation_failed", err)
		return errors.InternalError(c, "failed to generate token")
	}

	logger.Info(requestID, "node_registered", map[string]interface{}{
		"node_id": node.ID,
		"name":    node.Name,
	})

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"node":  node,
		"token": token,
	})
}

func (h *Handler) Heartbeat(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid node id", nil)
	}

	if err := h.nodeSvc.Heartbeat(c.Context(), id); err != nil {
		logger.Error(requestID, "heartbeat_failed", err)
		return errors.InternalError(c, "heartbeat failed")
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) GetSyncInfo(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid node id", nil)
	}

	var currentState service.NodeState
	if err := c.BodyParser(&currentState); err != nil {
		currentState = service.NodeState{}
	}

	plan, err := h.syncSvc.GetSyncPlan(c.Context(), id, currentState)
	if err != nil {
		logger.Error(requestID, "get_sync_plan_failed", err)
		return errors.InternalError(c, "failed to get sync plan")
	}

	return c.JSON(plan)
}

func (h *Handler) AckSync(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid node id", nil)
	}

	var req struct {
		SyncID uuid.UUID          `json:"sync_id"`
		Result service.SyncResult `json:"result"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	if err := h.syncSvc.AcknowledgeSync(c.Context(), id, req.SyncID, req.Result); err != nil {
		logger.Error(requestID, "acknowledge_sync_failed", err)
		return errors.InternalError(c, "failed to acknowledge sync")
	}

	return c.JSON(fiber.Map{"status": "acked"})
}
