package logger

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDKey = "request_id"

// RequestIDMiddleware generates and injects request IDs
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Locals(RequestIDKey, requestID)
		c.Set(RequestIDHeader, requestID)
		return c.Next()
	}
}

// LoggingMiddleware logs all requests and responses
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Locals(RequestIDKey).(string)
		start := time.Now()

		Info(requestID, "request_start", map[string]interface{}{
			"method": c.Method(),
			"path":   c.Path(),
			"ip":     c.IP(),
		})

		err := c.Next()

		duration := time.Since(start).Milliseconds()
		Info(requestID, "request_end", map[string]interface{}{
			"method":     c.Method(),
			"path":       c.Path(),
			"status":     c.Response().StatusCode(),
			"duration_ms": duration,
		})

		return err
	}
}

// GetRequestID extracts request ID from context
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
