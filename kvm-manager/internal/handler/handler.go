package handler

import (
	"net/http"

	"github.com/edgebase/platform/kvm-manager/internal/model"
	"github.com/edgebase/platform/kvm-manager/internal/service"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	vmManager service.VMManager
}

func NewHandler(vmManager service.VMManager) *Handler {
	return &Handler{
		vmManager: vmManager,
	}
}

func (h *Handler) CreateVM(c *fiber.Ctx) error {
	var spec model.VMSpec
	if err := c.BodyParser(&spec); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	vm, err := h.vmManager.CreateVM(c.Context(), spec)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(vm)
}

func (h *Handler) StartVM(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.vmManager.StartVM(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "started"})
}

func (h *Handler) StopVM(c *fiber.Ctx) error {
	id := c.Params("id")
	force := c.Query("force") == "true"
	if err := h.vmManager.StopVM(c.Context(), id, force); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "stopped"})
}

func (h *Handler) DeleteVM(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.vmManager.DeleteVM(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(http.StatusNoContent)
}

func (h *Handler) GetVM(c *fiber.Ctx) error {
	id := c.Params("id")
	vm, err := h.vmManager.GetVM(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "vm not found"})
	}
	return c.JSON(vm)
}

func (h *Handler) ListVMs(c *fiber.Ctx) error {
	vms, err := h.vmManager.ListVMs(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(vms)
}
