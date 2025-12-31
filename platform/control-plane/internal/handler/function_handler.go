package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateFunction(c *fiber.Ctx) error {
	var req struct {
		Name           string `json:"name"`
		Entrypoint     string `json:"entrypoint"`
		Runtime        string `json:"runtime"`
		MemoryPages    int32  `json:"memory_pages"`
		MaxExecutionMs int32  `json:"max_execution_ms"`
	}
	if err := c.BodyParser(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, "invalid request")
	}

	fn, err := h.artifactSvc.CreateFunction(c.Context(), req.Name, req.Entrypoint, req.Runtime, req.MemoryPages, req.MaxExecutionMs)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusCreated).JSON(fn)
}

func (h *Handler) GetFunction(c *fiber.Ctx) error {
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return err
	}

	fn, err := h.artifactSvc.GetFunction(c.Context(), id)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "function not found")
	}

	return c.JSON(fn)
}

func (h *Handler) UploadArtifact(c *fiber.Ctx) error {
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return err
	}

	file, err := c.FormFile("file")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "file required")
	}

	f, err := file.Open()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "failed to open file")
	}
	defer f.Close()

	buffer := make([]byte, file.Size)
	_, err = f.Read(buffer)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "failed to read file")
	}

	fn, err := h.artifactSvc.UploadArtifact(c.Context(), id, buffer)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(fn)
}

func (h *Handler) DownloadFunction(c *fiber.Ctx) error {
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return err
	}

	url, err := h.artifactSvc.GetDownloadURL(c.Context(), id)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return c.Redirect(url)
}

func (h *Handler) DeleteFunction(c *fiber.Ctx) error {
	id, err := h.parseUUID(c, "id")
	if err != nil {
		return err
	}

	if err := h.artifactSvc.DeleteFunction(c.Context(), id); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}

func (h *Handler) DownloadArtifact(c *fiber.Ctx) error {
	idStr := c.Params("id")
	version := c.Params("version")

	data, err := h.artifactSvc.GetArtifactData(c.Context(), idStr, version)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "artifact not found")
	}

	c.Set("Content-Type", "application/octet-stream")
	return c.Send(data)
}

func (h *Handler) DeployFunction(c *fiber.Ctx) error {
	functionID, err := h.parseUUID(c, "function_id")
	if err != nil {
		return err
	}

	nodeID, err := h.parseUUID(c, "node_id")
	if err != nil {
		return err
	}

	if err := h.syncSvc.QueueDeployment(c.Context(), nodeID, functionID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"status": "queued"})
}
