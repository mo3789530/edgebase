package model

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NodeID    uuid.UUID `gorm:"type:uuid;not null;index" json:"node_id"`
	Action    string    `gorm:"not null;index" json:"action"`
	Resource  string    `gorm:"not null" json:"resource"`
	Details   string    `gorm:"type:jsonb" json:"details"`
	Status    string    `gorm:"not null" json:"status"`
	CreatedAt time.Time `gorm:"not null;default:now();index" json:"created_at"`
}
