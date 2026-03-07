package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgebase/cluster-agent/internal/client/controlplane"
	"github.com/edgebase/cluster-agent/internal/config"
	gatewaySvc "github.com/edgebase/cluster-agent/internal/service/gateway"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	cpClient := controlplane.New(cfg)
	gateway := gatewaySvc.New(cpClient, cfg.ClusterID, cfg.GatewayRefreshIntvl, cfg.GatewayListenAddr, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := gateway.Start(ctx); err != nil {
		logger.Error("gateway stopped with error", "error", err)
		os.Exit(1)
	}
}
