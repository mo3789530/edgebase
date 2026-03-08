package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/pagination"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h *Handler) CreateRoute(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	var req struct {
		Host                 string   `json:"host"`
		Path                 string   `json:"path"`
		FunctionDefinitionID string   `json:"function_definition_id"`
		Methods              []string `json:"methods"`
		TimeoutMs            int32    `json:"timeout_ms"`
		RetryPolicy          string   `json:"retry_policy"`
		ClusterSelector      string   `json:"cluster_selector"`
	}
	if err := c.BodyParser(&req); err != nil {
		logger.Warn(requestID, "invalid_request_body", err, nil)
		return errors.BadRequest(c, "invalid request body", nil)
	}

	v := validator.New()
	v.Required("host", req.Host).MinLength("host", req.Host, 1)
	v.Required("path", req.Path).MinLength("path", req.Path, 1)
	v.Required("function_definition_id", req.FunctionDefinitionID).MinLength("function_definition_id", req.FunctionDefinitionID, 1)
	if !v.IsValid() {
		logger.Warn(requestID, "validation_failed", nil, v.ErrorMap())
		errs := make(map[string]interface{})
		for k, v := range v.ErrorMap() {
			errs[k] = v
		}
		return errors.BadRequest(c, "validation failed", errs)
	}

	functionDefinitionID, err := uuid.Parse(req.FunctionDefinitionID)
	if err != nil {
		return errors.BadRequest(c, "invalid function_definition_id", nil)
	}

	route, err := h.routeSvc.CreateRoute(c.Context(), service.CreateRouteInput{
		Host:                 req.Host,
		Path:                 req.Path,
		FunctionDefinitionID: functionDefinitionID,
		Methods:              req.Methods,
		TimeoutMs:            req.TimeoutMs,
		RetryPolicy:          req.RetryPolicy,
		ClusterSelector:      req.ClusterSelector,
	})
	if err != nil {
		logger.Error(requestID, "create_route_failed", err)
		return errors.InternalError(c, "failed to create route")
	}

	logger.Info(requestID, "route_created", map[string]interface{}{
		"host": req.Host,
		"path": req.Path,
	})

	return c.Status(http.StatusCreated).JSON(route)
}

func (h *Handler) ListRoutes(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)
	params := pagination.ParseParams(c)

	routesList, err := h.routeSvc.ListRoutes(c.Context())
	if err != nil {
		logger.Error(requestID, "list_routes_failed", err)
		return errors.InternalError(c, "failed to list routes")
	}

	routes := make([]interface{}, 0, len(routesList))
	for _, route := range routesList {
		routes = append(routes, route)
	}
	total := int64(len(routes))
	start := params.Offset
	end := params.Offset + params.Limit
	if end > int(total) {
		end = int(total)
	}

	var paginatedRoutes interface{}
	if start < int(total) {
		paginatedRoutes = routes[start:end]
	} else {
		paginatedRoutes = []interface{}{}
	}

	return c.JSON(pagination.NewResponse(paginatedRoutes, total, params))
}
