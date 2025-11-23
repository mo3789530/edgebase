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
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name           string    `gorm:"not null;uniqueIndex:idx_name_version" json:"name"`
	Version        string    `gorm:"not null;uniqueIndex:idx_name_version" json:"version"`
	Hash           string    `json:"hash"`
	SizeBytes      int64     `json:"size_bytes"`
	MinioPath      string    `json:"minio_path"`
	Entrypoint     string    `json:"entrypoint"`
	Runtime        string    `json:"runtime"`
	MemoryPages    int32     `json:"memory_pages"`
	MaxExecutionMs int32     `json:"max_execution_ms"`
	CreatedAt      time.Time `gorm:"not null;default:now()" json:"created_at"`
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
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NodeID     uuid.UUID `gorm:"type:uuid;not null" json:"node_id"`
	FunctionID uuid.UUID `gorm:"type:uuid;not null" json:"function_id"`
	Status     string    `gorm:"not null;default:'pending'" json:"status"`
	CreatedAt  time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null;default:now()" json:"updated_at"`
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

type Device struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DeviceName   string     `gorm:"not null" json:"device_name"`
	DeviceType   string     `gorm:"not null" json:"device_type"`
	Location     *string    `json:"location"`
	Status       string     `gorm:"not null;default:'active'" json:"status"`
	RegisteredAt time.Time  `gorm:"not null;default:now()" json:"registered_at"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
}

type TelemetryData struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DeviceID  uuid.UUID  `gorm:"type:uuid;not null" json:"device_id"`
	SensorID  string     `gorm:"not null" json:"sensor_id"`
	Timestamp time.Time  `gorm:"not null" json:"timestamp"`
	DataType  string     `gorm:"not null" json:"data_type"`
	Value     float64    `gorm:"not null" json:"value"`
	Unit      *string    `json:"unit"`
	Metadata  *string    `gorm:"type:jsonb" json:"metadata"`
	Version   int        `gorm:"not null;default:1" json:"version"`
	SyncedAt  *time.Time `json:"synced_at"`
}

type Command struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DeviceID  uuid.UUID `gorm:"type:uuid;not null" json:"device_id"`
	Type      string    `gorm:"not null" json:"type"`
	Payload   string    `gorm:"type:jsonb;not null" json:"payload"`
	Status    string    `gorm:"not null;default:'pending'" json:"status"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	ExecutedAt *time.Time `json:"executed_at"`
}

type SyncStatus struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DeviceID             uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"device_id"`
	LastSyncAt           *time.Time `json:"last_sync_at"`
	LastSyncStatus       *string    `json:"last_sync_status"`
	PendingRecordsCount  int        `gorm:"default:0" json:"pending_records_count"`
	TotalSyncedRecords   int64      `gorm:"default:0" json:"total_synced_records"`
}
