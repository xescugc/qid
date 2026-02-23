package qid

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"
)

func (q *Qid) CreateResourceVersion(ctx context.Context, tc, pn, rCan string, v resource.Version) (*resource.Version, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return nil, fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	id, err := q.Resources.CreateVersion(ctx, tc, pn, rCan, v)
	if err != nil {
		return nil, fmt.Errorf("failed to Create Resource Version: %w", err)
	}

	v.ID = id

	return &v, nil
}

func (q *Qid) ListResourceVersions(ctx context.Context, tc, pn, rCan string) ([]*resource.Version, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return nil, fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	rvers, err := q.Resources.FilterVersions(ctx, tc, pn, rCan)
	if err != nil {
		return nil, fmt.Errorf("failed to List Resource Version: %w", err)
	}

	slices.Reverse(rvers)

	return rvers, nil
}

func (q *Qid) GetPipelineResource(ctx context.Context, tc, pn, rCan string) (*resource.Resource, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return nil, fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	r, err := q.Resources.Find(ctx, tc, pn, rCan)
	if err != nil {
		return nil, fmt.Errorf("failed to find Resource: %w", err)
	}

	return r, nil
}

func (q *Qid) UpdatePipelineResource(ctx context.Context, tc, pn, rCan string, r resource.Resource) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	err := q.Resources.Update(ctx, tc, pn, rCan, r)
	if err != nil {
		return fmt.Errorf("failed to update Resource: %w", err)
	}

	return nil
}

func (q *Qid) TriggerPipelineResource(ctx context.Context, tc, pn, rCan string) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	r, err := q.Resources.Find(ctx, tc, pn, rCan)
	if err != nil {
		return fmt.Errorf("failed to find Resource: %w", err)
	}

	m := queue.Body{
		TeamCanonical:     tc,
		PipelineName:      pn,
		ResourceCanonical: rCan,
	}
	mb, err := json.Marshal(m)
	if err != nil {
		//return fmt.Errorf("failed to marshal Message Body: %w", err)
	}
	err = q.Topic.Send(ctx, &pubsub.Message{
		Body: mb,
	})
	if err != nil {
		//return fmt.Errorf("failed to Trigger Queue on Pipeline %q: %w", pn, err)
	}
	r.LastCheck = time.Now()
	_ = q.UpdatePipelineResource(ctx, tc, pn, r.Canonical, *r)

	return nil
}
