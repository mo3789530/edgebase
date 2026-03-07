package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/edgebase/platform/control-plane/internal/model"
	"github.com/edgebase/platform/control-plane/internal/repository"
	"github.com/google/uuid"
)

type RouteService interface {
	CreateRoute(ctx context.Context, input CreateRouteInput) (*model.RouteDefinition, error)
	ListRoutes(ctx context.Context) ([]model.RouteDefinition, error)
	ListGatewayRoutes(ctx context.Context, clusterID uuid.UUID) ([]GatewayRoute, error)
}

type CreateRouteInput struct {
	Host                 string
	Path                 string
	FunctionDefinitionID uuid.UUID
	Methods              []string
	TimeoutMs            int32
	RetryPolicy          string
	ClusterSelector      string
}

type GatewayRoute struct {
	ID          uuid.UUID `json:"id"`
	Host        string    `json:"host"`
	Path        string    `json:"path"`
	Methods     []string  `json:"methods"`
	ServiceName string    `json:"service_name"`
	Namespace   string    `json:"namespace"`
	TimeoutMs   int32     `json:"timeout_ms"`
}

type routeService struct {
	routeRepo      repository.RouteRepository
	targetRepo     repository.FunctionDeploymentTargetRepository
	definitionRepo repository.FunctionDefinitionRepository
}

func NewRouteService(
	routeRepo repository.RouteRepository,
	targetRepo repository.FunctionDeploymentTargetRepository,
	definitionRepo repository.FunctionDefinitionRepository,
) RouteService {
	return &routeService{
		routeRepo:      routeRepo,
		targetRepo:     targetRepo,
		definitionRepo: definitionRepo,
	}
}

func (s *routeService) CreateRoute(ctx context.Context, input CreateRouteInput) (*model.RouteDefinition, error) {
	if _, err := s.definitionRepo.GetByID(ctx, input.FunctionDefinitionID); err != nil {
		return nil, err
	}
	if len(input.Methods) == 0 {
		input.Methods = []string{"POST"}
	}
	if input.TimeoutMs <= 0 {
		input.TimeoutMs = 3000
	}

	methods, err := json.Marshal(input.Methods)
	if err != nil {
		return nil, fmt.Errorf("marshal methods: %w", err)
	}

	route := &model.RouteDefinition{
		ID:                   uuid.New(),
		Host:                 input.Host,
		Path:                 input.Path,
		Methods:              string(methods),
		FunctionDefinitionID: input.FunctionDefinitionID,
		TimeoutMs:            input.TimeoutMs,
		RetryPolicy:          input.RetryPolicy,
		ClusterSelector:      input.ClusterSelector,
		Enabled:              true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	if err := s.routeRepo.Create(ctx, route); err != nil {
		return nil, err
	}
	return route, nil
}

func (s *routeService) ListRoutes(ctx context.Context) ([]model.RouteDefinition, error) {
	return s.routeRepo.List(ctx)
}

func (s *routeService) ListGatewayRoutes(ctx context.Context, clusterID uuid.UUID) ([]GatewayRoute, error) {
	routes, err := s.routeRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	targets, err := s.targetRepo.ListByClusterID(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	targetByFunctionID := make(map[uuid.UUID]model.FunctionDeploymentTarget, len(targets))
	for _, target := range targets {
		targetByFunctionID[target.FunctionDefinitionID] = target
	}

	result := make([]GatewayRoute, 0, len(routes))
	for _, route := range routes {
		target, ok := targetByFunctionID[route.FunctionDefinitionID]
		if !ok {
			continue
		}
		definition, err := s.definitionRepo.GetByID(ctx, route.FunctionDefinitionID)
		if err != nil {
			return nil, err
		}
		methods := []string{}
		if err := json.Unmarshal([]byte(route.Methods), &methods); err != nil {
			return nil, fmt.Errorf("unmarshal route methods: %w", err)
		}
		result = append(result, GatewayRoute{
			ID:          route.ID,
			Host:        route.Host,
			Path:        route.Path,
			Methods:     methods,
			ServiceName: sanitizeGatewayName("fn-" + definition.Name),
			Namespace:   target.Namespace,
			TimeoutMs:   route.TimeoutMs,
		})
	}
	return result, nil
}

func sanitizeGatewayName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

