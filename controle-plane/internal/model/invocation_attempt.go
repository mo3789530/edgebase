package model

import (
	"time"

	"github.com/google/uuid"
)

type InvocationAttempt struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InvocationID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"invocation_id"`
	ClusterID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"cluster_id"`
	KnativeService  string     `gorm:"not null" json:"knative_service"`
	KnativeRevision string     `json:"knative_revision"`
	PodName         string     `json:"pod_name"`
	AttemptNo       int        `gorm:"not null" json:"attempt_no"`
	StartedAt       time.Time  `gorm:"not null;index" json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	Status          string     `gorm:"not null;default:'started'" json:"status"`
	StatusCode      *int       `json:"status_code,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}
