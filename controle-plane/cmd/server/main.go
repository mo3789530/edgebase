package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	"github.com/edgebase/platform/control-plane/internal/auth"
	"github.com/edgebase/platform/control-plane/internal/cache"
	"github.com/edgebase/platform/control-plane/internal/config"
	"github.com/edgebase/platform/control-plane/internal/cors"
	"github.com/edgebase/platform/control-plane/internal/db"
	"github.com/edgebase/platform/control-plane/internal/errors"
	"github.com/edgebase/platform/control-plane/internal/handler"
	"github.com/edgebase/platform/control-plane/internal/logger"
	"github.com/edgebase/platform/control-plane/internal/metrics"
	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/mqtt"
	"github.com/edgebase/platform/control-plane/internal/ratelimit"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/shutdown"
	"github.com/edgebase/platform/control-plane/internal/storage"
	"github.com/edgebase/platform/control-plane/internal/timeseries"
	"github.com/edgebase/platform/control-plane/internal/timeseries/influxdb"
)

func main() {
	// Load .env if present
	_ = godotenv.Load()

	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize database connection
	dbConn, err := db.Init(cfg.DatabaseURL, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, cfg.DBConnMaxLifetime)
	if err != nil {
		log.Fatalf("failed to connect DB: %v", err)
	}
	defer func() {
		if sqlDB, err := dbConn.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	// Auto Migrate
	if err := dbConn.AutoMigrate(
		&model.Node{},
		&model.Cluster{},
		&model.ClusterNode{},
		&model.ClusterSyncRecord{},
		&model.Function{},
		&model.SchemaMigration{},
		&model.NodeFunctionDeployment{},
		&model.SyncRecord{},
		&model.Device{},
		&model.TelemetryData{},
		&model.Command{},
		&model.SyncStatus{},
		&model.NodeSchemaStatus{},
		&model.AuditLog{},
	); err != nil {
		log.Fatalf("failed to migrate DB: %v", err)
	}

	// Initialize Storage client
	var storageClient storage.Client
	if cfg.S3Enabled {
		storageClient, err = storage.InitS3(context.Background(), cfg.S3Region, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket)
		if err != nil {
			log.Fatalf("failed to init S3: %v", err)
		}
		log.Println("Connected to S3")
	} else {
		storageClient, err = storage.Init(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOBucket)
		if err != nil {
			log.Fatalf("failed to init MinIO: %v", err)
		}
	}

	// Initialize MQTT client (optional)
	var mqttClient *mqtt.Client
	if cfg.MQTTEnabled {
		mqttClient, err = mqtt.Init(cfg.MQTTBroker, true)
		if err != nil {
			log.Fatalf("failed to init MQTT: %v", err)
		}
		// TODO: Inject MQTT client into services if needed
		_ = mqttClient
	}

	// Initialize Repositories
	nodeRepo := repository.NewNodeRepository(dbConn)
	clusterRepo := repository.NewClusterRepository(dbConn)
	clusterInventoryRepo := repository.NewClusterInventoryRepository(dbConn)
	clusterSyncRepo := repository.NewClusterSyncRepository(dbConn)
	funcRepo := repository.NewFunctionRepository(dbConn)
	schemaRepo := repository.NewSchemaRepository(dbConn)
	syncRepo := repository.NewSyncRepository(dbConn)
	telemetryRepo := repository.NewTelemetryRepository(dbConn)

	// Initialize Services
	nodeSvc := service.NewNodeService(nodeRepo)
	clusterSvc := service.NewClusterService(clusterRepo)
	clusterInventorySvc := service.NewClusterInventoryService(clusterRepo, clusterInventoryRepo)
	clusterSyncSvc := service.NewClusterSyncService(clusterSyncRepo, funcRepo, schemaRepo)
	artifactSvc := service.NewArtifactService(funcRepo, storageClient)
	schemaSvc := service.NewSchemaService(schemaRepo, mqttClient)
	syncSvc := service.NewSyncService(syncRepo, nodeRepo, funcRepo, schemaRepo, artifactSvc)
	telemetrySvc := service.NewTelemetryService(telemetryRepo)
	_ = service.NewAuditService(dbConn)

	// Initialize Time-Series System
	var (
		metricCollector timeseries.MetricCollector
		logWriter       timeseries.LogWriter
		tsShutdownMgr   *timeseries.ShutdownManager
	)

	if cfg.TimeSeriesEnabled {
		log.Println("Initializing time-series system...")

		// Client
		influxClient, err := influxdb.NewClient(influxdb.Config{
			URL:    cfg.TimeSeriesDBURL,
			Token:  cfg.TimeSeriesDBToken,
			Org:    cfg.TimeSeriesDBOrg,
			Bucket: cfg.TimeSeriesDBBucket,
		})
		if err != nil {
			log.Printf("Warning: Failed to connect to time-series DB: %v", err)
		} else {
			// Store
			tsStore := influxdb.NewStore(influxClient)

			// Retention
			retentionMgr := timeseries.NewRetentionManager(tsStore, cfg.TimeSeriesRetentionDays)
			if err := retentionMgr.ApplyPolicy(context.Background()); err != nil {
				log.Printf("Warning: Failed to apply retention policy: %v", err)
			}

			// Batch Manager
			batchMgr := timeseries.NewBatchManager(tsStore, cfg.TimeSeriesBatchSize, time.Duration(cfg.TimeSeriesBatchTimeout)*time.Second)

			// Buffered Store (Adapter)
			bufferedStore := timeseries.NewBufferedStore(tsStore, batchMgr)

			// Collector & Writer
			metricCollector = timeseries.NewCollector(bufferedStore)
			logWriter = timeseries.NewWriter(bufferedStore)

			// Shutdown Manager
			tsShutdownMgr = timeseries.NewShutdownManager(tsStore, batchMgr)

			log.Println("Time-series system initialized")
		}
	}

	// Initialize cache (5 minute TTL)
	_ = cache.New(5 * time.Minute)

	// Initialize logger
	logger.Init("INFO")

	// Initialize rate limiter (100 requests per second per IP)
	limiter := ratelimit.NewLimiter(100, 1*time.Second)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: errors.ErrorHandler,
	})

	// Global middleware
	app.Use(cors.Middleware())
	app.Use(logger.RequestIDMiddleware())
	app.Use(logger.LoggingMiddleware())
	app.Use(metrics.Middleware())
	app.Use(ratelimit.Middleware(limiter))

	// Initialize Auth Manager
	authMgr := auth.NewManager(cfg.JWTSecret)

	// Health check routes
	healthHandler := handler.NewHealthHandler(dbConn)
	app.Get("/health", healthHandler.Health)
	app.Get("/health/ready", healthHandler.Ready)
	app.Get("/health/live", healthHandler.Live)
	app.Get("/metrics", healthHandler.Metrics)

	// Register API routes
	h := handler.NewHandler(nodeSvc, clusterSvc, clusterInventorySvc, clusterSyncSvc, syncSvc, artifactSvc, schemaSvc, telemetrySvc, authMgr, time.Duration(cfg.TokenExpiryHours)*time.Hour, metricCollector, logWriter)
	h.RegisterRoutes(app)

	// Setup graceful shutdown
	shutdownMgr := shutdown.NewManager(app, dbConn)
	if tsShutdownMgr != nil {
		shutdownMgr.AddHook(tsShutdownMgr.Shutdown)
	}
	shutdownMgr.Start()

	// Start server
	addr := ":" + cfg.ServerPort
	logger.Info("", "server_starting", map[string]interface{}{
		"port": cfg.ServerPort,
	})
	if err := app.Listen(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
