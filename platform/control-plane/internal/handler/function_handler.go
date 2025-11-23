package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	fn, err := h.artifactSvc.CreateFunction(c.Context(), req.Name, req.Entrypoint, req.Runtime, req.MemoryPages, req.MaxExecutionMs)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fn)
}

func (h *Handler) GetFunction(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	fn, err := h.artifactSvc.GetFunction(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "function not found"})
	}

	return c.JSON(fn)
}

func (h *Handler) UploadArtifact(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file required"})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer f.Close()

	buffer := make([]byte, file.Size)
	_, err = f.Read(buffer)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}

	fn, err := h.artifactSvc.UploadArtifact(c.Context(), id, buffer)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(fn)
}

func (h *Handler) DownloadFunction(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	url, err := h.artifactSvc.GetDownloadURL(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Redirect(url)
}

func (h *Handler) DeleteFunction(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	if err := h.artifactSvc.DeleteFunction(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(http.StatusNoContent)
}

func (h *Handler) DownloadArtifact(c *fiber.Ctx) error {
	idStr := c.Params("id")
	version := c.Params("version")

	data, err := h.artifactSvc.GetArtifactData(c.Context(), idStr, version)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "artifact not found"})
	}

	c.Set("Content-Type", "application/octet-stream")
	return c.Send(data)
}

func (h *Handler) DeployFunction(c *fiber.Ctx) error {
	functionIDStr := c.Params("function_id")
	nodeIDStr := c.Params("node_id")

	functionID, err := uuid.Parse(functionIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}

	if err := h.syncSvc.QueueDeployment(c.Context(), nodeID, functionID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "queued"})
}
