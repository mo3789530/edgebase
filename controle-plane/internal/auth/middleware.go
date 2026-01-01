package auth

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	AuthorizationHeader = "Authorization"
	BearerScheme        = "Bearer"
	NodeIDContextKey    = "node_id"
)

// AuthMiddleware validates JWT tokens from Authorization header
func AuthMiddleware(authMgr *Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get(AuthorizationHeader)
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != BearerScheme {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}

		claims, err := authMgr.VerifyToken(parts[1])
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals(NodeIDContextKey, claims.NodeID)
		return c.Next()
	}
}

// OptionalAuthMiddleware validates JWT if present, but doesn't require it
func OptionalAuthMiddleware(authMgr *Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get(AuthorizationHeader)
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != BearerScheme {
			return c.Next()
		}

		claims, err := authMgr.VerifyToken(parts[1])
		if err == nil {
			c.Locals(NodeIDContextKey, claims.NodeID)
		}
		return c.Next()
	}
}

// GetNodeID extracts the node ID from context
func GetNodeID(c *fiber.Ctx) (uuid.UUID, error) {
	nodeID, ok := c.Locals(NodeIDContextKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.NewError(http.StatusUnauthorized, "node id not found in context")
	}
	return nodeID, nil
}
