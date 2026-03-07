package heartbeat

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/google/uuid"
)

type fakeReporter struct {
	mu    sync.Mutex
	calls int
}

func (f *fakeReporter) ReportHeartbeat(ctx context.Context, clusterID uuid.UUID, heartbeat model.Heartbeat) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return nil
}

func (f *fakeReporter) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeBuilder struct{}

func (f fakeBuilder) Build() model.Heartbeat {
	return model.Heartbeat{Health: model.HealthStatusHealthy, State: model.AgentStateHealthy}
}

func TestService_Start_SendsHeartbeat(t *testing.T) {
	reporter := &fakeReporter{}
	service := New(
		reporter,
		fakeBuilder{},
		uuid.New(),
		10*time.Millisecond,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go service.Start(ctx)
	time.Sleep(35 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if reporter.Calls() < 2 {
		t.Fatalf("calls = %d, want >= 2", reporter.Calls())
	}
}
