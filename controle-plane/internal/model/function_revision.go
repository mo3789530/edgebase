package model

import (
	"time"

	"github.com/google/uuid"
)

type FunctionRevision struct {
	ID                   uuid.UUID          `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	FunctionDefinitionID uuid.UUID          `gorm:"type:uuid;not null;index;uniqueIndex:idx_function_revision_version" json:"function_definition_id"`
	FunctionDefinition   FunctionDefinition `gorm:"foreignKey:FunctionDefinitionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"function_definition,omitempty"`
	Version              string             `gorm:"not null;uniqueIndex:idx_function_revision_version" json:"version"`
	Image                string             `gorm:"not null" json:"image"`
	ImageDigest          string             `gorm:"not null" json:"image_digest"`
	Command              string             `json:"command"`
	Args                 string             `json:"args"`
	Env                  string             `gorm:"type:jsonb" json:"env"`
	Port                 int32              `gorm:"not null;default:8080" json:"port"`
	HealthcheckPath      string             `gorm:"not null;default:'/health'" json:"healthcheck_path"`
	CreatedAt            time.Time          `gorm:"not null;default:now()" json:"created_at"`
}
