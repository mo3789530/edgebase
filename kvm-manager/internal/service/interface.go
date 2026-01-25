package service

import (
	"context"

	"github.com/edgebase/platform/kvm-manager/internal/model"
)

type VMManager interface {
	CreateVM(ctx context.Context, spec model.VMSpec) (*model.VM, error)
	StartVM(ctx context.Context, vmID string) error
	StopVM(ctx context.Context, vmID string, force bool) error
	DeleteVM(ctx context.Context, vmID string) error
	GetVM(ctx context.Context, vmID string) (*model.VM, error)
	ListVMs(ctx context.Context) ([]*model.VM, error)
}

type ResourceController interface {
	GetResourceUsage(ctx context.Context) (*model.ResourceUsage, error)
	ValidateResourceRequest(ctx context.Context, req model.ResourceRequest) error
	AllocateResources(ctx context.Context, vmID string, req model.ResourceRequest) error
	ReleaseResources(ctx context.Context, vmID string) error
}

type NetworkManager interface {
	CreateNetwork(ctx context.Context, config model.NetworkConfig) error
	AssignNetworkToVM(ctx context.Context, vmID string, networkID string) error
	GetNetworkConfig(ctx context.Context, vmID string) (*model.NetworkConfig, error)
}
