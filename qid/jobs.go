package qid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"
)

func (q *Qid) TriggerPipelineJob(ctx context.Context, tc, pn, jn string) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", jn)
	}

	_, err := q.Jobs.Find(ctx, tc, pn, jn)
	if err != nil {
		return fmt.Errorf("failed to Find Job %q on Pipeline %q: %w", jn, pn, err)
	}

	m := queue.Body{
		TeamCanonical: tc,
		PipelineName:  pn,
		JobName:       jn,
	}

	mb, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal Message Body: %w", err)
	}

	err = q.Topic.Send(ctx, &pubsub.Message{
		Body: mb,
	})
	if err != nil {
		return fmt.Errorf("failed to Trigger Job %q on Pipeline %q: %w", jn, pn, err)
	}

	return nil
}

func (q *Qid) GetPipelineJob(ctx context.Context, tc, pn, jn string) (*job.Job, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", jn)
	}

	j, err := q.Jobs.Find(ctx, tc, pn, jn)
	if err != nil {
		return nil, fmt.Errorf("failed to Find Job %q on Pipeline %q: %w", jn, pn, err)
	}

	return j, nil
}
