package sync

import (
	"context"
	"log/slog"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/edgebase/cluster-agent/internal/state"
	"github.com/google/uuid"
)

type PlanFetcher interface {
	FetchSyncPlan(ctx context.Context, clusterID uuid.UUID) (*model.SyncPlan, error)
}

type AckReporter interface {
	ReportSyncAck(ctx context.Context, clusterID uuid.UUID, ack model.SyncAck) error
}

type ResourceApplier interface {
	Apply(ctx context.Context, plan *model.SyncPlan) (model.SyncAck, error)
}

type Service struct {
	fetcher   PlanFetcher
	applier   ResourceApplier
	reporter  AckReporter
	store     state.Store
	clusterID uuid.UUID
	interval  time.Duration
	logger    *slog.Logger
}

func New(fetcher PlanFetcher, applier ResourceApplier, reporter AckReporter, store state.Store, clusterID uuid.UUID, interval time.Duration, logger *slog.Logger) *Service {
	return &Service{
		fetcher:   fetcher,
		applier:   applier,
		reporter:  reporter,
		store:     store,
		clusterID: clusterID,
		interval:  interval,
		logger:    logger,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.syncOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncOnce(ctx)
		}
	}
}

func (s *Service) syncOnce(ctx context.Context) {
	plan, err := s.fetcher.FetchSyncPlan(ctx, s.clusterID)
	if err != nil {
		s.logger.Warn("fetch sync plan failed", "error", err)
		return
	}
	if plan == nil || plan.SyncID == uuid.Nil {
		return
	}
	if len(plan.Actions) == 0 {
		return
	}

	if plan.SyncID == s.store.LastSyncID() && plan.Generation <= s.store.LastGeneration() {
		s.logger.Debug("skip duplicated sync plan", "sync_id", plan.SyncID.String(), "generation", plan.Generation)
		return
	}

	ack, err := s.applier.Apply(ctx, plan)
	if err != nil {
		s.logger.Warn("apply sync plan failed", "sync_id", plan.SyncID.String(), "error", err)
		ack = model.SyncAck{
			SyncID:  plan.SyncID,
			Success: false,
			Results: []model.SyncAckResource{{
				ResourceType: "Plan",
				ResourceName: plan.SyncID.String(),
				Status:       "failed",
				ErrorMessage: err.Error(),
			}},
		}
	}

	if err := s.reporter.ReportSyncAck(ctx, s.clusterID, ack); err != nil {
		s.logger.Warn("report sync ack failed", "sync_id", plan.SyncID.String(), "error", err)
		return
	}

	s.store.SetLastSyncID(plan.SyncID)
	s.store.SetLastGeneration(plan.Generation)
	s.logger.Info("sync plan processed", "sync_id", plan.SyncID.String(), "success", ack.Success)
}
