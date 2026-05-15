package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/xescugc/pikoci/pikoci/queue"
	"github.com/xescugc/pikoci/pikoci/resource"
	"gocloud.dev/pubsub"
)

// Scheduler polls the database for resources due for a check and sends messages to the topic.
type Scheduler struct {
	resources resource.Repository
	topic     queue.Topic
	logger    *slog.Logger
	interval  time.Duration
}

// New creates a new Scheduler.
func New(resources resource.Repository, topic queue.Topic, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		resources: resources,
		topic:     topic,
		logger:    logger,
		interval:  10 * time.Second,
	}
}

// Start launches the polling goroutine that ticks every interval.
func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.tick(ctx)
			}
		}
	}()
}

func (s *Scheduler) tick(ctx context.Context) {
	due, err := s.resources.FilterDueResources(ctx)
	if err != nil {
		s.logger.Error("failed to filter due resources", "error", err)
		return
	}

	for _, rwp := range due {
		s.logger.Info("Checking resource ...", "Pipeline", rwp.PipelineName, "Resource", rwp.Canonical)

		m := queue.Body{
			TeamCanonical:     rwp.TeamCanonical,
			PipelineName:      rwp.PipelineName,
			ResourceCanonical: rwp.Canonical,
		}
		mb, err := json.Marshal(m)
		if err != nil {
			s.logger.Error("failed to marshal Message Body", "error", err)
			continue
		}
		err = s.topic.Send(ctx, &pubsub.Message{
			Body: mb,
		})
		if err != nil {
			s.logger.Error("failed to send Topic", "error", err)
			continue
		}

		now := time.Now()
		rwp.Resource.LastCheck = now

		spec := rwp.CheckInterval
		if spec == "" {
			spec = "@every 1m"
		}
		nextCheck, err := ComputeNextCheck(spec, now)
		if err != nil {
			s.logger.Error("failed to compute next check", "error", err)
			continue
		}
		rwp.Resource.NextCheck = nextCheck

		err = s.resources.Update(ctx, rwp.TeamCanonical, rwp.PipelineName, rwp.Canonical, rwp.Resource)
		if err != nil {
			s.logger.Error("failed to update resource", "error", err)
		}
	}
}
