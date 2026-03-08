package model

import (
	"time"

	"github.com/google/uuid"
)

type ClusterStatus string

const (
	ClusterStatusOnline  ClusterStatus = "online"
	ClusterStatusOffline ClusterStatus = "offline"
	ClusterStatusSyncing ClusterStatus = "syncing"
)

type Cluster struct {
	ID              uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name            string            `gorm:"not null;uniqueIndex" json:"name"`
	Region          string            `gorm:"not null" json:"region"`
	Environment     string            `gorm:"not null" json:"environment"`
	Status          ClusterStatus     `gorm:"not null" json:"status"`
	APIEndpoint     string            `gorm:"not null" json:"api_endpoint"`
	Labels          map[string]string `gorm:"serializer:json" json:"labels"`
	AuthTokenHash   string            `gorm:"not null" json:"-"`
	LastHeartbeatAt *time.Time        `json:"last_heartbeat_at"`
	LastInventoryAt *time.Time        `json:"last_inventory_at"`
	CreatedAt       time.Time         `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"not null;default:now()" json:"updated_at"`
}

type ClusterNode struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ClusterID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"cluster_id"`
	NodeName         string     `gorm:"not null" json:"node_name"`
	Role             string     `gorm:"not null" json:"role"`
	InternalIP       string     `json:"internal_ip"`
	Status           string     `gorm:"not null" json:"status"`
	KubeletVersion   string     `json:"kubelet_version"`
	OSImage          string     `json:"os_image"`
	ContainerRuntime string     `json:"container_runtime"`
	LastSeenAt       *time.Time `json:"last_seen_at"`
	CreatedAt        time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

type ClusterSyncRecord struct {
	ID             uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ClusterID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"cluster_id"`
	SyncType       string     `gorm:"not null" json:"sync_type"`
	Status         string     `gorm:"not null" json:"status"`
	StartedAt      time.Time  `gorm:"not null;default:now()" json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	ErrorMessage   string     `json:"error_message"`
	ChangesSummary string     `json:"changes_summary"`
}
