package errors

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type ErrorResponse struct {
	Error      string                 `json:"error"`
	Message    string                 `json:"message"`
	RequestID  string                 `json:"request_id,omitempty"`
	StatusCode int                    `json:"status_code"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// ErrorHandler is a global error handler middleware
func ErrorHandler(c *fiber.Ctx, err error) error {
	requestID := c.Locals("request_id").(string)

	if fiberErr, ok := err.(*fiber.Error); ok {
		return c.Status(fiberErr.Code).JSON(ErrorResponse{
			Error:      http.StatusText(fiberErr.Code),
			Message:    fiberErr.Message,
			RequestID:  requestID,
			StatusCode: fiberErr.Code,
		})
	}

	return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
		Error:      "Internal Server Error",
		Message:    err.Error(),
		RequestID:  requestID,
		StatusCode: http.StatusInternalServerError,
	})
}

// BadRequest returns a 400 error
func BadRequest(c *fiber.Ctx, message string, details map[string]interface{}) error {
	requestID := c.Locals("request_id").(string)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Error:      "Bad Request",
		Message:    message,
		RequestID:  requestID,
		StatusCode: http.StatusBadRequest,
		Details:    details,
	})
}

// Unauthorized returns a 401 error
func Unauthorized(c *fiber.Ctx, message string) error {
	requestID := c.Locals("request_id").(string)
	return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
		Error:      "Unauthorized",
		Message:    message,
		RequestID:  requestID,
		StatusCode: http.StatusUnauthorized,
	})
}

// NotFound returns a 404 error
func NotFound(c *fiber.Ctx, message string) error {
	requestID := c.Locals("request_id").(string)
	return c.Status(http.StatusNotFound).JSON(ErrorResponse{
		Error:      "Not Found",
		Message:    message,
		RequestID:  requestID,
		StatusCode: http.StatusNotFound,
	})
}

// InternalError returns a 500 error
func InternalError(c *fiber.Ctx, message string) error {
	requestID := c.Locals("request_id").(string)
	return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
		Error:      "Internal Server Error",
		Message:    message,
		RequestID:  requestID,
		StatusCode: http.StatusInternalServerError,
	})
}
