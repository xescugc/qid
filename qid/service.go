package qid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"
)

type Service interface {
	CreatePipeline(ctx context.Context, pn string, pp []byte) error
	GetPipeline(ctx context.Context, pn string) (*pipeline.Pipeline, error)
	DeletePipeline(ctx context.Context, pn string) error
	TriggerPipelineJob(ctx context.Context, ppn, jn string) error
	GetPipelineJob(ctx context.Context, ppn, jn string) (*job.Job, error)
}

type Qid struct {
	Topic     queue.Topic
	Pipelines pipeline.Repository
	Jobs      job.Repository
}

func New(t queue.Topic, pr pipeline.Repository, jr job.Repository) *Qid {
	return &Qid{
		Topic:     t,
		Pipelines: pr,
		Jobs:      jr,
	}
}

func (q *Qid) CreatePipeline(ctx context.Context, pn string, rpp []byte) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	}
	var pp pipeline.Pipeline
	err := json.Unmarshal(rpp, &pp)
	if err != nil {
		return fmt.Errorf("failed to Unmarshal Pipeline config: %w", err)
	}

	pp.Name = pn

	_, err = q.Pipelines.Create(ctx, pp)
	if err != nil {
		return fmt.Errorf("failed to create Pipeline %q: %w", pn, err)
	}

	for _, j := range pp.Jobs {
		if !utils.ValidateCanonical(j.Name) {
			return fmt.Errorf("invalid Job Name format %q", j.Name)
		}
		_, err = q.Jobs.Create(ctx, pn, j)
		if err != nil {
			return fmt.Errorf("failed to create Job %q: %w", j.Name, err)
		}
	}
	return nil
}

func (q *Qid) GetPipeline(ctx context.Context, pn string) (*pipeline.Pipeline, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	pp, err := q.Pipelines.Find(ctx, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline %q: %w", pn, err)
	}

	jobs, err := q.Jobs.Filter(ctx, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Jobs from Pipeline %q: %w", pn, err)
	}

	for _, j := range jobs {
		pp.Jobs = append(pp.Jobs, *j)
	}

	return pp, nil
}

func (q *Qid) DeletePipeline(ctx context.Context, pn string) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	err := q.Pipelines.Delete(ctx, pn)
	if err != nil {
		return fmt.Errorf("failed to delete Pipeline %q: %w", pn, err)
	}

	return nil
}

func (q *Qid) TriggerPipelineJob(ctx context.Context, ppn, jn string) error {
	if !utils.ValidateCanonical(ppn) {
		return fmt.Errorf("invalid Pipeline Name format %q", ppn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", jn)
	}

	_, err := q.Jobs.Find(ctx, ppn, jn)
	if err != nil {
		return fmt.Errorf("failed to Find Job %q on Pipeline %q: %w", jn, ppn, err)
	}

	err = q.Topic.Send(ctx, &pubsub.Message{
		Metadata: map[string]string{
			"pipeline_name": ppn,
			"job_name":      jn,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to Trigger Job %q on Pipeline %q: %w", jn, ppn, err)
	}

	return nil
}

func (q *Qid) GetPipelineJob(ctx context.Context, ppn, jn string) (*job.Job, error) {
	if !utils.ValidateCanonical(ppn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", ppn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", jn)
	}

	j, err := q.Jobs.Find(ctx, ppn, jn)
	if err != nil {
		return nil, fmt.Errorf("failed to Find Job %q on Pipeline %q: %w", jn, ppn, err)
	}

	return j, nil
}
