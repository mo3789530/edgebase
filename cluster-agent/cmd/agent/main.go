package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgebase/cluster-agent/internal/client/controlplane"
	"github.com/edgebase/cluster-agent/internal/client/k8s"
	"github.com/edgebase/cluster-agent/internal/config"
	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/edgebase/cluster-agent/internal/service/apply"
	heartbeatSvc "github.com/edgebase/cluster-agent/internal/service/heartbeat"
	inventorySvc "github.com/edgebase/cluster-agent/internal/service/inventory"
	syncSvc "github.com/edgebase/cluster-agent/internal/service/sync"
	"github.com/edgebase/cluster-agent/internal/state"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	cpClient := controlplane.New(cfg)
	store := state.NewMemoryStore()
	collector := inventorySvc.Collector(inventorySvc.EmptyCollector{})
	resourceApplier := syncSvc.ResourceApplier(apply.NoopApplier{})

	k8sClient, err := k8s.New(cfg.KubeconfigPath)
	if err != nil {
		logger.Warn("failed to initialize kubernetes client, fallback to noop components", "error", err)
	} else {
		collector = inventorySvc.NewK8sCollector(k8sClient.Clientset(), cfg.TargetNamespaces)
		resourceApplier = apply.NewK8sApplier(k8sClient.Clientset())
	}

	hbService := heartbeatSvc.New(
		cpClient,
		staticHeartbeatBuilder{agentVersion: cfg.AgentVersion},
		cfg.ClusterID,
		cfg.HeartbeatInterval,
		logger,
	)
	invService := inventorySvc.New(
		collector,
		cpClient,
		store,
		cfg.ClusterID,
		cfg.InventoryInterval,
		logger,
	)
	sService := syncSvc.New(
		cpClient,
		resourceApplier,
		cpClient,
		store,
		cfg.ClusterID,
		cfg.SyncInterval,
		logger,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go hbService.Start(ctx)
	go invService.Start(ctx)
	go sService.Start(ctx)

	<-ctx.Done()
	logger.Info("cluster-agent shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	<-shutdownCtx.Done()
}

type staticHeartbeatBuilder struct {
	agentVersion string
}

func (b staticHeartbeatBuilder) Build() model.Heartbeat {
	return model.Heartbeat{
		AgentVersion:     b.agentVersion,
		Health:           model.HealthStatusHealthy,
		State:            model.AgentStateHealthy,
		LastSyncSuccess:  true,
		KubernetesAccess: false,
		ObservedAt:       time.Now().UTC(),
	}
}
