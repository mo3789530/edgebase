package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/pagination"
	"github.com/edgebase/platform/control-plane/internal/validator"
	"github.com/gofiber/fiber/v2"
)

type RegisterSchemaRequest struct {
	Version     int    `json:"version"`
	UpSQL       string `json:"up_sql"`
	DownSQL     string `json:"down_sql"`
	Description string `json:"description"`
}

func (h *Handler) RegisterSchema(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	var req RegisterSchemaRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("up_sql", req.UpSQL).MinLength("up_sql", req.UpSQL, 1)
	if !v.IsValid() {
		logger.Warn(requestID, "validation_failed", nil, v.ErrorMap())
		errs := make(map[string]interface{})
		for k, v := range v.ErrorMap() {
			errs[k] = v
		}
		return errors.BadRequest(c, "validation failed", errs)
	}

	if err := h.schemaSvc.RegisterSchema(c.Context(), req.Version, req.UpSQL, req.DownSQL, req.Description); err != nil {
		logger.Error(requestID, "register_schema_failed", err)
		return errors.InternalError(c, "failed to register schema")
	}

	logger.Info(requestID, "schema_registered", map[string]interface{}{
		"version": req.Version,
	})

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "schema registered"})
}

func (h *Handler) ListSchemas(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	params := pagination.ParseParams(c)

	schemas, err := h.schemaSvc.ListSchemas(c.Context())
	if err != nil {
		logger.Error(requestID, "list_schemas_failed", err)
		return errors.InternalError(c, "failed to list schemas")
	}

	total := int64(len(schemas))
	start := params.Offset
	end := params.Offset + params.Limit
	if end > int(total) {
		end = int(total)
	}

	var paginatedSchemas interface{}
	if start < int(total) {
		paginatedSchemas = schemas[start:end]
	} else {
		paginatedSchemas = []interface{}{}
	}

	return c.JSON(pagination.NewResponse(paginatedSchemas, total, params))
}
