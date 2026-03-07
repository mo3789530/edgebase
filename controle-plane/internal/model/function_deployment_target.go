package model

import (
	"time"

	"github.com/google/uuid"
)

type FunctionDeploymentTarget struct {
	ID                    uuid.UUID          `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	FunctionDefinitionID  uuid.UUID          `gorm:"type:uuid;not null;index;uniqueIndex:idx_function_target_cluster" json:"function_definition_id"`
	FunctionDefinition    FunctionDefinition `gorm:"foreignKey:FunctionDefinitionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"function_definition,omitempty"`
	ClusterID             uuid.UUID          `gorm:"type:uuid;not null;index;uniqueIndex:idx_function_target_cluster" json:"cluster_id"`
	Namespace             string             `gorm:"not null;default:'edge-functions'" json:"namespace"`
	DesiredRevisionID     uuid.UUID          `gorm:"type:uuid;not null;index" json:"desired_revision_id"`
	DesiredRevision       FunctionRevision   `gorm:"foreignKey:DesiredRevisionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"desired_revision,omitempty"`
	LastAppliedRevisionID *uuid.UUID         `gorm:"type:uuid;index" json:"last_applied_revision_id,omitempty"`
	Replicas              int32              `gorm:"not null;default:1" json:"replicas"`
	RolloutStrategy       string             `gorm:"not null;default:'rolling'" json:"rollout_strategy"`
	Enabled               bool               `gorm:"not null;default:true" json:"enabled"`
	Status                string             `gorm:"not null;default:'pending'" json:"status"`
	UpdatedAt             time.Time          `gorm:"not null;default:now()" json:"updated_at"`
	CreatedAt             time.Time          `gorm:"not null;default:now()" json:"created_at"`
}
