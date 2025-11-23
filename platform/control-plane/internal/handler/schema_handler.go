package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type RegisterSchemaRequest struct {
	Version     int    `json:"version"`
	UpSQL       string `json:"up_sql"`
	DownSQL     string `json:"down_sql"`
	Description string `json:"description"`
}

func (h *Handler) RegisterSchema(c *fiber.Ctx) error {
	var req RegisterSchemaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.schemaSvc.RegisterSchema(c.Context(), req.Version, req.UpSQL, req.DownSQL, req.Description); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "schema registered"})
}

func (h *Handler) ListSchemas(c *fiber.Ctx) error {
	schemas, err := h.schemaSvc.ListSchemas(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(schemas)
}
