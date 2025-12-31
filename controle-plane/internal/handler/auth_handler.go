package handler

import (
	"net/http"
	"strings"

	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	requestID := logger.GetRequestID(c)

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		logger.Warn(requestID, "missing_authorization_header", nil, nil)
		return errors.Unauthorized(c, "missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		logger.Warn(requestID, "invalid_authorization_header", nil, nil)
		return errors.Unauthorized(c, "invalid authorization header")
	}

	newToken, err := h.authMgr.RefreshToken(parts[1], h.tokenExpiry)
	if err != nil {
		logger.Warn(requestID, "token_refresh_failed", err, nil)
		return errors.Unauthorized(c, "failed to refresh token")
	}

	logger.Info(requestID, "token_refreshed", nil)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"token": newToken,
	})
}
