package handler

import (
	"net/http"

	"time"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/timeseries"
	"github.com/edgebase/platform/control-plane/internal/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h *Handler) CreateFunction(c *fiber.Ctx) (err error) {
	requestID := logger.GetRequestID(c)
	start := time.Now()

	defer func() {
		if h.metricCollector != nil {
			status := timeseries.StatusSuccess
			if err != nil {
				status = timeseries.StatusFailure
			}
			// Use generic ID for API metrics
			_ = h.metricCollector.RecordExecutionEnd(c.Context(), "api_create_function", requestID, time.Since(start), status, err)
		}
	}()

	var req struct {
		Name                  string `json:"name"`
		Description           string `json:"description"`
		RuntimeKind           string `json:"runtime_kind"`
		DefaultTimeoutSeconds int32  `json:"default_timeout_seconds"`
		DefaultMemoryMB       int32  `json:"default_memory_mb"`
		DefaultCPUMillis      int32  `json:"default_cpu_millis"`
	}
	if err = c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("name", req.Name).MinLength("name", req.Name, 1).MaxLength("name", req.Name, 255)

	if !v.IsValid() {
		logger.Warn(requestID, "validation_failed", nil, v.ErrorMap())
		errs := make(map[string]interface{})
		for k, v := range v.ErrorMap() {
			errs[k] = v
		}
		// Capture error from helper
		err = errors.BadRequest(c, "validation failed", errs)
		return err
	}

	fn, err := h.functionSvc.CreateDefinition(
		c.Context(),
		req.Name,
		req.Description,
		req.RuntimeKind,
		req.DefaultTimeoutSeconds,
		req.DefaultMemoryMB,
		req.DefaultCPUMillis,
	)
	if err != nil {
		logger.Error(requestID, "create_function_failed", err)
		return errors.InternalError(c, "failed to create function")
	}

	logger.Info(requestID, "function_created", map[string]interface{}{
		"function_id": fn.ID,
		"name":        fn.Name,
	})

	return c.Status(http.StatusCreated).JSON(fn)
}

func (h *Handler) GetFunction(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid function id", nil)
	}

	fn, err := h.functionSvc.GetDefinition(c.Context(), id)
	if err != nil {
		logger.Warn(requestID, "function_not_found", err, nil)
		return errors.NotFound(c, "function not found")
	}

	return c.JSON(fn)
}

func (h *Handler) ListFunctions(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)

	functions, err := h.functionSvc.ListDefinitions(c.Context())
	if err != nil {
		logger.Error(requestID, "list_functions_failed", err)
		return errors.InternalError(c, "failed to list functions")
	}

	return c.JSON(functions)
}

func (h *Handler) CreateFunctionRevision(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	functionDefinitionID, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid function id", nil)
	}

	var req struct {
		Version         string `json:"version"`
		Image           string `json:"image"`
		ImageDigest     string `json:"image_digest"`
		Command         string `json:"command"`
		Args            string `json:"args"`
		Env             string `json:"env"`
		Port            int32  `json:"port"`
		HealthcheckPath string `json:"healthcheck_path"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("version", req.Version).MinLength("version", req.Version, 1)
	v.Required("image", req.Image).MinLength("image", req.Image, 1)
	v.Required("image_digest", req.ImageDigest).MinLength("image_digest", req.ImageDigest, 1)

	if !v.IsValid() {
		logger.Warn(requestID, "validation_failed", nil, v.ErrorMap())
		errs := make(map[string]interface{})
		for k, v := range v.ErrorMap() {
			errs[k] = v
		}
		return errors.BadRequest(c, "validation failed", errs)
	}

	revision, err := h.functionSvc.CreateRevision(c.Context(), functionDefinitionID, service.CreateFunctionRevisionInput{
		Version:         req.Version,
		Image:           req.Image,
		ImageDigest:     req.ImageDigest,
		Command:         req.Command,
		Args:            req.Args,
		Env:             req.Env,
		Port:            req.Port,
		HealthcheckPath: req.HealthcheckPath,
	})
	if err != nil {
		logger.Error(requestID, "create_function_revision_failed", err)
		return errors.InternalError(c, "failed to create function revision")
	}

	logger.Info(requestID, "function_revision_created", map[string]interface{}{
		"function_definition_id": functionDefinitionID,
		"revision_id":            revision.ID,
		"version":                revision.Version,
	})

	return c.Status(http.StatusCreated).JSON(revision)
}

func (h *Handler) CreateFunctionDeploymentTargets(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	functionDefinitionID, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid function id", nil)
	}

	var req struct {
		ClusterIDs        []string `json:"cluster_ids"`
		Namespace         string   `json:"namespace"`
		DesiredRevisionID string   `json:"desired_revision_id"`
		Replicas          int32    `json:"replicas"`
		RolloutStrategy   string   `json:"rollout_strategy"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("desired_revision_id", req.DesiredRevisionID).MinLength("desired_revision_id", req.DesiredRevisionID, 1)
	if len(req.ClusterIDs) == 0 {
		v.Required("cluster_ids", "")
	}
	if !v.IsValid() {
		logger.Warn(requestID, "validation_failed", nil, v.ErrorMap())
		errs := make(map[string]interface{})
		for k, v := range v.ErrorMap() {
			errs[k] = v
		}
		return errors.BadRequest(c, "validation failed", errs)
	}

	revisionID, err := uuid.Parse(req.DesiredRevisionID)
	if err != nil {
		return errors.BadRequest(c, "invalid desired_revision_id", nil)
	}

	clusterIDs := make([]uuid.UUID, 0, len(req.ClusterIDs))
	for _, rawID := range req.ClusterIDs {
		clusterID, parseErr := uuid.Parse(rawID)
		if parseErr != nil {
			return errors.BadRequest(c, "invalid cluster id", nil)
		}
		clusterIDs = append(clusterIDs, clusterID)
	}

	targets, err := h.deploymentSvc.CreateTargets(c.Context(), functionDefinitionID, service.CreateDeploymentTargetsInput{
		ClusterIDs:        clusterIDs,
		Namespace:         req.Namespace,
		DesiredRevisionID: revisionID,
		Replicas:          req.Replicas,
		RolloutStrategy:   req.RolloutStrategy,
	})
	if err != nil {
		logger.Error(requestID, "create_function_deployments_failed", err)
		return errors.InternalError(c, "failed to create function deployments")
	}

	return c.Status(http.StatusCreated).JSON(targets)
}

func (h *Handler) UploadArtifact(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid function id", nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		logger.Warn(requestID, "file_missing", err, nil)
		return errors.BadRequest(c, "file required", nil)
	}

	f, err := file.Open()
	if err != nil {
		logger.Error(requestID, "file_open_failed", err)
		return errors.InternalError(c, "failed to open file")
	}
	defer f.Close()

	buffer := make([]byte, file.Size)
	_, err = f.Read(buffer)
	if err != nil {
		logger.Error(requestID, "file_read_failed", err)
		return errors.InternalError(c, "failed to read file")
	}

	fn, err := h.artifactSvc.UploadArtifact(c.Context(), id, buffer)
	if err != nil {
		logger.Error(requestID, "upload_artifact_failed", err)
		return errors.InternalError(c, "failed to upload artifact")
	}

	logger.Info(requestID, "artifact_uploaded", map[string]interface{}{
		"function_id": id,
		"size":        file.Size,
	})

	return c.Status(http.StatusOK).JSON(fn)
}

func (h *Handler) DownloadFunction(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid function id", nil)
	}

	url, err := h.artifactSvc.GetDownloadURL(c.Context(), id)
	if err != nil {
		logger.Error(requestID, "get_download_url_failed", err)
		return errors.InternalError(c, "failed to get download URL")
	}

	return c.Redirect(url)
}

func (h *Handler) DeleteFunction(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return errors.BadRequest(c, "invalid function id", nil)
	}

	if err := h.artifactSvc.DeleteFunction(c.Context(), id); err != nil {
		logger.Error(requestID, "delete_function_failed", err)
		return errors.InternalError(c, "failed to delete function")
	}

	logger.Info(requestID, "function_deleted", map[string]interface{}{
		"function_id": id,
	})

	return c.SendStatus(http.StatusNoContent)
}

func (h *Handler) DownloadArtifact(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	idStr := c.Params("id")
	version := c.Params("version")

	data, err := h.artifactSvc.GetArtifactData(c.Context(), idStr, version)
	if err != nil {
		logger.Warn(requestID, "artifact_not_found", err, nil)
		return errors.NotFound(c, "artifact not found")
	}

	c.Set("Content-Type", "application/octet-stream")
	return c.Send(data)
}

func (h *Handler) DeployFunction(c *fiber.Ctx) (err error) {
	requestID := logger.GetRequestID(c)
	start := time.Now()

	defer func() {
		if h.metricCollector != nil {
			status := timeseries.StatusSuccess
			if err != nil {
				status = timeseries.StatusFailure
			}
			_ = h.metricCollector.RecordExecutionEnd(c.Context(), "api_deploy_function", requestID, time.Since(start), status, err)
		}
	}()

	functionID, err := h.parseUUID(c, "function_id")
	if err != nil {
		return errors.BadRequest(c, "invalid function id", nil)
	}

	nodeID, err := h.parseUUID(c, "node_id")
	if err != nil {
		return errors.BadRequest(c, "invalid node id", nil)
	}

	if err = h.syncSvc.QueueDeployment(c.Context(), nodeID, functionID); err != nil {
		logger.Error(requestID, "queue_deployment_failed", err)
		return errors.InternalError(c, "failed to queue deployment")
	}

	logger.Info(requestID, "deployment_queued", map[string]interface{}{
		"function_id": functionID,
		"node_id":     nodeID,
	})

	return c.JSON(fiber.Map{"status": "queued"})
}
