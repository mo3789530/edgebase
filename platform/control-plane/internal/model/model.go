package model

import (
	"time"

	"github.com/google/uuid"
)

type NodeStatus string

const (
	NodeStatusOnline  NodeStatus = "online"
	NodeStatusOffline NodeStatus = "offline"
	NodeStatusSyncing NodeStatus = "syncing"
)

type Node struct {
	ID                   uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name                 string     `gorm:"not null" json:"name"`
	Region               string     `json:"region"`
	Status               NodeStatus `gorm:"not null" json:"status"`
	AuthTokenHash        string     `gorm:"not null" json:"-"`
	CurrentSchemaVersion int        `gorm:"default:0;not null" json:"current_schema_version"`
	LastHeartbeatAt      *time.Time `json:"last_heartbeat_at"`
	LastSyncAt           *time.Time `json:"last_sync_at"`
	CreatedAt            time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

type Function struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name      string    `gorm:"not null;uniqueIndex:idx_name_version" json:"name"`
	Version   string    `gorm:"not null;uniqueIndex:idx_name_version" json:"version"`
	Hash      string    `gorm:"not null" json:"hash"`
	SizeBytes int64     `gorm:"not null" json:"size_bytes"`
	MinioPath string    `gorm:"not null" json:"minio_path"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

type SchemaMigration struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Version     int       `gorm:"unique;not null" json:"version"`
	Description string    `json:"description"`
	UpSQL       string    `gorm:"not null" json:"up_sql"`
	DownSQL     string    `json:"down_sql"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

type NodeFunctionDeployment struct {
	NodeID     uuid.UUID `gorm:"primaryKey;type:uuid" json:"node_id"`
	FunctionID uuid.UUID `gorm:"primaryKey;type:uuid" json:"function_id"`
	DeployedAt time.Time `gorm:"not null;default:now()" json:"deployed_at"`
}

type SyncRecord struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NodeID           uuid.UUID  `gorm:"type:uuid;not null" json:"node_id"`
	SyncType         string     `gorm:"not null" json:"sync_type"`
	Status           string     `gorm:"not null" json:"status"`
	StartedAt        time.Time  `gorm:"not null;default:now()" json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	ErrorMessage     string     `json:"error_message"`
	FunctionsAdded   int        `gorm:"default:0" json:"functions_added"`
	FunctionsRemoved int        `gorm:"default:0" json:"functions_removed"`
	SchemasApplied   int        `gorm:"default:0" json:"schemas_applied"`
}
