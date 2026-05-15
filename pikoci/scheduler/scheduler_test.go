package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/pikoci/pikoci/mock"
	"github.com/xescugc/pikoci/pikoci/queue"
	"github.com/xescugc/pikoci/pikoci/resource"
	"go.uber.org/mock/gomock"
	"gocloud.dev/pubsub"
)

func TestTick_NoDueResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	rr := mock.NewResourceRepository(ctrl)
	topic := mock.NewTopic(ctrl)
	logger := slog.Default()

	s := New(rr, topic, logger)

	rr.EXPECT().FilterDueResources(gomock.Any()).Return(nil, nil)

	s.tick(context.Background())
}

func TestTick_ProcessesDueResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	rr := mock.NewResourceRepository(ctrl)
	topic := mock.NewTopic(ctrl)
	logger := slog.Default()

	s := New(rr, topic, logger)

	due := []*resource.ResourceWithPipeline{
		{
			Resource: resource.Resource{
				ID:            1,
				Canonical:     "cron.timer",
				CheckInterval: "@every 30s",
			},
			TeamCanonical: "main",
			PipelineName:  "my-pipeline",
		},
	}

	rr.EXPECT().FilterDueResources(gomock.Any()).Return(due, nil)

	expectedBody := queue.Body{
		TeamCanonical:     "main",
		PipelineName:      "my-pipeline",
		ResourceCanonical: "cron.timer",
	}
	mb, _ := json.Marshal(expectedBody)
	topic.EXPECT().Send(gomock.Any(), &pubsub.Message{Body: mb}).Return(nil)

	rr.EXPECT().Update(gomock.Any(), "main", "my-pipeline", "cron.timer", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn, rCan string, r resource.Resource) error {
			assert.False(t, r.LastCheck.IsZero(), "LastCheck should be set")
			assert.False(t, r.NextCheck.IsZero(), "NextCheck should be set")
			assert.True(t, r.NextCheck.After(r.LastCheck), "NextCheck should be after LastCheck")
			// For @every 30s, next check should be ~30s after last check
			diff := r.NextCheck.Sub(r.LastCheck)
			assert.InDelta(t, 30*time.Second, diff, float64(2*time.Second))
			return nil
		},
	)

	s.tick(context.Background())
}

func TestTick_MultipleDueResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	rr := mock.NewResourceRepository(ctrl)
	topic := mock.NewTopic(ctrl)
	logger := slog.Default()

	s := New(rr, topic, logger)

	due := []*resource.ResourceWithPipeline{
		{
			Resource:      resource.Resource{ID: 1, Canonical: "cron.a", CheckInterval: "@every 1m"},
			TeamCanonical: "main",
			PipelineName:  "pp1",
		},
		{
			Resource:      resource.Resource{ID: 2, Canonical: "git.b", CheckInterval: "@every 5m"},
			TeamCanonical: "team2",
			PipelineName:  "pp2",
		},
	}

	rr.EXPECT().FilterDueResources(gomock.Any()).Return(due, nil)
	topic.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	rr.EXPECT().Update(gomock.Any(), "main", "pp1", "cron.a", gomock.Any()).Return(nil)
	rr.EXPECT().Update(gomock.Any(), "team2", "pp2", "git.b", gomock.Any()).Return(nil)

	s.tick(context.Background())
}

func TestTick_DefaultCheckInterval(t *testing.T) {
	ctrl := gomock.NewController(t)
	rr := mock.NewResourceRepository(ctrl)
	topic := mock.NewTopic(ctrl)
	logger := slog.Default()

	s := New(rr, topic, logger)

	due := []*resource.ResourceWithPipeline{
		{
			Resource:      resource.Resource{ID: 1, Canonical: "cron.x", CheckInterval: ""},
			TeamCanonical: "main",
			PipelineName:  "pp",
		},
	}

	rr.EXPECT().FilterDueResources(gomock.Any()).Return(due, nil)
	topic.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
	rr.EXPECT().Update(gomock.Any(), "main", "pp", "cron.x", gomock.Any()).DoAndReturn(
		func(ctx context.Context, tc, pn, rCan string, r resource.Resource) error {
			// Default is @every 1m
			diff := r.NextCheck.Sub(r.LastCheck)
			assert.InDelta(t, 1*time.Minute, diff, float64(2*time.Second))
			return nil
		},
	)

	s.tick(context.Background())
}

func TestStart_StopsOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	rr := mock.NewResourceRepository(ctrl)
	topic := mock.NewTopic(ctrl)
	logger := slog.Default()

	s := New(rr, topic, logger)
	s.interval = 50 * time.Millisecond

	rr.EXPECT().FilterDueResources(gomock.Any()).Return(nil, nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Let it tick a couple of times
	time.Sleep(150 * time.Millisecond)
	cancel()
	// Give goroutine time to exit
	time.Sleep(100 * time.Millisecond)
}

func TestTick_SendErrorSkipsResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	rr := mock.NewResourceRepository(ctrl)
	topic := mock.NewTopic(ctrl)
	logger := slog.Default()

	s := New(rr, topic, logger)

	due := []*resource.ResourceWithPipeline{
		{
			Resource:      resource.Resource{ID: 1, Canonical: "cron.fail", CheckInterval: "@every 1m"},
			TeamCanonical: "main",
			PipelineName:  "pp",
		},
	}

	rr.EXPECT().FilterDueResources(gomock.Any()).Return(due, nil)
	topic.EXPECT().Send(gomock.Any(), gomock.Any()).Return(assert.AnError)
	// Update should NOT be called because Send failed

	s.tick(context.Background())
}
