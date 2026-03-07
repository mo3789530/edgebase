package model

import (
	"time"

	"github.com/google/uuid"
)

type RouteDefinition struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Host                 string    `gorm:"not null;index:idx_route_host_path" json:"host"`
	Path                 string    `gorm:"not null;index:idx_route_host_path" json:"path"`
	Methods              string    `gorm:"type:jsonb;not null" json:"methods"`
	FunctionDefinitionID uuid.UUID `gorm:"type:uuid;not null;index" json:"function_definition_id"`
	TimeoutMs            int32     `gorm:"not null;default:3000" json:"timeout_ms"`
	RetryPolicy          string    `json:"retry_policy"`
	ClusterSelector      string    `json:"cluster_selector"`
	Enabled              bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt            time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt            time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

