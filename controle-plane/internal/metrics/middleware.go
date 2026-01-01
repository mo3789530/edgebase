package metrics

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Middleware records request metrics
func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		IncrementActiveConnections()
		defer DecrementActiveConnections()

		start := time.Now()
		err := c.Next()
		latency := time.Since(start).Milliseconds()

		isError := c.Response().StatusCode() >= 400
		RecordRequest(c.Method(), c.Path(), latency, isError)

		return err
	}
}
