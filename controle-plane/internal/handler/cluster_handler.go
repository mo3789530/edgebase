package handler

import (
	"errors"
	"net/http"

	appErr "github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegisterClusterRequest struct {
	Name        string            `json:"name"`
	Region      string            `json:"region"`
	Environment string            `json:"environment"`
	APIEndpoint string            `json:"api_endpoint"`
	Labels      map[string]string `json:"labels"`
}

type ClusterInventoryNodeRequest struct {
	NodeName         string `json:"node_name"`
	Role             string `json:"role"`
	InternalIP       string `json:"internal_ip"`
	Status           string `json:"status"`
	KubeletVersion   string `json:"kubelet_version"`
	OSImage          string `json:"os_image"`
	ContainerRuntime string `json:"container_runtime"`
}

type ClusterInventoryRequest struct {
	Nodes []ClusterInventoryNodeRequest `json:"nodes"`
}

type ClusterSyncAckRequest struct {
	SyncID uuid.UUID                 `json:"sync_id"`
	Result service.ClusterSyncResult `json:"result"`
}

func (h *Handler) RegisterCluster(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	var req RegisterClusterRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return appErr.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("name", req.Name).MinLength("name", req.Name, 1).MaxLength("name", req.Name, 255)
	v.Required("region", req.Region).MinLength("region", req.Region, 1).MaxLength("region", req.Region, 255)
	v.Required("environment", req.Environment).MinLength("environment", req.Environment, 1).MaxLength("environment", req.Environment, 255)
	v.Required("api_endpoint", req.APIEndpoint).MinLength("api_endpoint", req.APIEndpoint, 1).MaxLength("api_endpoint", req.APIEndpoint, 1000)

	if !v.IsValid() {
		errs := make(map[string]interface{})
		for k, val := range v.ErrorMap() {
			errs[k] = val
		}
		logger.Warn(requestID, "validation_failed", nil, errs)
		return appErr.BadRequest(c, "validation failed", errs)
	}

	cluster, _, err := h.clusterSvc.RegisterCluster(c.Context(), service.RegisterClusterInput{
		Name:        req.Name,
		Region:      req.Region,
		Environment: req.Environment,
		APIEndpoint: req.APIEndpoint,
		Labels:      req.Labels,
	})
	if err != nil {
		logger.Error(requestID, "register_cluster_failed", err)
		return appErr.InternalError(c, "failed to register cluster")
	}

	token, err := h.authMgr.GenerateToken(cluster.ID, h.tokenExpiry)
	if err != nil {
		logger.Error(requestID, "cluster_token_generation_failed", err)
		return appErr.InternalError(c, "failed to generate token")
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"cluster": cluster,
		"token":   token,
	})
}

func (h *Handler) ListClusters(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	clusters, err := h.clusterSvc.ListClusters(c.Context())
	if err != nil {
		logger.Error(requestID, "list_clusters_failed", err)
		return appErr.InternalError(c, "failed to list clusters")
	}
	return c.JSON(fiber.Map{"clusters": clusters})
}

func (h *Handler) GetCluster(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return appErr.BadRequest(c, "invalid cluster id", nil)
	}

	cluster, err := h.clusterSvc.GetCluster(c.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return appErr.NotFound(c, "cluster not found")
		}
		logger.Error(requestID, "get_cluster_failed", err)
		return appErr.InternalError(c, "failed to get cluster")
	}
	return c.JSON(fiber.Map{"cluster": cluster})
}

func (h *Handler) ClusterHeartbeat(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return appErr.BadRequest(c, "invalid cluster id", nil)
	}

	if err := h.clusterSvc.Heartbeat(c.Context(), id); err != nil {
		logger.Error(requestID, "cluster_heartbeat_failed", err)
		return appErr.InternalError(c, "heartbeat failed")
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) UpdateClusterInventory(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return appErr.BadRequest(c, "invalid cluster id", nil)
	}

	var req ClusterInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return appErr.BadRequest(c, "invalid request body", nil)
	}

	nodes := make([]service.ClusterInventoryNodeInput, 0, len(req.Nodes))
	for _, node := range req.Nodes {
		nodes = append(nodes, service.ClusterInventoryNodeInput{
			NodeName:         node.NodeName,
			Role:             node.Role,
			InternalIP:       node.InternalIP,
			Status:           node.Status,
			KubeletVersion:   node.KubeletVersion,
			OSImage:          node.OSImage,
			ContainerRuntime: node.ContainerRuntime,
		})
	}

	if err := h.clusterInvSvc.UpdateInventory(c.Context(), id, service.ClusterInventoryInput{Nodes: nodes}); err != nil {
		logger.Error(requestID, "update_cluster_inventory_failed", err)
		return appErr.InternalError(c, "failed to update cluster inventory")
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) GetClusterSync(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return appErr.BadRequest(c, "invalid cluster id", nil)
	}

	var currentState service.ClusterState
	if err := c.BodyParser(&currentState); err != nil {
		currentState = service.ClusterState{}
	}

	plan, err := h.clusterSyncSvc.GetSyncPlan(c.Context(), id, currentState)
	if err != nil {
		logger.Error(requestID, "get_cluster_sync_plan_failed", err)
		return appErr.InternalError(c, "failed to get cluster sync plan")
	}

	return c.JSON(plan)
}

func (h *Handler) AckClusterSync(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return appErr.BadRequest(c, "invalid cluster id", nil)
	}

	var req ClusterSyncAckRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return appErr.BadRequest(c, "invalid request body", nil)
	}

	if err := h.clusterSyncSvc.AcknowledgeSync(c.Context(), id, req.SyncID, req.Result); err != nil {
		logger.Error(requestID, "ack_cluster_sync_failed", err)
		return appErr.InternalError(c, "failed to acknowledge cluster sync")
	}
	return c.JSON(fiber.Map{"status": "acked"})
}
