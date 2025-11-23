package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	"github.com/edgebase/platform/control-plane/internal/config"
	"github.com/edgebase/platform/control-plane/internal/db"
	"github.com/edgebase/platform/control-plane/internal/handler"
	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/mqtt"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/storage"
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
		&model.Function{},
		&model.SchemaMigration{},
		&model.NodeFunctionDeployment{},
		&model.SyncRecord{},
	); err != nil {
		log.Fatalf("failed to migrate DB: %v", err)
	}

	// Initialize MinIO client
	minioClient, err := storage.Init(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOBucket)
	if err != nil {
		log.Fatalf("failed to init MinIO: %v", err)
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
	funcRepo := repository.NewFunctionRepository(dbConn)
	schemaRepo := repository.NewSchemaRepository(dbConn)
	syncRepo := repository.NewSyncRepository(dbConn)

	// Initialize Services
	nodeSvc := service.NewNodeService(nodeRepo)
	artifactSvc := service.NewArtifactService(funcRepo, minioClient)
	schemaSvc := service.NewSchemaService(schemaRepo)
	syncSvc := service.NewSyncService(syncRepo, nodeRepo, funcRepo, schemaRepo, artifactSvc)

	// Initialize Fiber app
	app := fiber.New()

	// Register routes
	h := handler.NewHandler(nodeSvc, syncSvc, artifactSvc, schemaSvc)
	h.RegisterRoutes(app)

	// Start server
	addr := ":" + cfg.ServerPort
	log.Printf("starting server on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
