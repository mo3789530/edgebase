package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CreateRoute(c *fiber.Ctx) error {
	var req struct {
		Host        string   `json:"host"`
		Path        string   `json:"path"`
		FunctionID  string   `json:"function_id"`
		Methods     []string `json:"methods"`
		Priority    int32    `json:"priority"`
		PopSelector *string  `json:"pop_selector"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	route, err := h.syncSvc.CreateRoute(c.Context(), req.Host, req.Path, req.FunctionID, req.Methods, req.Priority, req.PopSelector)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(route)
}

func (h *Handler) ListRoutes(c *fiber.Ctx) error {
	routes, err := h.syncSvc.ListRoutes(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(routes)
}
