package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Middleware returns a rate limiting middleware
func Middleware(limiter *Limiter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.IP()
		remaining := limiter.GetRemaining(key)

		c.Set("X-RateLimit-Limit", strconv.Itoa(limiter.limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limiter.window).Unix(), 10))

		if !limiter.Allow(key) {
			return c.Status(http.StatusTooManyRequests).JSON(fiber.Map{
				"error": "rate limit exceeded",
			})
		}

		return c.Next()
	}
}
