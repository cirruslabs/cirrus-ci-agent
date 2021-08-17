package updatebatcher

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"log"
)

type UpdateBatcher struct {
	allUpdates     []*api.CommandResult
	currentUpdates []*api.CommandResult
}

func New() *UpdateBatcher {
	return &UpdateBatcher{
		allUpdates:     []*api.CommandResult{},
		currentUpdates: []*api.CommandResult{},
	}
}

func (ub *UpdateBatcher) Queue(update *api.CommandResult) {
	ub.allUpdates = append(ub.allUpdates, update)
	ub.currentUpdates = append(ub.currentUpdates, update)
}

func (ub *UpdateBatcher) Flush(ctx context.Context, taskIdentification *api.TaskIdentification) {
	_, err := client.CirrusClient.ReportCommandUpdates(ctx, &api.ReportCommandUpdatesRequest{
		TaskIdentification: taskIdentification,
		Updates:            ub.currentUpdates,
	})
	if err != nil {
		log.Printf("Failed to report command updates: %v\n", err)
		return
	}
	ub.currentUpdates = ub.currentUpdates[:0]
}

func (ub *UpdateBatcher) History() []*api.CommandResult {
	return ub.allUpdates
}
