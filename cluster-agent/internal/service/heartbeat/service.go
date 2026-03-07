package heartbeat

import (
	"context"
	"log/slog"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
)

type Reporter interface {
	ReportHeartbeat(ctx context.Context, clusterID uuid.UUID, heartbeat model.Heartbeat) error
}

type Builder interface {
	Build() model.Heartbeat
}

type Service struct {
	reporter  Reporter
	builder   Builder
	clusterID uuid.UUID
	interval  time.Duration
	logger    *slog.Logger
}

func New(reporter Reporter, builder Builder, clusterID uuid.UUID, interval time.Duration, logger *slog.Logger) *Service {
	return &Service{
		reporter:  reporter,
		builder:   builder,
		clusterID: clusterID,
		interval:  interval,
		logger:    logger,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.send(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.send(ctx)
		}
	}
}

func (s *Service) send(ctx context.Context) {
	heartbeat := s.builder.Build()
	if heartbeat.ObservedAt.IsZero() {
		heartbeat.ObservedAt = time.Now().UTC()
	}

	if err := s.reporter.ReportHeartbeat(ctx, s.clusterID, heartbeat); err != nil {
		s.logger.Warn("heartbeat failed", "error", err)
		return
	}

	s.logger.Debug("heartbeat sent")
}
