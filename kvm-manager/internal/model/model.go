package model

import (
	"time"
)

type VMSpec struct {
	Name       string            `json:"name"`
	TemplateID string            `json:"template_id"`
	Resources  ResourceRequest   `json:"resources"`
	Network    NetworkConfig     `json:"network"`
	Storage    StorageConfig     `json:"storage"`
	Metadata   map[string]string `json:"metadata"`
}

type ResourceRequest struct {
	CPUCores int `json:"cpu_cores"`
	MemoryMB int `json:"memory_mb"`
	DiskGB   int `json:"disk_gb"`
}

type NetworkConfig struct {
	Mode       string   `json:"mode"` // "bridged", "nat", "isolated"
	Interfaces []string `json:"interfaces"`
	IPAddress  string   `json:"ip_address,omitempty"`
}

type StorageConfig struct {
	SizeGB int    `json:"size_gb"`
	Path   string `json:"path"`
}

type VM struct {
	ID        string            `json:"id"`
	NodeID    string            `json:"node_id"`
	Name      string            `json:"name"`
	Status    VMStatus          `json:"status"`
	Resources ResourceRequest   `json:"resources"`
	Network   NetworkConfig     `json:"network"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata"`
}

type VMStatus string

const (
	VMStatusCreating VMStatus = "creating"
	VMStatusStopped  VMStatus = "stopped"
	VMStatusRunning  VMStatus = "running"
	VMStatusPaused   VMStatus = "paused"
	VMStatusError    VMStatus = "error"
)

type VMTemplate struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	OSType       string            `json:"os_type"`
	ImagePath    string            `json:"image_path"`
	MinResources ResourceRequest   `json:"min_resources"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
}

type ResourceUsage struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   int     `json:"memory_mb"`
	StorageGB  int     `json:"storage_gb"`
}
