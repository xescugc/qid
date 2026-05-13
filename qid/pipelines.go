package qid

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/awalterschulze/gographviz"
	"github.com/go-kit/kit/log/level"
	cron "github.com/netresearch/go-cron"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"
)

func (q *Qid) newCronResourceFunc(tc, ppName, resCan string) func() {
	return func() {
		level.Info(q.logger).Log("msg", "Checking resource ...", "Pipeline", ppName, "Resource", resCan)
		r, err := q.Resources.Find(q.Ctx, tc, ppName, resCan)
		if err != nil {
			level.Error(q.logger).Log("msg", "failed to find Resource", "error", err.Error())
			return
		}
		m := queue.Body{
			TeamCanonical:     tc,
			PipelineName:      ppName,
			ResourceCanonical: resCan,
		}
		mb, err := json.Marshal(m)
		if err != nil {
			level.Error(q.logger).Log("msg", "failed to marshal Message Body", "error", err.Error())
			return
		}
		err = q.Topic.Send(q.Ctx, &pubsub.Message{
			Body: mb,
		})
		if err != nil {
			level.Error(q.logger).Log("msg", "failed to send Topic", "error", err.Error())
			return
		}
		r.LastCheck = time.Now()
		_ = q.UpdatePipelineResource(q.Ctx, tc, ppName, resCan, *r)
	}
}

func (q *Qid) CreatePipeline(ctx context.Context, tc, pn string, rpp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	pp, err := q.readPipeline(ctx, rpp, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to read Pipeline config: %w", err)
	}

	pp.Name = pn
	pp.Raw = rpp

	_, err = q.Pipelines.Create(ctx, tc, *pp)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pipeline %q: %w", pn, err)
	}

	for _, j := range pp.Jobs {
		if !utils.ValidateCanonical(j.Name) {
			return nil, fmt.Errorf("invalid Job Name format %q", j.Name)
		}
		_, err = q.Jobs.Create(ctx, tc, pn, j)
		if err != nil {
			return nil, fmt.Errorf("failed to create Job %q: %w", j.Name, err)
		}
	}

	for _, rt := range pp.ResourceTypes {
		if !utils.ValidateCanonical(rt.Name) {
			return nil, fmt.Errorf("invalid ResourceType Name format %q", rt.Name)
		}
		_, err = q.ResourceTypes.Create(ctx, tc, pn, rt)
		if err != nil {
			return nil, fmt.Errorf("failed to create ResourceType %q: %w", rt.Name, err)
		}
	}

	for _, r := range pp.Resources {
		if !utils.ValidateCanonical(r.Name) {
			return nil, fmt.Errorf("invalid Resource Name format %q", r.Name)
		}
		spec := r.CheckInterval
		if spec == "" {
			spec = "@every 1m"
		}
		eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(tc, pp.Name, r.Canonical))
		if err != nil {
			return nil, fmt.Errorf("failed to add Cron Func %q: %w", r.Name, err)
		}
		r.CronID = uint64(eid)
		_, err = q.Resources.Create(ctx, tc, pn, r)
		if err != nil {
			q.cron.Remove(eid)
			return nil, fmt.Errorf("failed to create Resource %q: %w", r.Name, err)
		}
	}

	for _, ru := range pp.Runners {
		if !utils.ValidateCanonical(ru.Name) {
			return nil, fmt.Errorf("invalid Runner Name format %q", ru.Name)
		}
		_, err = q.Runners.Create(ctx, tc, pn, ru)
		if err != nil {
			return nil, fmt.Errorf("failed to create Runner %q: %w", ru.Name, err)
		}
	}

	cp, err := q.GetPipeline(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline: %w", err)
	}
	return cp, nil
}

func (q *Qid) UpdatePipeline(ctx context.Context, tc, pn string, rpp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	pp, err := q.readPipeline(ctx, rpp, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to read Pipeline config: %w", err)
	}

	pp.Name = pn
	pp.Raw = rpp

	err = q.Pipelines.Update(ctx, tc, pn, *pp)

	dbpp, err := q.GetPipeline(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline %q: %w", pn, err)
	}

	dbjbs := make(map[string]struct{})
	for _, j := range dbpp.Jobs {
		dbjbs[j.Name] = struct{}{}
	}
	for _, j := range pp.Jobs {
		if !utils.ValidateCanonical(j.Name) {
			return nil, fmt.Errorf("invalid Job Name format %q", j.Name)
		}
		if _, ok := dbjbs[j.Name]; ok {
			delete(dbjbs, j.Name)
			err = q.Jobs.Update(ctx, tc, pn, j.Name, j)
			if err != nil {
				return nil, fmt.Errorf("failed to update Job %q: %w", j.Name, err)
			}
		} else {
			_, err = q.Jobs.Create(ctx, tc, pn, j)
			if err != nil {
				return nil, fmt.Errorf("failed to create Job %q: %w", j.Name, err)
			}
		}
	}
	for jn := range dbjbs {
		err = q.Jobs.Delete(ctx, tc, pn, jn)
		if err != nil {
			return nil, fmt.Errorf("failed to delete Job %q: %w", jn, err)
		}
	}

	dbrts := make(map[string]struct{})
	for _, rt := range dbpp.ResourceTypes {
		dbrts[rt.Name] = struct{}{}
	}
	for _, rt := range pp.ResourceTypes {
		if !utils.ValidateCanonical(rt.Name) {
			return nil, fmt.Errorf("invalid ResourceType Name format %q", rt.Name)
		}
		if _, ok := dbrts[rt.Name]; ok {
			delete(dbrts, rt.Name)
			err = q.ResourceTypes.Update(ctx, tc, pn, rt.Name, rt)
			if err != nil {
				return nil, fmt.Errorf("failed to update ResourceType %q: %w", rt.Name, err)
			}
		} else {
			_, err = q.ResourceTypes.Create(ctx, tc, pn, rt)
			if err != nil {
				return nil, fmt.Errorf("failed to create ResourceType %q: %w", rt.Name, err)
			}
		}
	}
	for rt := range dbrts {
		err = q.ResourceTypes.Delete(ctx, tc, pn, rt)
		if err != nil {
			return nil, fmt.Errorf("failed to delete ResourceType %q: %w", rt, err)
		}
	}

	dbrs := make(map[string]resource.Resource)
	for _, r := range dbpp.Resources {
		dbrs[r.Canonical] = r
	}
	for _, r := range pp.Resources {
		if !utils.ValidateCanonical(r.Name) {
			return nil, fmt.Errorf("invalid Resource Name format %q", r.Name)
		}
		if dbr, ok := dbrs[r.Canonical]; ok {
			delete(dbrs, r.Canonical)
			if dbr.CheckInterval != r.CheckInterval {
				q.cron.Remove(cron.EntryID(dbr.CronID))
				spec := r.CheckInterval
				if spec == "" {
					spec = "@every 1m"
				}
				eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(tc, pp.Name, r.Canonical))
				if err != nil {
					return nil, fmt.Errorf("failed to add Cron Func %q: %w", r.Canonical, err)
				}
				r.CronID = uint64(eid)
			}
			err = q.Resources.Update(ctx, tc, pn, r.Canonical, r)
			if err != nil {
				return nil, fmt.Errorf("failed to update Resource %q: %w", r.Canonical, err)
			}
		} else {
			q.cron.Remove(cron.EntryID(dbr.CronID))
			spec := r.CheckInterval
			if spec == "" {
				spec = "@every 1m"
			}
			eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(tc, pp.Name, r.Canonical))
			if err != nil {
				return nil, fmt.Errorf("failed to add Cron Func %q: %w", r.Canonical, err)
			}
			r.CronID = uint64(eid)
			_, err = q.Resources.Create(ctx, tc, pn, r)
			if err != nil {
				return nil, fmt.Errorf("failed to create Resource %q: %w", r.Canonical, err)
			}
		}
	}
	for rc := range dbrs {
		err = q.Resources.Delete(ctx, tc, pn, rc)
		if err != nil {
			return nil, fmt.Errorf("failed to delete Resource %q: %w", rc, err)
		}
	}

	dbru := make(map[string]struct{})
	for _, ru := range dbpp.Runners {
		dbru[ru.Name] = struct{}{}
	}
	for _, ru := range pp.Runners {
		if !utils.ValidateCanonical(ru.Name) {
			return nil, fmt.Errorf("invalid Resource Name format %q", ru.Name)
		}
		if _, ok := dbru[ru.Name]; ok {
			delete(dbru, ru.Name)
			err = q.Runners.Update(ctx, tc, pn, ru.Name, ru)
			if err != nil {
				return nil, fmt.Errorf("failed to update Runner %q: %w", ru.Name, err)
			}
		} else {
			_, err = q.Runners.Create(ctx, tc, pn, ru)
			if err != nil {
				return nil, fmt.Errorf("failed to create Runner %q: %w", ru.Name, err)
			}
		}
	}
	for run := range dbru {
		err = q.Runners.Delete(ctx, tc, pn, run)
		if err != nil {
			return nil, fmt.Errorf("failed to delete Runner %q: %w", run, err)
		}
	}
	up, err := q.GetPipeline(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline: %w", err)
	}

	return up, nil
}

func (q *Qid) ListPipelines(ctx context.Context, tc string) ([]*pipeline.Pipeline, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	}

	pps, err := q.Pipelines.Filter(ctx, tc)
	if err != nil {
		return nil, fmt.Errorf("failed to fitler Pipelines: %w", err)
	}

	for _, pp := range pps {
		jobs, err := q.Jobs.Filter(ctx, tc, pp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Jobs from Pipeline %q: %w", pp.Name, err)
		}

		resources, err := q.Resources.Filter(ctx, tc, pp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Resources from Pipeline %q: %w", pp.Name, err)
		}

		restypes, err := q.ResourceTypes.Filter(ctx, tc, pp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Resource Types from Pipeline %q: %w", pp.Name, err)
		}

		runners, err := q.Runners.Filter(ctx, tc, pp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get Runners from Pipeline %q: %w", pp.Name, err)
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

		for _, ru := range runners {
			pp.Runners = append(pp.Runners, *ru)
		}
	}

	return pps, nil
}

func (q *Qid) GetPipeline(ctx context.Context, tc, pn string) (*pipeline.Pipeline, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	pp, err := q.Pipelines.Find(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline %q: %w", pn, err)
	}

	jobs, err := q.Jobs.Filter(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Jobs from Pipeline %q: %w", pn, err)
	}

	resources, err := q.Resources.Filter(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Resources from Pipeline %q: %w", pn, err)
	}

	restypes, err := q.ResourceTypes.Filter(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Resource Types from Pipeline %q: %w", pn, err)
	}

	runners, err := q.Runners.Filter(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Runners from Pipeline %q: %w", pn, err)
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

	for _, ru := range runners {
		pp.Runners = append(pp.Runners, *ru)
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

func (q *Qid) GetPipelineImage(ctx context.Context, tc, pn, format string) ([]byte, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	}
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

	pp, err := q.GetPipeline(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline %q: %w", pn, err)
	}

	img, err := q.generateImage(ctx, tc, pp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	return img, err
}

func (q *Qid) generateImage(ctx context.Context, tc string, pp *pipeline.Pipeline) ([]byte, error) {
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
		vurl := fmt.Sprintf(`"/teams/%s/pipelines/%s/resources/%s/versions"`, tc, pp.Name, r.Canonical)
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
		builds, err := q.Builds.Filter(ctx, tc, pp.Name, j.Name)
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

		burl := fmt.Sprintf(`"/teams/%s/pipelines/%s/jobs/%s/builds"`, tc, pp.Name, j.Name)
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
					vurl := fmt.Sprintf(`"/teams/%s/pipelines/%s/resources/%s/versions"`, tc, pp.Name, rCan)
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

func (q *Qid) CreatePipelineImage(ctx context.Context, tc string, pipeline []byte, vars map[string]interface{}, format string) ([]byte, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	}

	pp, err := q.readPipeline(ctx, pipeline, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to read Pipeline: %w", err)
	}

	pp.Name = "qid"

	img, err := q.generateImage(ctx, tc, pp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	return img, err
}

func (q *Qid) DeletePipeline(ctx context.Context, tc, pn string) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	}
	resources, err := q.Resources.Filter(ctx, tc, pn)
	if err != nil {
		return fmt.Errorf("failed to get Resources from Pipeline %q: %w", pn, err)
	}

	// Removes all the Cron Resources
	for _, res := range resources {
		q.cron.Remove(cron.EntryID(res.CronID))
	}

	err = q.Pipelines.Delete(ctx, tc, pn)
	if err != nil {
		return fmt.Errorf("failed to delete Pipeline %q: %w", pn, err)
	}

	return nil
}
