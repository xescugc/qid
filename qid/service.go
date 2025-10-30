package qid

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/awalterschulze/gographviz"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type Service interface {
	CreatePipeline(ctx context.Context, pn string, pp []byte, vars map[string]interface{}) error
	UpdatePipeline(ctx context.Context, pn string, pp []byte, vars map[string]interface{}) error
	GetPipeline(ctx context.Context, pn string) (*pipeline.Pipeline, error)
	DeletePipeline(ctx context.Context, pn string) error
	ListPipelines(ctx context.Context) ([]*pipeline.Pipeline, error)

	GetPipelineImage(ctx context.Context, pn, format string) ([]byte, error)
	CreatePipelineImage(ctx context.Context, pp []byte, vars map[string]interface{}, format string) ([]byte, error)

	TriggerPipelineJob(ctx context.Context, pn, jn string) error
	GetPipelineJob(ctx context.Context, pn, jn string) (*job.Job, error)

	CreateJobBuild(ctx context.Context, pn, jn string, b build.Build) (*build.Build, error)
	UpdateJobBuild(ctx context.Context, pn, jn string, bID uint32, b build.Build) error
	ListJobBuilds(ctx context.Context, pn, jn string) ([]*build.Build, error)

	GetPipelineResource(ctx context.Context, pn, rCan string) (*resource.Resource, error)
	UpdatePipelineResource(ctx context.Context, pn, rCan string, r resource.Resource) error
	TriggerPipelineResource(ctx context.Context, pn, rCan string) error
	CreateResourceVersion(ctx context.Context, pn, rCan string, v resource.Version) error
	ListResourceVersions(ctx context.Context, pn, rCan string) ([]*resource.Version, error)
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
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			level.Info(q.logger).Log("msg", "Checking for resources ....")
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
					// Default interval is 1m so if it's not set we'll check that
					ci := "1m"
					if r.CheckInterval != "" {
						ci = r.CheckInterval
					}
					d, err := time.ParseDuration(ci)
					if err != nil {
						level.Error(q.logger).Log("msg", "failed to parse CheckInterval", "CheckInterval", ci, "error", err.Error())
						continue
					}

					if time.Now().Sub(r.LastCheck) < d {
						// Not yet on the interval to check
						continue
					}
					for _, rt := range restypes {
						if r.Type == rt.Name {
							m := queue.Body{
								PipelineName:      pp.Name,
								ResourceCanonical: r.Canonical,
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
							_ = q.UpdatePipelineResource(ctx, pp.Name, r.Canonical, *r)
						}
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (q *Qid) CreatePipeline(ctx context.Context, pn string, rpp []byte, vars map[string]interface{}) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	pp, err := q.readPipeline(ctx, rpp, vars)
	if err != nil {
		return fmt.Errorf("failed to read Pipeline config: %w", err)
	}

	pp.Name = pn
	pp.Raw = rpp

	_, err = q.Pipelines.Create(ctx, *pp)
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

func (q *Qid) UpdatePipeline(ctx context.Context, pn string, rpp []byte, vars map[string]interface{}) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	pp, err := q.readPipeline(ctx, rpp, vars)
	if err != nil {
		return fmt.Errorf("failed to read Pipeline config: %w", err)
	}

	pp.Name = pn
	pp.Raw = rpp

	err = q.Pipelines.Update(ctx, pn, *pp)

	dbpp, err := q.GetPipeline(ctx, pn)
	if err != nil {
		return fmt.Errorf("failed to get Pipeline %q: %w", pn, err)
	}

	dbjbs := make(map[string]struct{})
	for _, j := range dbpp.Jobs {
		dbjbs[j.Name] = struct{}{}
	}
	for _, j := range pp.Jobs {
		if !utils.ValidateCanonical(j.Name) {
			return fmt.Errorf("invalid Job Name format %q", j.Name)
		}
		if _, ok := dbjbs[j.Name]; ok {
			delete(dbjbs, j.Name)
			err = q.Jobs.Update(ctx, pn, j.Name, j)
			if err != nil {
				return fmt.Errorf("failed to update Job %q: %w", j.Name, err)
			}
		} else {
			_, err = q.Jobs.Create(ctx, pn, j)
			if err != nil {
				return fmt.Errorf("failed to create Job %q: %w", j.Name, err)
			}
		}
	}
	for jn := range dbjbs {
		err = q.Jobs.Delete(ctx, pn, jn)
		if err != nil {
			return fmt.Errorf("failed to delete Job %q: %w", jn, err)
		}
	}

	dbrts := make(map[string]struct{})
	for _, rt := range dbpp.ResourceTypes {
		dbrts[rt.Name] = struct{}{}
	}
	for _, rt := range pp.ResourceTypes {
		if !utils.ValidateCanonical(rt.Name) {
			return fmt.Errorf("invalid ResourceType Name format %q", rt.Name)
		}
		if _, ok := dbrts[rt.Name]; ok {
			delete(dbrts, rt.Name)
			err = q.ResourceTypes.Update(ctx, pn, rt.Name, rt)
			if err != nil {
				return fmt.Errorf("failed to update ResourceType %q: %w", rt.Name, err)
			}
		} else {
			_, err = q.ResourceTypes.Create(ctx, pn, rt)
			if err != nil {
				return fmt.Errorf("failed to create ResourceType %q: %w", rt.Name, err)
			}
		}
	}
	for rt := range dbrts {
		err = q.ResourceTypes.Delete(ctx, pn, rt)
		if err != nil {
			return fmt.Errorf("failed to delete ResourceType %q: %w", rt, err)
		}
	}

	dbrs := make(map[string]struct{})
	for _, r := range dbpp.Resources {
		dbrs[r.Name] = struct{}{}
	}
	for _, r := range pp.Resources {
		if !utils.ValidateCanonical(r.Name) {
			return fmt.Errorf("invalid Resource Name format %q", r.Name)
		}
		if _, ok := dbrs[r.Name]; ok {
			delete(dbrs, r.Name)
			err = q.Resources.Update(ctx, pn, r.Canonical, r)
			if err != nil {
				return fmt.Errorf("failed to update Resource %q: %w", r.Name, err)
			}
		} else {
			_, err = q.Resources.Create(ctx, pn, r)
			if err != nil {
				return fmt.Errorf("failed to create Resource %q: %w", r.Name, err)
			}
		}
	}
	for rn := range dbrs {
		err = q.Resources.Delete(ctx, pn, rn)
		if err != nil {
			return fmt.Errorf("failed to delete Resource %q: %w", rn, err)
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

var (
	jobColors = map[build.Status]string{
		build.Started:   "6",
		build.Failed:    "1",
		build.Succeeded: "3",
	}
	colorscheme = "set19"
)

func (q *Qid) GetPipelineImage(ctx context.Context, pn, format string) ([]byte, error) {
	if format == "" {
		format = "dot"
	}
	if strings.Contains(format, ".") {
		format = strings.Split(format, ".")[1]
	}

	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if format != "dot" {
		return nil, fmt.Errorf("invalid image format %q", format)
	}

	pp, err := q.GetPipeline(ctx, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline %q: %w", pn, err)
	}

	img, err := q.generateImage(ctx, pp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	return img, err
}

func (q *Qid) generateImage(ctx context.Context, pp *pipeline.Pipeline) ([]byte, error) {
	var (
		pn  = pp.Name
		err error
	)

	graph := gographviz.NewGraph()
	graph.SetName(pn)
	graph.SetStrict(true)
	graph.AddAttr(pn, string(gographviz.RankDir), "LR")
	graph.AddAttr(pn, string(gographviz.ColorScheme), colorscheme)

	resourceColors := make(map[string]string)
	// Print all the resources
	for _, r := range pp.Resources {
		vurl := fmt.Sprintf(`"%s/pipelines/%s/resources/%s/versions"`, "http://localhost:4000", pp.Name, r.Canonical)
		color := "0"
		if r.Logs != "" {
			color = "1"
		}
		resourceColors[r.Canonical] = color
		err = graph.AddNode(pn, fmt.Sprintf(`"%s"`, r.Canonical), map[string]string{
			string(gographviz.Margin):      "0.1",
			string(gographviz.Shape):       "cds",
			string(gographviz.FillColor):   "9",
			string(gographviz.Style):       "filled",
			string(gographviz.FontColor):   "white",
			string(gographviz.ColorScheme): colorscheme,
			string(gographviz.URL):         vurl,
			string(gographviz.Color):       color,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add node to Graph: %w", err)
		}
	}

	// Print all the Jobs and the connection to resources
	for i, j := range pp.Jobs {
		jg := pn
		builds, err := q.Builds.Filter(ctx, pp.Name, j.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to filter builds from Job %q: %w", j.Name, err)
		}
		slices.Reverse(builds)
		color := "9"
		var (
			cb *build.Build
			pb *build.Build
		)
		if len(builds) != 0 {
			cb = builds[0]
			if len(builds) > 1 && cb.Status == build.Started {
				pb = builds[1]
				if c, ok := jobColors[pb.Status]; ok {
					color = c
				}
			}
		}

		if pb == nil && cb != nil && cb.Status != build.Started {
			if c, ok := jobColors[cb.Status]; ok {
				color = c
			}
		}

		style := "invis"
		if cb != nil && cb.Status == build.Started {
			style = `"dashed,bold"`
		}

		jg = fmt.Sprintf("cluster_%d", i)
		graph.AddSubGraph(pn, jg, map[string]string{
			string(gographviz.Style):       style,
			string(gographviz.Color):       jobColors[build.Started],
			string(gographviz.ColorScheme): colorscheme,
		})

		burl := fmt.Sprintf(`"%s/pipelines/%s/jobs/%s/builds"`, "http://localhost:4000", pp.Name, j.Name)
		err = graph.AddNode(jg, j.Name, map[string]string{
			string(gographviz.Margin):      "0.5",
			string(gographviz.Shape):       "rectangle",
			string(gographviz.FillColor):   color,
			string(gographviz.Style):       "filled",
			string(gographviz.FontColor):   "white",
			string(gographviz.ColorScheme): colorscheme,
			string(gographviz.URL):         burl,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add node to Graph: %w", err)
		}
		for _, g := range j.Get {
			if len(g.Passed) == 0 {
				rCan := fmt.Sprintf(`"%s.%s"`, g.Type, g.Name)
				opt := make(map[string]string)
				if g.Trigger {
					opt[string(gographviz.Style)] = "solid"
				} else {
					opt[string(gographviz.Style)] = "dashed"
				}
				err = graph.AddEdge(rCan, j.Name, false, opt)
				if err != nil {
					return nil, fmt.Errorf("failed to add edge to Graph: %w", err)
				}
			}
		}
	}

	// Now we print all the jobs interconnections depending on resources
	for _, j := range pp.Jobs {
		for _, g := range j.Get {
			if len(g.Passed) != 0 {
				for _, p := range g.Passed {
					nn := fmt.Sprintf(`"%s-%s-%s"`, p, g.Name, j.Name)
					rCan := fmt.Sprintf("%s.%s", g.Type, g.Name)
					vurl := fmt.Sprintf(`"%s/pipelines/%s/resources/%s/versions"`, "http://localhost:4000", pp.Name, rCan)
					color := resourceColors[rCan]
					err = graph.AddNode(pn, nn, map[string]string{
						string(gographviz.Label):       fmt.Sprintf(`"%s"`, rCan),
						string(gographviz.Margin):      "0.1",
						string(gographviz.Shape):       "cds",
						string(gographviz.FillColor):   "9",
						string(gographviz.Style):       "filled",
						string(gographviz.FontColor):   "white",
						string(gographviz.ColorScheme): colorscheme,
						string(gographviz.URL):         vurl,
						string(gographviz.Color):       color,
					})
					if err != nil {
						return nil, fmt.Errorf("failed to add node to Graph: %w", err)
					}
					err = graph.AddEdge(p, nn, false, nil)
					if err != nil {
						return nil, fmt.Errorf("failed to add edge to Graph: %w", err)
					}
					err = graph.AddEdge(nn, j.Name, false, nil)
					if err != nil {
						return nil, fmt.Errorf("failed to add edge to Graph: %w", err)
					}
				}
			}
		}
	}

	str := graph.String()
	// TODO: check for errors
	return []byte(str), nil
}

func (q *Qid) CreatePipelineImage(ctx context.Context, pipeline []byte, vars map[string]interface{}, format string) ([]byte, error) {
	pp, err := q.readPipeline(ctx, pipeline, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to read Pipeline: %w", err)
	}

	pp.Name = "qid"

	img, err := q.generateImage(ctx, pp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	return img, err
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

func (q *Qid) CreateResourceVersion(ctx context.Context, pn, rCan string, v resource.Version) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	_, err := q.Resources.CreateVersion(ctx, pn, rCan, v)
	if err != nil {
		return fmt.Errorf("failed to Create Resource Version: %w", err)
	}

	return nil
}

func (q *Qid) ListResourceVersions(ctx context.Context, pn, rCan string) ([]*resource.Version, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return nil, fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	rvers, err := q.Resources.FilterVersions(ctx, pn, rCan)
	if err != nil {
		return nil, fmt.Errorf("failed to List Resource Version: %w", err)
	}

	slices.Reverse(rvers)

	return rvers, nil
}

func (q *Qid) ListJobBuilds(ctx context.Context, pn, jn string) ([]*build.Build, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", pn)
	}

	builds, err := q.Builds.Filter(ctx, pn, jn)
	if err != nil {
		return nil, fmt.Errorf("failed to list Builds: %w", err)
	}

	slices.Reverse(builds)

	return builds, nil
}

func (q *Qid) GetPipelineResource(ctx context.Context, pn, rCan string) (*resource.Resource, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return nil, fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	r, err := q.Resources.Find(ctx, pn, rCan)
	if err != nil {
		return nil, fmt.Errorf("failed to find Resource: %w", err)
	}

	return r, nil
}

func (q *Qid) UpdatePipelineResource(ctx context.Context, pn, rCan string, r resource.Resource) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	err := q.Resources.Update(ctx, pn, rCan, r)
	if err != nil {
		return fmt.Errorf("failed to update Resource: %w", err)
	}

	return nil
}

func (q *Qid) TriggerPipelineResource(ctx context.Context, pn, rCan string) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	r, err := q.Resources.Find(ctx, pn, rCan)
	if err != nil {
		return fmt.Errorf("failed to find Resource: %w", err)
	}

	m := queue.Body{
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
	_ = q.UpdatePipelineResource(ctx, pn, r.Canonical, *r)

	return nil
}
