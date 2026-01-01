package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/metrics"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if err := h.db.Exec("SELECT 1").Error; err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not_ready",
			"error":  "database connection failed",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ready",
	})
}

func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "alive",
	})
}

func (h *HealthHandler) Metrics(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(metrics.GetMetrics())
}
