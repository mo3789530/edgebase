package handler

import (
	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	nodeSvc       service.NodeService
	syncSvc       service.SyncService
	artifactSvc   service.ArtifactService
	schemaSvc     service.SchemaService
	telemetrySvc  service.TelemetryService
}

func NewHandler(
	nodeSvc service.NodeService,
	syncSvc service.SyncService,
	artifactSvc service.ArtifactService,
	schemaSvc service.SchemaService,
	telemetrySvc service.TelemetryService,
) *Handler {
	return &Handler{
		nodeSvc:       nodeSvc,
		syncSvc:       syncSvc,
		artifactSvc:   artifactSvc,
		schemaSvc:     schemaSvc,
		telemetrySvc:  telemetrySvc,
	}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Node endpoints
	nodes := api.Group("/nodes")
	nodes.Post("/register", h.RegisterNode)
	nodes.Post("/:id/heartbeat", h.Heartbeat)
	nodes.Get("/:id/sync", h.GetSyncInfo)
	nodes.Post("/:id/sync/ack", h.AckSync)

	// Function (WASM) endpoints
	funcs := api.Group("/functions")
	funcs.Post("/", h.CreateFunction)
	funcs.Get("/:id", h.GetFunction)
	funcs.Post("/:id/upload", h.UploadArtifact)
	funcs.Get("/:id/download", h.DownloadFunction)
	funcs.Delete("/:id", h.DeleteFunction)

	// Artifact endpoints
	artifacts := api.Group("/artifacts")
	artifacts.Get("/:id/:version", h.DownloadArtifact)

	// Deployment endpoints
	deploy := api.Group("/functions/:function_id/deploy")
	deploy.Post("/:node_id", h.DeployFunction)

	// Route endpoints
	routes := api.Group("/routes")
	routes.Post("/", h.CreateRoute)
	routes.Get("/", h.ListRoutes)

	// Schema endpoints
	schemas := api.Group("/schemas")
	schemas.Post("/", h.RegisterSchema)
	schemas.Get("/", h.ListSchemas)

	// Telemetry endpoints
	sync := api.Group("/sync")
	sync.Post("/telemetry", h.SyncTelemetry)
	sync.Get("/commands/:device_id", h.GetCommands)
	sync.Post("/ack/:command_id", h.AckCommand)
	sync.Get("/status/:device_id", h.GetSyncStatus)

	// Device endpoints
	devices := api.Group("/devices")
	devices.Post("/register", h.RegisterDevice)
}

