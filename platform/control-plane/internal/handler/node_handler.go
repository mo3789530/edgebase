package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RegisterNodeRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

func (h *Handler) RegisterNode(c *fiber.Ctx) error {
	var req RegisterNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	node, token, err := h.nodeSvc.RegisterNode(c.Context(), req.Name, req.Region)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"node":  node,
		"token": token,
	})
}

func (h *Handler) Heartbeat(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}

	if err := h.nodeSvc.Heartbeat(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) GetSyncInfo(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}

	var currentState service.NodeState
	if err := c.BodyParser(&currentState); err != nil {
		currentState = service.NodeState{}
	}

	plan, err := h.syncSvc.GetSyncPlan(c.Context(), id, currentState)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(plan)
}

func (h *Handler) AckSync(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}

	var req struct {
		SyncID uuid.UUID          `json:"sync_id"`
		Result service.SyncResult `json:"result"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.syncSvc.AcknowledgeSync(c.Context(), id, req.SyncID, req.Result); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "acked"})
}
