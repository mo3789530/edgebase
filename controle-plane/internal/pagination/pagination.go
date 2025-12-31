package pagination

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Params struct {
	Limit  int
	Offset int
}

type Response struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	Page       int         `json:"page"`
	TotalPages int         `json:"total_pages"`
}

// ParseParams extracts pagination parameters from request
func ParseParams(c *fiber.Ctx) Params {
	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return Params{Limit: limit, Offset: offset}
}

// Paginate applies pagination to a query
func Paginate(params Params) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(params.Offset).Limit(params.Limit)
	}
}

// NewResponse creates a paginated response
func NewResponse(data interface{}, total int64, params Params) Response {
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	page := (params.Offset / params.Limit) + 1

	return Response{
		Data:       data,
		Total:      total,
		Limit:      params.Limit,
		Offset:     params.Offset,
		Page:       page,
		TotalPages: totalPages,
	}
}
