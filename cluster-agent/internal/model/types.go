package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type HealthStatus string

const (
	HealthStatusHealthy     HealthStatus = "healthy"
	HealthStatusDegraded    HealthStatus = "degraded"
	HealthStatusUnreachable HealthStatus = "unreachable"
)

type AgentState string

const (
	AgentStateStarting    AgentState = "starting"
	AgentStateHealthy     AgentState = "healthy"
	AgentStateSyncing     AgentState = "syncing"
	AgentStateDegraded    AgentState = "degraded"
	AgentStateUnreachable AgentState = "unreachable"
)

type Heartbeat struct {
	AgentVersion     string       `json:"agent_version"`
	ClusterVersion   string       `json:"cluster_version,omitempty"`
	Health           HealthStatus `json:"health"`
	State            AgentState   `json:"state"`
	LastSyncSuccess  bool         `json:"last_sync_success"`
	ObservedAt       time.Time    `json:"observed_at"`
	KubernetesAccess bool         `json:"kubernetes_access"`
}

type ClusterInventory struct {
	ClusterID         uuid.UUID    `json:"cluster_id"`
	ObservedAt        time.Time    `json:"observed_at"`
	KubernetesVersion string       `json:"kubernetes_version,omitempty"`
	Nodes             []NodeInfo   `json:"nodes,omitempty"`
	Deployments       []Deployment `json:"deployments,omitempty"`
	Services          []Service    `json:"services,omitempty"`
	Pods              []PodInfo    `json:"pods,omitempty"`
}

type NodeInfo struct {
	Name       string `json:"name"`
	Role       string `json:"role,omitempty"`
	InternalIP string `json:"internal_ip,omitempty"`
	Status     string `json:"status"`
}

type Deployment struct {
	Namespace         string `json:"namespace"`
	Name              string `json:"name"`
	Image             string `json:"image,omitempty"`
	ReadyReplicas     int32  `json:"ready_replicas,omitempty"`
	AvailableReplicas int32  `json:"available_replicas,omitempty"`
}

type Service struct {
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Selector  map[string]string `json:"selector,omitempty"`
}

type PodInfo struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	NodeName  string `json:"node_name,omitempty"`
}

type NodeState struct {
	SchemaVersion int             `json:"schema_version"`
	Functions     []FunctionState `json:"functions"`
}

type FunctionState struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type SyncPlan struct {
	SyncID     uuid.UUID    `json:"sync_id"`
	Generation int64        `json:"generation,omitempty"`
	Actions    []SyncAction `json:"actions"`
}

type SyncAction struct {
	Type        string          `json:"type"`
	Order       int             `json:"order"`
	Payload     json.RawMessage `json:"payload"`
	Description string          `json:"description,omitempty"`
}

const (
	ActionApplyDeployment   = "APPLY_DEPLOYMENT"
	ActionApplyService      = "APPLY_SERVICE"
	ActionDeleteDeployment  = "DELETE_DEPLOYMENT"
	ActionDeleteService     = "DELETE_SERVICE"
	ActionRestartDeployment = "RESTART_DEPLOYMENT"
)

type SyncAck struct {
	SyncID  uuid.UUID         `json:"sync_id"`
	Success bool              `json:"success"`
	Results []SyncAckResource `json:"results"`
}

type SyncAckResource struct {
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}
