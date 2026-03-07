package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	ControlPlaneBaseURL string
	ClusterID           uuid.UUID
	Token               string
	AgentVersion        string
	KubeconfigPath      string
	TargetNamespaces    []string
	RequestTimeout      time.Duration
	HeartbeatInterval   time.Duration
	InventoryInterval   time.Duration
	SyncInterval        time.Duration
	Paths               EndpointPaths
}

type EndpointPaths struct {
	Heartbeat string
	Inventory string
	Sync      string
	Ack       string
}

func Load() (Config, error) {
	clusterIDStr := strings.TrimSpace(os.Getenv("AGENT_CLUSTER_ID"))
	if clusterIDStr == "" {
		return Config{}, errors.New("AGENT_CLUSTER_ID is required")
	}

	clusterID, err := uuid.Parse(clusterIDStr)
	if err != nil {
		return Config{}, fmt.Errorf("parse AGENT_CLUSTER_ID: %w", err)
	}

	token := strings.TrimSpace(os.Getenv("AGENT_TOKEN"))
	if token == "" {
		return Config{}, errors.New("AGENT_TOKEN is required")
	}

	cfg := Config{
		ControlPlaneBaseURL: getEnv("AGENT_CONTROL_PLANE_URL", "http://localhost:8000"),
		ClusterID:           clusterID,
		Token:               token,
		AgentVersion:        getEnv("AGENT_VERSION", "dev"),
		KubeconfigPath:      getEnv("AGENT_KUBECONFIG", ""),
		TargetNamespaces:    splitCSV(getEnv("AGENT_TARGET_NAMESPACES", "edge-functions")),
		RequestTimeout:      getDurationEnv("AGENT_REQUEST_TIMEOUT", 10*time.Second),
		HeartbeatInterval:   getDurationEnv("AGENT_HEARTBEAT_INTERVAL", 15*time.Second),
		InventoryInterval:   getDurationEnv("AGENT_INVENTORY_INTERVAL", time.Minute),
		SyncInterval:        getDurationEnv("AGENT_SYNC_INTERVAL", 10*time.Second),
		Paths: EndpointPaths{
			Heartbeat: getEnv("AGENT_PATH_HEARTBEAT", "/api/v1/clusters/%s/heartbeat"),
			Inventory: getEnv("AGENT_PATH_INVENTORY", "/api/v1/clusters/%s/inventory"),
			Sync:      getEnv("AGENT_PATH_SYNC", "/api/v1/clusters/%s/sync"),
			Ack:       getEnv("AGENT_PATH_ACK", "/api/v1/clusters/%s/sync/ack"),
		},
	}

	if cfg.HeartbeatInterval <= 0 || cfg.InventoryInterval <= 0 || cfg.SyncInterval <= 0 || cfg.RequestTimeout <= 0 {
		return Config{}, errors.New("intervals and timeout must be > 0")
	}

	cfg.ControlPlaneBaseURL = strings.TrimRight(cfg.ControlPlaneBaseURL, "/")
	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	if strings.ContainsAny(v, "hms") {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}

	seconds, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	namespaces := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		namespaces = append(namespaces, trimmed)
	}
	if len(namespaces) == 0 {
		return []string{"edge-functions"}
	}
	return namespaces
}
