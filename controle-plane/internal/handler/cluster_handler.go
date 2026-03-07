package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ClusterHeartbeat is a compatibility endpoint for cluster-agent design docs.
// It delegates to the same node heartbeat flow.
func (h *Handler) ClusterHeartbeat(c *fiber.Ctx) error {
	return h.Heartbeat(c)
}

// ClusterGetSyncInfo is a compatibility endpoint for cluster-agent design docs.
// It delegates to the same sync planner.
func (h *Handler) ClusterGetSyncInfo(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid cluster id", nil)
	}

	plan, err := h.controllerSvc.GetClusterSyncPlan(c.Context(), id)
	if err != nil {
		logger.Error(requestID, "get_cluster_sync_plan_failed", err)
		return errors.InternalError(c, "failed to get sync plan")
	}

	return c.JSON(plan)
}

func (h *Handler) ClusterAckSync(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid cluster id", nil)
	}

	var req struct {
		SyncID uuid.UUID          `json:"sync_id"`
		Result service.SyncResult `json:"result"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	if err := h.controllerSvc.AcknowledgeClusterSync(c.Context(), id, req.SyncID, req.Result); err != nil {
		logger.Error(requestID, "acknowledge_cluster_sync_failed", err)
		return errors.InternalError(c, "failed to acknowledge sync")
	}

	return c.JSON(fiber.Map{"status": "acked"})
}

func (h *Handler) ClusterListGatewayRoutes(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid cluster id", nil)
	}

	routes, err := h.routeSvc.ListGatewayRoutes(c.Context(), id)
	if err != nil {
		logger.Error(requestID, "list_gateway_routes_failed", err)
		return errors.InternalError(c, "failed to list gateway routes")
	}

	return c.JSON(routes)
}

type ClusterInventoryRequest struct {
	ClusterID         uuid.UUID       `json:"cluster_id"`
	ObservedAt        string          `json:"observed_at"`
	KubernetesVersion string          `json:"kubernetes_version"`
	Nodes             json.RawMessage `json:"nodes"`
	Deployments       json.RawMessage `json:"deployments"`
	Services          json.RawMessage `json:"services"`
	Pods              json.RawMessage `json:"pods"`
}

// ClusterInventory receives current cluster inventory snapshots from cluster-agent.
// MVP behavior accepts payload and returns acknowledgement.
func (h *Handler) ClusterInventory(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid cluster id", nil)
	}

	var req ClusterInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	if req.ClusterID != uuid.Nil && req.ClusterID != id {
		return errors.BadRequest(c, "cluster_id mismatch", nil)
	}

	if h.inventorySvc != nil {
		input := service.ClusterInventoryInput{
			ClusterID:         req.ClusterID,
			KubernetesVersion: req.KubernetesVersion,
			Nodes:             req.Nodes,
			Deployments:       req.Deployments,
			Services:          req.Services,
			Pods:              req.Pods,
		}
		if req.ObservedAt != "" {
			if observedAt, parseErr := parseRFC3339(req.ObservedAt); parseErr == nil {
				input.ObservedAt = observedAt
			}
		}
		if err := h.inventorySvc.SaveSnapshot(c.Context(), id, input); err != nil {
			logger.Error(requestID, "save_cluster_inventory_failed", err)
			return errors.InternalError(c, "failed to save inventory")
		}
	}

	logger.Info(requestID, "cluster_inventory_received", map[string]interface{}{
		"cluster_id": id.String(),
	})

	return c.Status(http.StatusAccepted).JSON(fiber.Map{"status": "accepted"})
}

func parseRFC3339(v string) (time.Time, error) {
	return time.Parse(time.RFC3339, v)
}
