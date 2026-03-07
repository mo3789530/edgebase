package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgebase/cluster-agent/internal/config"
	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
)

func TestClient_ReportHeartbeat(t *testing.T) {
	clusterID := uuid.New()
	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.URL.Path; got != "/api/v1/nodes/"+clusterID.String()+"/heartbeat" {
			t.Fatalf("path = %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("auth header = %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	client := New(config.Config{
		ControlPlaneBaseURL: ts.URL,
		Token:               "token",
		Paths: config.EndpointPaths{
			Heartbeat: "/api/v1/nodes/%s/heartbeat",
		},
	})

	err := client.ReportHeartbeat(context.Background(), clusterID, model.Heartbeat{})
	if err != nil {
		t.Fatalf("ReportHeartbeat() error = %v", err)
	}
	if !called {
		t.Fatal("server was not called")
	}
}

func TestClient_FetchSyncPlan(t *testing.T) {
	clusterID := uuid.New()
	syncID := uuid.New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.URL.Path; got != "/api/v1/nodes/"+clusterID.String()+"/sync" {
			t.Fatalf("path = %s", got)
		}
		response := model.SyncPlan{
			SyncID: syncID,
			Actions: []model.SyncAction{
				{Type: "APPLY_DEPLOYMENT", Order: 1},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	client := New(config.Config{
		ControlPlaneBaseURL: ts.URL,
		Token:               "token",
		Paths: config.EndpointPaths{
			Sync: "/api/v1/nodes/%s/sync",
		},
	})

	plan, err := client.FetchSyncPlan(context.Background(), clusterID)
	if err != nil {
		t.Fatalf("FetchSyncPlan() error = %v", err)
	}
	if plan.SyncID != syncID {
		t.Fatalf("SyncID = %s, want %s", plan.SyncID, syncID)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("actions len = %d", len(plan.Actions))
	}
}

func TestClient_ReportInventory_404(t *testing.T) {
	clusterID := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := New(config.Config{
		ControlPlaneBaseURL: ts.URL,
		Token:               "token",
		Paths: config.EndpointPaths{
			Inventory: "/api/v1/clusters/%s/inventory",
		},
	})

	err := client.ReportInventory(context.Background(), clusterID, model.ClusterInventory{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInventoryEndpointUnavailable) {
		t.Fatalf("expected inventory unavailable error, got %v", err)
	}
}
