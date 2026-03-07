package apply

import (
	"context"

	"github.com/edgebase/cluster-agent/internal/model"
)

type NoopApplier struct{}

func (a NoopApplier) Apply(ctx context.Context, plan *model.SyncPlan) (model.SyncAck, error) {
	results := make([]model.SyncAckResource, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		results = append(results, model.SyncAckResource{
			ResourceType: action.Type,
			ResourceName: action.Description,
			Status:       "skipped",
		})
	}

	return model.SyncAck{
		SyncID:  plan.SyncID,
		Success: true,
		Results: results,
	}, nil
}
