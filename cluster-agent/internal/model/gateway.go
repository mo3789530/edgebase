package model

import "github.com/google/uuid"

type GatewayRoute struct {
	ID          uuid.UUID `json:"id"`
	Host        string    `json:"host"`
	Path        string    `json:"path"`
	Methods     []string  `json:"methods"`
	ServiceName string    `json:"service_name"`
	Namespace   string    `json:"namespace"`
	TimeoutMs   int32     `json:"timeout_ms"`
}

