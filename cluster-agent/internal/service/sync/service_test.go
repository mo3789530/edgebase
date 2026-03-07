package sync

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/edgebase/cluster-agent/internal/model"
	"github.com/edgebase/cluster-agent/internal/state"
	"github.com/google/uuid"
)

type fakeFetcher struct {
	plan *model.SyncPlan
}

func (f fakeFetcher) FetchSyncPlan(ctx context.Context, clusterID uuid.UUID) (*model.SyncPlan, error) {
	return f.plan, nil
}

type fakeApplier struct {
	calls int
}

func (a *fakeApplier) Apply(ctx context.Context, plan *model.SyncPlan) (model.SyncAck, error) {
	a.calls++
	return model.SyncAck{SyncID: plan.SyncID, Success: true}, nil
}

type fakeAckReporter struct {
	calls int
}

func (r *fakeAckReporter) ReportSyncAck(ctx context.Context, clusterID uuid.UUID, ack model.SyncAck) error {
	r.calls++
	return nil
}

func TestService_SkipDuplicatedPlan(t *testing.T) {
	plan := &model.SyncPlan{
		SyncID:     uuid.New(),
		Generation: 10,
		Actions:    []model.SyncAction{{Type: "APPLY_DEPLOYMENT", Order: 1}},
	}

	store := state.NewMemoryStore()
	store.SetLastSyncID(plan.SyncID)
	store.SetLastGeneration(plan.Generation)

	applier := &fakeApplier{}
	reporter := &fakeAckReporter{}
	svc := New(
		fakeFetcher{plan: plan},
		applier,
		reporter,
		store,
		uuid.New(),
		time.Second,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	svc.syncOnce(context.Background())

	if applier.calls != 0 {
		t.Fatalf("applier calls = %d, want 0", applier.calls)
	}
	if reporter.calls != 0 {
		t.Fatalf("reporter calls = %d, want 0", reporter.calls)
	}
}
