package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
	"log/slog"
)

type fakeFetcher struct {
	routes []model.GatewayRoute
}

func (f fakeFetcher) FetchGatewayRoutes(ctx context.Context, clusterID uuid.UUID) ([]model.GatewayRoute, error) {
	return f.routes, nil
}

func TestMatchRoute(t *testing.T) {
	svc := New(fakeFetcher{}, uuid.New(), time.Second, ":0", slog.Default())
	svc.routes = []model.GatewayRoute{{
		ID:          uuid.New(),
		Host:        "api.edgebase.local",
		Path:        "/normalize",
		Methods:     []string{"POST"},
		ServiceName: "fn-telemetry-normalizer",
		Namespace:   "edge-functions",
		TimeoutMs:   3000,
	}}

	req := httptest.NewRequest(http.MethodPost, "http://api.edgebase.local/normalize", nil)
	route, ok := svc.matchRoute(req)
	if !ok {
		t.Fatalf("expected route match")
	}
	if route.ServiceName != "fn-telemetry-normalizer" {
		t.Fatalf("service name = %s", route.ServiceName)
	}
}

