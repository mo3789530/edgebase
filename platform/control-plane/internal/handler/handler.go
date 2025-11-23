package handler

import (
	"net/http"

	"github.com/edgebase/platform/control-plane/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	nodeSvc     service.NodeService
	syncSvc     service.SyncService
	artifactSvc service.ArtifactService
	schemaSvc   service.SchemaService
}

func NewHandler(
	nodeSvc service.NodeService,
	syncSvc service.SyncService,
	artifactSvc service.ArtifactService,
	schemaSvc service.SchemaService,
) *Handler {
	return &Handler{
		nodeSvc:     nodeSvc,
		syncSvc:     syncSvc,
		artifactSvc: artifactSvc,
		schemaSvc:   schemaSvc,
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
	funcs.Post("/", h.UploadFunction)
	funcs.Get("/:id", h.GetFunctionMeta)
	funcs.Get("/:id/download", h.DownloadFunction)
	funcs.Delete("/:id", h.DeleteFunction)

	// Schema endpoints
	schemas := api.Group("/schemas")
	schemas.Post("/", h.RegisterSchema)
	schemas.Get("/", h.ListSchemas)
}

type RegisterNodeRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

func (h *Handler) RegisterNode(c *fiber.Ctx) error {
	var req RegisterNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	node, token, err := h.nodeSvc.RegisterNode(c.Context(), req.Name, req.Region)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"node":  node,
		"token": token,
	})
}

func (h *Handler) Heartbeat(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}

	if err := h.nodeSvc.Heartbeat(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) GetSyncInfo(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}

	// Parse current state from query or body? Design says "Poll Sync".
	// Usually GET request params or body. GET with body is discouraged but possible.
	// Or maybe it's a POST? Design says "GET /api/v1/nodes/{id}/sync".
	// If it's GET, state must be in query params.
	// But state is complex (list of functions).
	// Maybe the design meant POST for sync check?
	// "EA1 -->|Poll Sync| API".
	// Let's assume for now we just return the plan based on what we know, OR we expect a body.
	// Fiber supports body in GET.
	var currentState service.NodeState
	if err := c.BodyParser(&currentState); err != nil {
		// If no body, assume empty state (fresh node)
		currentState = service.NodeState{}
	}

	plan, err := h.syncSvc.GetSyncPlan(c.Context(), id, currentState)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(plan)
}

func (h *Handler) AckSync(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid node id"})
	}

	var req struct {
		SyncID uuid.UUID          `json:"sync_id"`
		Result service.SyncResult `json:"result"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.syncSvc.AcknowledgeSync(c.Context(), id, req.SyncID, req.Result); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "acked"})
}

func (h *Handler) UploadFunction(c *fiber.Ctx) error {
	name := c.FormValue("name")
	version := c.FormValue("version")

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file required"})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open file"})
	}
	defer f.Close()

	buffer := make([]byte, file.Size)
	_, err = f.Read(buffer)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}

	fn, err := h.artifactSvc.UploadFunction(c.Context(), name, version, buffer)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fn)
}

func (h *Handler) GetFunctionMeta(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	fn, err := h.artifactSvc.GetFunction(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "function not found"})
	}

	return c.JSON(fn)
}

func (h *Handler) DownloadFunction(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	url, err := h.artifactSvc.GetDownloadURL(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Redirect or return URL? Design says "Download WASM".
	// Usually we return a 302 Redirect to the signed URL.
	return c.Redirect(url)
}

func (h *Handler) DeleteFunction(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid function id"})
	}

	if err := h.artifactSvc.DeleteFunction(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(http.StatusNoContent)
}

type RegisterSchemaRequest struct {
	Version     int    `json:"version"`
	UpSQL       string `json:"up_sql"`
	DownSQL     string `json:"down_sql"`
	Description string `json:"description"`
}

func (h *Handler) RegisterSchema(c *fiber.Ctx) error {
	var req RegisterSchemaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.schemaSvc.RegisterSchema(c.Context(), req.Version, req.UpSQL, req.DownSQL, req.Description); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "schema registered"})
}

func (h *Handler) ListSchemas(c *fiber.Ctx) error {
	schemas, err := h.schemaSvc.ListSchemas(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(schemas)
}
