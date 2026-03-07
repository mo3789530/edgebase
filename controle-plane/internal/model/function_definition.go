package model

import (
	"time"

	"github.com/google/uuid"
)

type FunctionDefinition struct {
	ID                    uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name                  string    `gorm:"not null;uniqueIndex" json:"name"`
	Description           string    `json:"description"`
	RuntimeKind           string    `gorm:"not null;default:'container'" json:"runtime_kind"`
	DefaultTimeoutSeconds int32     `gorm:"not null;default:3" json:"default_timeout_seconds"`
	DefaultMemoryMB       int32     `gorm:"not null;default:128" json:"default_memory_mb"`
	DefaultCPUMillis      int32     `gorm:"not null;default:250" json:"default_cpu_millis"`
	CreatedAt             time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt             time.Time `gorm:"not null;default:now()" json:"updated_at"`
}
