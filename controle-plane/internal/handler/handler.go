package handler

import (
	"net/http"
	"time"

	"github.com/edgebase/platform/control-plane/internal/auth"
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	nodeSvc       service.NodeService
	syncSvc       service.SyncService
	artifactSvc   service.ArtifactService
	schemaSvc     service.SchemaService
	telemetrySvc  service.TelemetryService
	authMgr       *auth.Manager
	tokenExpiry   time.Duration
}

func NewHandler(
	nodeSvc service.NodeService,
	syncSvc service.SyncService,
	artifactSvc service.ArtifactService,
	schemaSvc service.SchemaService,
	telemetrySvc service.TelemetryService,
	authMgr *auth.Manager,
	tokenExpiry time.Duration,
) *Handler {
	return &Handler{
		nodeSvc:       nodeSvc,
		syncSvc:       syncSvc,
		artifactSvc:   artifactSvc,
		schemaSvc:     schemaSvc,
		telemetrySvc:  telemetrySvc,
		authMgr:       authMgr,
		tokenExpiry:   tokenExpiry,
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

	// Auth endpoints
	authGroup := api.Group("/auth")
	authGroup.Post("/refresh", h.RefreshToken)

	// Function (WASM) endpoints (require auth)
	funcs := api.Group("/functions", auth.AuthMiddleware(h.authMgr))
	funcs.Post("/", h.CreateFunction)
	funcs.Get("/:id", h.GetFunction)
	funcs.Post("/:id/upload", h.UploadArtifact)
	funcs.Get("/:id/download", h.DownloadFunction)
	funcs.Delete("/:id", h.DeleteFunction)

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



