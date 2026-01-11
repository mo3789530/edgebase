package handler

import (
	"net/http"

	"time"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/timeseries"
	"github.com/edgebase/platform/control-plane/internal/validator"
	"github.com/gofiber/fiber/v2"
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
		Name           string `json:"name"`
		Entrypoint     string `json:"entrypoint"`
		Runtime        string `json:"runtime"`
		MemoryPages    int32  `json:"memory_pages"`
		MaxExecutionMs int32  `json:"max_execution_ms"`
	}
	if err = c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("name", req.Name).MinLength("name", req.Name, 1).MaxLength("name", req.Name, 255)
	v.Required("entrypoint", req.Entrypoint).MinLength("entrypoint", req.Entrypoint, 1)
	v.Required("runtime", req.Runtime).MinLength("runtime", req.Runtime, 1)

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

	fn, err := h.artifactSvc.CreateFunction(c.Context(), req.Name, req.Entrypoint, req.Runtime, req.MemoryPages, req.MaxExecutionMs)
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

	fn, err := h.artifactSvc.GetFunction(c.Context(), id)
	if err != nil {
		logger.Warn(requestID, "function_not_found", err, nil)
		return errors.NotFound(c, "function not found")
	}

	return c.JSON(fn)
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
