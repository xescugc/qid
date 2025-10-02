package qid

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"

	"github.com/go-kit/kit/log"
)

type Service interface {
	CreatePipeline(ctx context.Context, pn string, pp []byte) error
	GetPipeline(ctx context.Context, pn string) (*pipeline.Pipeline, error)
	DeletePipeline(ctx context.Context, pn string) error
	ListPipelines(ctx context.Context) ([]*pipeline.Pipeline, error)

	TriggerPipelineJob(ctx context.Context, pn, jn string) error
	GetPipelineJob(ctx context.Context, pn, jn string) (*job.Job, error)

	CreateJobBuild(ctx context.Context, pn, jn string, b build.Build) (*build.Build, error)
	UpdateJobBuild(ctx context.Context, pn, jn string, bID uint32, b build.Build) error

	CreateResourceVersion(ctx context.Context, pn, rn, rt string, v resource.Version) error
	ListResourceVersions(ctx context.Context, pn, rn, rt string) ([]*resource.Version, error)
}

type Qid struct {
	Topic         queue.Topic
	Pipelines     pipeline.Repository
	Jobs          job.Repository
	Resources     resource.Repository
	ResourceTypes restype.Repository
	Builds        build.Repository

	logger log.Logger
}

func New(ctx context.Context, t queue.Topic, pr pipeline.Repository, jr job.Repository, rr resource.Repository, rt restype.Repository, br build.Repository, l log.Logger) *Qid {
	q := &Qid{
		Topic:         t,
		Pipelines:     pr,
		Jobs:          jr,
		Resources:     rr,
		ResourceTypes: rt,
		Builds:        br,
		logger:        l,
	}

	go q.resourceCheck(ctx)

	return q
}

func (q *Qid) resourceCheck(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			q.logger.Log("msg", "Checking for resources ....")
			pps, _ := q.Pipelines.Filter(ctx)
			for _, pp := range pps {
				resources, err := q.Resources.Filter(ctx, pp.Name)
				if err != nil {
					//return nil, fmt.Errorf("failed to get Resources from Pipeline %q: %w", pn, err)
				}

				restypes, err := q.ResourceTypes.Filter(ctx, pp.Name)
				if err != nil {
					//return nil, fmt.Errorf("failed to get Resource Types from Pipeline %q: %w", pn, err)
				}
				for _, r := range resources {
					for _, rt := range restypes {
						if r.Type == rt.Name {
							m := queue.Body{
								PipelineName: pp.Name,
								ResourceName: r.Name,
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
						}
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (q *Qid) CreatePipeline(ctx context.Context, pn string, rpp []byte) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	}
	var pp pipeline.Pipeline
	err := hclsimple.Decode("pipeline.hcl", rpp, nil, &pp)
	if err != nil {
		return fmt.Errorf("failed to Decode Pipeline config: %w", err)
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

	for _, rt := range pp.ResourceTypes {
		if !utils.ValidateCanonical(rt.Name) {
			return fmt.Errorf("invalid ResourceType Name format %q", rt.Name)
		}
		_, err = q.ResourceTypes.Create(ctx, pn, rt)
		if err != nil {
			return fmt.Errorf("failed to create ResourceType %q: %w", rt.Name, err)
		}
	}

	for _, r := range pp.Resources {
		if !utils.ValidateCanonical(r.Name) {
			return fmt.Errorf("invalid Resource Name format %q", r.Name)
		}
		_, err = q.Resources.Create(ctx, pn, r)
		if err != nil {
			return fmt.Errorf("failed to create Resource %q: %w", r.Name, err)
		}
	}
	return nil
}

func (q *Qid) ListPipelines(ctx context.Context) ([]*pipeline.Pipeline, error) {
	pps, err := q.Pipelines.Filter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fitler Pipelines: %w", err)
	}

	for _, pp := range pps {
		jobs, err := q.Jobs.Filter(ctx, pp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Jobs from Pipeline %q: %w", pp.Name, err)
		}

		resources, err := q.Resources.Filter(ctx, pp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Resources from Pipeline %q: %w", pp.Name, err)
		}

		restypes, err := q.ResourceTypes.Filter(ctx, pp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Resource Types from Pipeline %q: %w", pp.Name, err)
		}

		for _, j := range jobs {
			pp.Jobs = append(pp.Jobs, *j)
		}

		for _, r := range resources {
			pp.Resources = append(pp.Resources, *r)
		}

		for _, rt := range restypes {
			pp.ResourceTypes = append(pp.ResourceTypes, *rt)
		}
	}

	return pps, nil
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

	resources, err := q.Resources.Filter(ctx, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Resources from Pipeline %q: %w", pn, err)
	}

	restypes, err := q.ResourceTypes.Filter(ctx, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Resource Types from Pipeline %q: %w", pn, err)
	}

	for _, j := range jobs {
		pp.Jobs = append(pp.Jobs, *j)
	}

	for _, r := range resources {
		pp.Resources = append(pp.Resources, *r)
	}

	for _, rt := range restypes {
		pp.ResourceTypes = append(pp.ResourceTypes, *rt)
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

func (q *Qid) TriggerPipelineJob(ctx context.Context, pn, jn string) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", jn)
	}

	_, err := q.Jobs.Find(ctx, pn, jn)
	if err != nil {
		return fmt.Errorf("failed to Find Job %q on Pipeline %q: %w", jn, pn, err)
	}

	m := queue.Body{
		PipelineName: pn,
		JobName:      jn,
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

func (q *Qid) GetPipelineJob(ctx context.Context, pn, jn string) (*job.Job, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", jn)
	}

	j, err := q.Jobs.Find(ctx, pn, jn)
	if err != nil {
		return nil, fmt.Errorf("failed to Find Job %q on Pipeline %q: %w", jn, pn, err)
	}

	return j, nil
}

func (q *Qid) CreateJobBuild(ctx context.Context, pn, jn string, b build.Build) (*build.Build, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", pn)
	}

	id, err := q.Builds.Create(ctx, pn, jn, b)
	if err != nil {
		return nil, fmt.Errorf("failed to Create Build: %w", err)
	}

	b.ID = id

	return &b, nil
}

func (q *Qid) UpdateJobBuild(ctx context.Context, pn, jn string, bID uint32, b build.Build) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", pn)
	}

	err := q.Builds.Update(ctx, pn, jn, bID, b)
	if err != nil {
		return fmt.Errorf("failed to Update Build: %w", err)
	}

	return nil
}

func (q *Qid) CreateResourceVersion(ctx context.Context, pn, rt, rn string, v resource.Version) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(rt) {
		return fmt.Errorf("invalid Resource Type format %q", rt)
	} else if !utils.ValidateCanonical(rn) {
		return fmt.Errorf("invalid Resource Name format %q", rn)
	}

	_, err := q.Resources.CreateVersion(ctx, pn, rt, rn, v)
	if err != nil {
		return fmt.Errorf("failed to Create Resource Version: %w", err)
	}

	return nil
}
func (q *Qid) ListResourceVersions(ctx context.Context, pn, rt, rn string) ([]*resource.Version, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(rt) {
		return nil, fmt.Errorf("invalid Resource Type format %q", rt)
	} else if !utils.ValidateCanonical(rn) {
		return nil, fmt.Errorf("invalid Resource Name format %q", rn)
	}

	rvers, err := q.Resources.FilterVersions(ctx, pn, rt, rn)
	if err != nil {
		return nil, fmt.Errorf("failed to List Resource Version: %w", err)
	}

	return rvers, nil
}
