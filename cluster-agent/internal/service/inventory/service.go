package inventory

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/edgebase/cluster-agent/internal/client/controlplane"
	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/edgebase/cluster-agent/internal/state"
	"github.com/google/uuid"
)

type Collector interface {
	Collect(ctx context.Context, clusterID uuid.UUID) (model.ClusterInventory, error)
}

type Reporter interface {
	ReportInventory(ctx context.Context, clusterID uuid.UUID, inventory model.ClusterInventory) error
}

type Service struct {
	collector Collector
	reporter  Reporter
	store     state.Store
	clusterID uuid.UUID
	interval  time.Duration
	logger    *slog.Logger
}

func New(collector Collector, reporter Reporter, store state.Store, clusterID uuid.UUID, interval time.Duration, logger *slog.Logger) *Service {
	return &Service{
		collector: collector,
		reporter:  reporter,
		store:     store,
		clusterID: clusterID,
		interval:  interval,
		logger:    logger,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.collectAndSend(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectAndSend(ctx)
		}
	}
}

func (s *Service) collectAndSend(ctx context.Context) {
	inventory, err := s.collector.Collect(ctx, s.clusterID)
	if err != nil {
		s.logger.Warn("inventory collection failed", "error", err)
		return
	}
	if inventory.ObservedAt.IsZero() {
		inventory.ObservedAt = time.Now().UTC()
	}

	err = s.reporter.ReportInventory(ctx, s.clusterID, inventory)
	if err != nil {
		if errors.Is(err, controlplane.ErrInventoryEndpointUnavailable) {
			s.logger.Info("inventory endpoint is not available on control plane yet")
			return
		}
		s.logger.Warn("inventory reporting failed", "error", err)
		return
	}

	s.store.SetLastInventoryAt(inventory.ObservedAt)
	s.logger.Debug("inventory sent")
}

type EmptyCollector struct{}

func (c EmptyCollector) Collect(ctx context.Context, clusterID uuid.UUID) (model.ClusterInventory, error) {
	return model.ClusterInventory{
		ClusterID:  clusterID,
		ObservedAt: time.Now().UTC(),
	}, nil
}
