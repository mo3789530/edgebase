package model

import (
	"time"

	"github.com/google/uuid"
)

type Invocation struct {
	ID                   uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	RouteID              *uuid.UUID `gorm:"type:uuid;index" json:"route_id,omitempty"`
	FunctionDefinitionID uuid.UUID  `gorm:"type:uuid;not null;index" json:"function_definition_id"`
	TriggerType          string     `gorm:"not null" json:"trigger_type"`
	RequestID            string     `gorm:"not null;index" json:"request_id"`
	StartedAt            time.Time  `gorm:"not null;index" json:"started_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	FinalStatus          string     `gorm:"not null;default:'started'" json:"final_status"`
	ClientStatusCode     *int       `json:"client_status_code,omitempty"`
	CreatedAt            time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}
