package cors

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// Middleware returns a CORS middleware with default configuration
func Middleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Content-Type,Authorization,X-Request-ID",
		ExposeHeaders: "X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset",
		MaxAge:       3600,
	})
}

// CustomMiddleware returns a CORS middleware with custom configuration
func CustomMiddleware(allowOrigins string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: allowOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Content-Type,Authorization,X-Request-ID",
		ExposeHeaders: "X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset",
		MaxAge:       3600,
	})
}
