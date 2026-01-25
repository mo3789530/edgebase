package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VM struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	NodeID    uuid.UUID      `json:"node_id" gorm:"type:uuid;not null"`
	Name      string         `json:"name" gorm:"not null"`
	Status    string         `json:"status"` // creating, stopped, running, etc.
	CPUCores  int            `json:"cpu_cores"`
	MemoryMB  int            `json:"memory_mb"`
	DiskGB    int            `json:"disk_gb"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type VMTemplate struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string         `json:"name" gorm:"not null"`
	ImagePath string         `json:"image_path"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
