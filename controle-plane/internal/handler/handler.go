package handler

import (
	"net/http"
	"time"

	"github.com/edgebase/platform/control-plane/internal/auth"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/edgebase/platform/control-plane/internal/timeseries"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	nodeSvc         service.NodeService
	syncSvc         service.SyncService
	artifactSvc     service.ArtifactService
	functionSvc     service.FunctionCatalogService
	deploymentSvc   service.FunctionDeploymentService
	controllerSvc   service.FunctionControllerService
	routeSvc        service.RouteService
	schemaSvc       service.SchemaService
	telemetrySvc    service.TelemetryService
	inventorySvc    service.InventoryService
	authMgr         *auth.Manager
	tokenExpiry     time.Duration
	metricCollector timeseries.MetricCollector
	logWriter       timeseries.LogWriter
}

func NewHandler(
	nodeSvc service.NodeService,
	syncSvc service.SyncService,
	artifactSvc service.ArtifactService,
	functionSvc service.FunctionCatalogService,
	deploymentSvc service.FunctionDeploymentService,
	controllerSvc service.FunctionControllerService,
	routeSvc service.RouteService,
	schemaSvc service.SchemaService,
	telemetrySvc service.TelemetryService,
	inventorySvc service.InventoryService,
	authMgr *auth.Manager,
	tokenExpiry time.Duration,
	metricCollector timeseries.MetricCollector,
	logWriter timeseries.LogWriter,
) *Handler {
	return &Handler{
		nodeSvc:         nodeSvc,
		syncSvc:         syncSvc,
		artifactSvc:     artifactSvc,
		functionSvc:     functionSvc,
		deploymentSvc:   deploymentSvc,
		controllerSvc:   controllerSvc,
		routeSvc:        routeSvc,
		schemaSvc:       schemaSvc,
		telemetrySvc:    telemetrySvc,
		inventorySvc:    inventorySvc,
		authMgr:         authMgr,
		tokenExpiry:     tokenExpiry,
		metricCollector: metricCollector,
		logWriter:       logWriter,
	}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Node endpoints (register is public, others require auth)
	nodes := api.Group("/nodes")
	nodes.Post("/register", h.RegisterNode)
	nodes.Post("/:id/heartbeat", auth.AuthMiddleware(h.authMgr), h.Heartbeat)
	nodes.Get("/:id/sync", auth.AuthMiddleware(h.authMgr), h.GetSyncInfo)
	nodes.Post("/:id/sync/ack", auth.AuthMiddleware(h.authMgr), h.AckSync)
	nodes.Post("/:id/schema_status", auth.AuthMiddleware(h.authMgr), h.UpdateSchemaStatus)

	// Cluster-agent compatibility endpoints
	clusters := api.Group("/clusters", auth.AuthMiddleware(h.authMgr))
		clusters.Post("/:id/heartbeat", h.ClusterHeartbeat)
		clusters.Post("/:id/inventory", h.ClusterInventory)
		clusters.Get("/:id/sync", h.ClusterGetSyncInfo)
		clusters.Post("/:id/sync/ack", h.ClusterAckSync)
		clusters.Get("/:id/gateway/routes", h.ClusterListGatewayRoutes)

	// Auth endpoints
	authGroup := api.Group("/auth")
	authGroup.Post("/refresh", h.RefreshToken)

	// Function endpoints (require auth)
	api.Post("/functions", auth.AuthMiddleware(h.authMgr), h.CreateFunction)
	api.Get("/functions", auth.AuthMiddleware(h.authMgr), h.ListFunctions)
	api.Get("/functions/:id", auth.AuthMiddleware(h.authMgr), h.GetFunction)
	api.Post("/functions/:id/revisions", auth.AuthMiddleware(h.authMgr), h.CreateFunctionRevision)
	api.Post("/functions/:id/deployments", auth.AuthMiddleware(h.authMgr), h.CreateFunctionDeploymentTargets)
	api.Post("/functions/:id/upload", auth.AuthMiddleware(h.authMgr), h.UploadArtifact)
	api.Get("/functions/:id/download", auth.AuthMiddleware(h.authMgr), h.DownloadFunction)
	api.Delete("/functions/:id", auth.AuthMiddleware(h.authMgr), h.DeleteFunction)

	// Artifact endpoints (require auth)
	artifacts := api.Group("/artifacts", auth.AuthMiddleware(h.authMgr))
	artifacts.Get("/:id/:version", h.DownloadArtifact)

	// Deployment endpoints (require auth)
	deploy := api.Group("/functions/:function_id/deploy", auth.AuthMiddleware(h.authMgr))
	deploy.Post("/:node_id", h.DeployFunction)

	// Route endpoints (require auth)
	routes := api.Group("/routes", auth.AuthMiddleware(h.authMgr))
	routes.Post("/", h.CreateRoute)
	routes.Get("/", h.ListRoutes)

	// Schema endpoints (require auth)
	schemas := api.Group("/schemas", auth.AuthMiddleware(h.authMgr))
	schemas.Post("/", h.RegisterSchema)
	schemas.Get("/", h.ListSchemas)
	schemas.Get("/:version/download", h.DownloadSchema)

	// Telemetry endpoints (require auth)
	sync := api.Group("/sync", auth.AuthMiddleware(h.authMgr))
	sync.Post("/telemetry", h.SyncTelemetry)
	sync.Get("/commands/:device_id", h.GetCommands)
	sync.Post("/ack/:command_id", h.AckCommand)
	sync.Get("/status/:device_id", h.GetSyncStatus)

	// Device endpoints (require auth)
	devices := api.Group("/devices", auth.AuthMiddleware(h.authMgr))
	devices.Post("/register", h.RegisterDevice)

	// Documentation endpoints
	app.Get("/docs", h.DocsHTML)
	app.Get("/openapi.yaml", h.OpenAPISpec)
}

// parseUUID extracts and validates UUID from route parameter
func (h *Handler) parseUUID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	idStr := c.Params(param)
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
		return uuid.Nil, err
	}
	return id, nil
}
