package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateVM(c *fiber.Ctx) error {
	nodeID, err := h.parseUUID(c, "node_id")
	if err != nil {
		return errors.BadRequest(c, "invalid node id", nil)
	}

	var req model.VM
	if err := c.BodyParser(&req); err != nil {
		return errors.BadRequest(c, "invalid request body", nil)
	}

	if req.Name == "" {
		return errors.BadRequest(c, "vm name is required", nil)
	}

	vm, err := h.vmSvc.CreateVM(c.Context(), nodeID, req)
	if err != nil {
		return errors.InternalError(c, "failed to create vm")
	}

	return c.Status(http.StatusCreated).JSON(vm)
}

func (h *Handler) ListVMs(c *fiber.Ctx) error {
	nodeID, err := h.parseUUID(c, "node_id")
	if err != nil {
		return errors.BadRequest(c, "invalid node id", nil)
	}

	vms, err := h.vmSvc.ListVMs(c.Context(), nodeID)
	if err != nil {
		return errors.InternalError(c, "failed to list vms")
	}

	return c.JSON(vms)
}

func (h *Handler) GetVM(c *fiber.Ctx) error {
	vmID, err := h.parseUUID(c, "vm_id")
	if err != nil {
		return errors.BadRequest(c, "invalid vm id", nil)
	}

	vm, err := h.vmSvc.GetVM(c.Context(), vmID)
	if err != nil {
		return errors.InternalError(c, "failed to get vm")
	}

	return c.JSON(vm)
}

func (h *Handler) StartVM(c *fiber.Ctx) error {
	vmID, err := h.parseUUID(c, "vm_id")
	if err != nil {
		return errors.BadRequest(c, "invalid vm id", nil)
	}

	if err := h.vmSvc.StartVM(c.Context(), vmID); err != nil {
		return errors.InternalError(c, "failed to start vm")
	}

	return c.JSON(fiber.Map{"status": "started"})
}

func (h *Handler) StopVM(c *fiber.Ctx) error {
	vmID, err := h.parseUUID(c, "vm_id")
	if err != nil {
		return errors.BadRequest(c, "invalid vm id", nil)
	}

	if err := h.vmSvc.StopVM(c.Context(), vmID); err != nil {
		return errors.InternalError(c, "failed to stop vm")
	}

	return c.JSON(fiber.Map{"status": "stopped"})
}

func (h *Handler) DeleteVM(c *fiber.Ctx) error {
	vmID, err := h.parseUUID(c, "vm_id")
	if err != nil {
		return errors.BadRequest(c, "invalid vm id", nil)
	}

	if err := h.vmSvc.DeleteVM(c.Context(), vmID); err != nil {
		return errors.InternalError(c, "failed to delete vm")
	}

	return c.SendStatus(http.StatusNoContent)
}
