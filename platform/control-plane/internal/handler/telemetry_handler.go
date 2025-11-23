package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type BatchResult struct {
	Success  bool `json:"success"`
	Inserted int  `json:"inserted"`
	Failed   int  `json:"failed"`
}

type CommandAck struct {
	Success   bool `json:"success"`
	Timestamp string `json:"timestamp"`
}

type DeviceRegistration struct {
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
	Location   string `json:"location"`
}

func (h *Handler) SyncTelemetry(c *fiber.Ctx) error {
	var batch []model.TelemetryData
	if err := c.BodyParser(&batch); err != nil {
		return c.Status(http.StatusBadRequest).JSON(BatchResult{
			Success:  false,
			Inserted: 0,
			Failed:   0,
		})
	}

	if len(batch) == 0 {
		return c.Status(http.StatusBadRequest).JSON(BatchResult{
			Success:  false,
			Inserted: 0,
			Failed:   0,
		})
	}

	inserted, err := h.telemetrySvc.SyncTelemetry(c.Context(), batch)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(BatchResult{
			Success:  false,
			Inserted: 0,
			Failed:   len(batch),
		})
	}

	return c.Status(http.StatusOK).JSON(BatchResult{
		Success:  true,
		Inserted: inserted,
		Failed:   0,
	})
}

func (h *Handler) GetCommands(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("device_id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON([]model.Command{})
	}

	commands, err := h.telemetrySvc.GetCommands(c.Context(), deviceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON([]model.Command{})
	}

	return c.Status(http.StatusOK).JSON(commands)
}

func (h *Handler) AckCommand(c *fiber.Ctx) error {
	commandID, err := uuid.Parse(c.Params("command_id"))
	if err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	var ack CommandAck
	if err := c.BodyParser(&ack); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	if err := h.telemetrySvc.AckCommand(c.Context(), commandID, ack.Success); err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	return c.SendStatus(http.StatusOK)
}

func (h *Handler) GetSyncStatus(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("device_id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(nil)
	}

	status, err := h.telemetrySvc.GetSyncStatus(c.Context(), deviceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(nil)
	}

	return c.Status(http.StatusOK).JSON(status)
}

func (h *Handler) RegisterDevice(c *fiber.Ctx) error {
	var reg DeviceRegistration
	if err := c.BodyParser(&reg); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	deviceID, err := h.telemetrySvc.RegisterDevice(c.Context(), reg.DeviceName, reg.DeviceType, reg.Location)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"device_id": deviceID})
}
