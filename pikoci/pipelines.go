package pikoci

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/awalterschulze/gographviz"
	"github.com/google/uuid"
	cron "github.com/netresearch/go-cron"
	"github.com/xescugc/pikoci/pikoci/build"
	"github.com/xescugc/pikoci/pikoci/pipeline"
	"github.com/xescugc/pikoci/pikoci/queue"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/unitwork"
	"github.com/xescugc/pikoci/pikoci/utils"
	"gocloud.dev/pubsub"
)

func (q *PikoCI) newCronResourceFunc(tc, ppName, resCan string) func() {
	return func() {
		q.logger.Info("Checking resource ...", "Pipeline", ppName, "Resource", resCan)
		r, err := q.Resources.Find(q.Ctx, tc, ppName, resCan)
		if err != nil {
			q.logger.Error("failed to find Resource", "error", err)
			return
		}
		m := queue.Body{
			TeamCanonical:     tc,
			PipelineName:      ppName,
			ResourceCanonical: resCan,
		}
		mb, err := json.Marshal(m)
		if err != nil {
			q.logger.Error("failed to marshal Message Body", "error", err)
			return
		}
		err = q.Topic.Send(q.Ctx, &pubsub.Message{
			Body: mb,
		})
		if err != nil {
			q.logger.Error("failed to send Topic", "error", err)
			return
		}
		r.LastCheck = time.Now()
		_ = q.UpdatePipelineResource(q.Ctx, tc, ppName, resCan, *r)
	}
}

func (q *PikoCI) CreatePipeline(ctx context.Context, tc, pn string, rpp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error) {
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

	var cronEntries []cron.EntryID
	var cp *pipeline.Pipeline
	err = q.StartUoW(ctx, func(uow unitwork.UnitOfWork) error {
		_, err := uow.Pipelines().Create(ctx, tc, *pp)
		if err != nil {
			return fmt.Errorf("failed to create Pipeline %q: %w", pn, err)
		}

		for _, j := range pp.Jobs {
			if !utils.ValidateCanonical(j.Name) {
				return fmt.Errorf("invalid Job Name format %q", j.Name)
			}
			_, err = uow.Jobs().Create(ctx, tc, pn, j)
			if err != nil {
				return fmt.Errorf("failed to create Job %q: %w", j.Name, err)
			}
		}

		for _, rt := range pp.ResourceTypes {
			if !utils.ValidateCanonical(rt.Name) {
				return fmt.Errorf("invalid ResourceType Name format %q", rt.Name)
			}
			_, err = uow.ResourceTypes().Create(ctx, tc, pn, rt)
			if err != nil {
				return fmt.Errorf("failed to create ResourceType %q: %w", rt.Name, err)
			}
		}

		for _, r := range pp.Resources {
			if !utils.ValidateCanonical(r.Name) {
				return fmt.Errorf("invalid Resource Name format %q", r.Name)
			}
			spec := r.CheckInterval
			if spec == "" {
				spec = "@every 1m"
			}
			eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(tc, pp.Name, r.Canonical))
			if err != nil {
				return fmt.Errorf("failed to add Cron Func %q: %w", r.Name, err)
			}
			cronEntries = append(cronEntries, eid)
			r.CronID = uint64(eid)
			r.WebhookToken = uuid.New().String()
			_, err = uow.Resources().Create(ctx, tc, pn, r)
			if err != nil {
				q.cron.Remove(eid)
				return fmt.Errorf("failed to create Resource %q: %w", r.Name, err)
			}
		}

		for _, ru := range pp.Runners {
			if !utils.ValidateCanonical(ru.Name) {
				return fmt.Errorf("invalid Runner Name format %q", ru.Name)
			}
			_, err = uow.Runners().Create(ctx, tc, pn, ru)
			if err != nil {
				return fmt.Errorf("failed to create Runner %q: %w", ru.Name, err)
			}
		}

		cp, err = uow.Pipelines().Find(ctx, tc, pn)
		if err != nil {
			return fmt.Errorf("failed to get Pipeline: %w", err)
		}
		return nil
	})
	if err != nil {
		// Clean up any cron entries that were added
		for _, eid := range cronEntries {
			q.cron.Remove(eid)
		}
		return nil, err
	}
	return cp, nil
}

func (q *PikoCI) UpdatePipeline(ctx context.Context, tc, pn string, rpp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error) {
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

	var up *pipeline.Pipeline
	err = q.StartUoW(ctx, func(uow unitwork.UnitOfWork) error {
		err := uow.Pipelines().Update(ctx, tc, pn, *pp)
		if err != nil {
			return fmt.Errorf("failed to update Pipeline %q: %w", pn, err)
		}

		dbpp, err := uow.Pipelines().Find(ctx, tc, pn)
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
				err = uow.Jobs().Update(ctx, tc, pn, j.Name, j)
				if err != nil {
					return fmt.Errorf("failed to update Job %q: %w", j.Name, err)
				}
			} else {
				_, err = uow.Jobs().Create(ctx, tc, pn, j)
				if err != nil {
					return fmt.Errorf("failed to create Job %q: %w", j.Name, err)
				}
			}
		}
		for jn := range dbjbs {
			err = uow.Jobs().Delete(ctx, tc, pn, jn)
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
				err = uow.ResourceTypes().Update(ctx, tc, pn, rt.Name, rt)
				if err != nil {
					return fmt.Errorf("failed to update ResourceType %q: %w", rt.Name, err)
				}
			} else {
				_, err = uow.ResourceTypes().Create(ctx, tc, pn, rt)
				if err != nil {
					return fmt.Errorf("failed to create ResourceType %q: %w", rt.Name, err)
				}
			}
		}
		for rt := range dbrts {
			err = uow.ResourceTypes().Delete(ctx, tc, pn, rt)
			if err != nil {
				return fmt.Errorf("failed to delete ResourceType %q: %w", rt, err)
			}
		}

		dbrs := make(map[string]resource.Resource)
		for _, r := range dbpp.Resources {
			dbrs[r.Canonical] = r
		}
		for _, r := range pp.Resources {
			if !utils.ValidateCanonical(r.Name) {
				return fmt.Errorf("invalid Resource Name format %q", r.Name)
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
						return fmt.Errorf("failed to add Cron Func %q: %w", r.Canonical, err)
					}
					r.CronID = uint64(eid)
				}
				r.WebhookToken = dbr.WebhookToken
				err = uow.Resources().Update(ctx, tc, pn, r.Canonical, r)
				if err != nil {
					return fmt.Errorf("failed to update Resource %q: %w", r.Canonical, err)
				}
			} else {
				spec := r.CheckInterval
				if spec == "" {
					spec = "@every 1m"
				}
				eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(tc, pp.Name, r.Canonical))
				if err != nil {
					return fmt.Errorf("failed to add Cron Func %q: %w", r.Canonical, err)
				}
				r.CronID = uint64(eid)
				r.WebhookToken = uuid.New().String()
				_, err = uow.Resources().Create(ctx, tc, pn, r)
				if err != nil {
					return fmt.Errorf("failed to create Resource %q: %w", r.Canonical, err)
				}
			}
		}
		for rc := range dbrs {
			err = uow.Resources().Delete(ctx, tc, pn, rc)
			if err != nil {
				return fmt.Errorf("failed to delete Resource %q: %w", rc, err)
			}
		}

		dbru := make(map[string]struct{})
		for _, ru := range dbpp.Runners {
			dbru[ru.Name] = struct{}{}
		}
		for _, ru := range pp.Runners {
			if !utils.ValidateCanonical(ru.Name) {
				return fmt.Errorf("invalid Resource Name format %q", ru.Name)
			}
			if _, ok := dbru[ru.Name]; ok {
				delete(dbru, ru.Name)
				err = uow.Runners().Update(ctx, tc, pn, ru.Name, ru)
				if err != nil {
					return fmt.Errorf("failed to update Runner %q: %w", ru.Name, err)
				}
			} else {
				_, err = uow.Runners().Create(ctx, tc, pn, ru)
				if err != nil {
					return fmt.Errorf("failed to create Runner %q: %w", ru.Name, err)
				}
			}
		}
		for run := range dbru {
			err = uow.Runners().Delete(ctx, tc, pn, run)
			if err != nil {
				return fmt.Errorf("failed to delete Runner %q: %w", run, err)
			}
		}

		up, err = uow.Pipelines().Find(ctx, tc, pn)
		if err != nil {
			return fmt.Errorf("failed to get Pipeline: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return up, nil
}

func (q *PikoCI) ListPipelines(ctx context.Context, tc string) ([]*pipeline.Pipeline, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	}

	pps, err := q.Pipelines.Filter(ctx, tc)
	if err != nil {
		return nil, fmt.Errorf("failed to filter Pipelines: %w", err)
	}

	return pps, nil
}

func (q *PikoCI) GetPipeline(ctx context.Context, tc, pn string) (*pipeline.Pipeline, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	pp, err := q.Pipelines.Find(ctx, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Pipeline %q: %w", pn, err)
	}

	return pp, nil
}

var (
	jobColors = map[build.Status]string{
		build.Started:   `"#FFA300"`,
		build.Failed:    `"#FF004D"`,
		build.Succeeded: `"#00A83A"`,
	}
	jobBorderColors = map[build.Status]string{
		build.Started:   `"#CC8200"`,
		build.Failed:    `"#CC003E"`,
		build.Succeeded: `"#008030"`,
	}
	colorResource       = `"#83769C"`
	colorResourceBorder = `"#5F574F"`
	colorDefault        = `"#83769C"`
	colorDefaultBorder  = `"#5F574F"`
	colorError          = `"#FF004D"`
)

func (q *PikoCI) GetPipelineImage(ctx context.Context, tc, pn, format string) ([]byte, error) {
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

func (q *PikoCI) generateImage(ctx context.Context, tc string, pp *pipeline.Pipeline) ([]byte, error) {
	var (
		pn  = pp.Name
		err error
	)

	graph := gographviz.NewGraph()
	graph.SetName(pn)
	graph.SetStrict(true)
	graph.AddAttr(pn, string(gographviz.RankDir), "LR")

	resourceBorders := make(map[string]string)
	// Print all the resources
	for _, r := range pp.Resources {
		vurl := fmt.Sprintf(`"/teams/%s/pipelines/%s/resources/%s/versions"`, tc, pp.Name, r.Canonical)
		borderColor := colorResourceBorder
		if r.Logs != "" {
			borderColor = colorError
		}
		resourceBorders[r.Canonical] = borderColor
		err = graph.AddNode(pn, fmt.Sprintf(`"%s"`, r.Canonical), map[string]string{
			string(gographviz.Margin):    "0.2",
			string(gographviz.Shape):     "cds",
			string(gographviz.FillColor): colorResource,
			string(gographviz.Style):     "filled",
			string(gographviz.FontColor): "white",
			string(gographviz.URL):       vurl,
			string(gographviz.Color):     borderColor,
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
		color := colorDefault
		borderColor := colorDefaultBorder
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
				if c, ok := jobBorderColors[pb.Status]; ok {
					borderColor = c
				}
			}
		}

		if pb == nil && cb != nil && cb.Status != build.Started {
			if c, ok := jobColors[cb.Status]; ok {
				color = c
			}
			if c, ok := jobBorderColors[cb.Status]; ok {
				borderColor = c
			}
		}

		style := "invis"
		if cb != nil && cb.Status == build.Started {
			style = `"dashed,bold"`
		}

		jg = fmt.Sprintf("cluster_%d", i)
		graph.AddSubGraph(pn, jg, map[string]string{
			string(gographviz.Style): style,
			string(gographviz.Color): jobBorderColors[build.Started],
		})

		burl := fmt.Sprintf(`"/teams/%s/pipelines/%s/jobs/%s/builds"`, tc, pp.Name, j.Name)
		err = graph.AddNode(jg, j.Name, map[string]string{
			string(gographviz.Margin):    "0.5",
			string(gographviz.Shape):     "rectangle",
			string(gographviz.FillColor): color,
			string(gographviz.Style):     "filled",
			string(gographviz.FontColor): "white",
			string(gographviz.Color):     borderColor,
			string(gographviz.URL):       burl,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add node to Graph: %w", err)
		}
		// Draw resource→job edges for get steps without passed constraints
		for _, g := range j.GetSteps() {
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
		// Draw job→resource edges for put steps
		for _, ps := range j.Plan {
			if ps.Type == "put" && ps.Put != nil {
				rCan := fmt.Sprintf(`"%s.%s"`, ps.Put.Type, ps.Put.Name)
				err = graph.AddEdge(j.Name, rCan, false, map[string]string{
					string(gographviz.Style): "solid",
				})
				if err != nil {
					return nil, fmt.Errorf("failed to add edge to Graph: %w", err)
				}
			}
		}
	}

	// Now we print all the jobs interconnections depending on resources
	for _, j := range pp.Jobs {
		for _, g := range j.GetSteps() {
			if len(g.Passed) != 0 {
				for _, p := range g.Passed {
					nn := fmt.Sprintf(`"%s-%s-%s"`, p, g.Name, j.Name)
					rCan := fmt.Sprintf("%s.%s", g.Type, g.Name)
					vurl := fmt.Sprintf(`"/teams/%s/pipelines/%s/resources/%s/versions"`, tc, pp.Name, rCan)
					border := resourceBorders[rCan]
					err = graph.AddNode(pn, nn, map[string]string{
						string(gographviz.Label):     fmt.Sprintf(`"%s"`, rCan),
						string(gographviz.Margin):    "0.2",
						string(gographviz.Shape):     "cds",
						string(gographviz.FillColor): colorResource,
						string(gographviz.Style):     "filled",
						string(gographviz.FontColor): "white",
						string(gographviz.URL):       vurl,
						string(gographviz.Color):     border,
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
	return []byte(str), nil
}

func (q *PikoCI) CreatePipelineImage(ctx context.Context, tc string, pipeline []byte, vars map[string]interface{}, format string) ([]byte, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	}

	pp, err := q.readPipeline(ctx, pipeline, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to read Pipeline: %w", err)
	}

	pp.Name = "pikoci"

	img, err := q.generateImage(ctx, tc, pp)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	return img, err
}

func (q *PikoCI) DeletePipeline(ctx context.Context, tc, pn string) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	}

	return q.StartUoW(ctx, func(uow unitwork.UnitOfWork) error {
		resources, err := uow.Resources().Filter(ctx, tc, pn)
		if err != nil {
			return fmt.Errorf("failed to get Resources from Pipeline %q: %w", pn, err)
		}

		// Removes all the Cron Resources
		for _, res := range resources {
			q.cron.Remove(cron.EntryID(res.CronID))
		}

		err = uow.Pipelines().Delete(ctx, tc, pn)
		if err != nil {
			return fmt.Errorf("failed to delete Pipeline %q: %w", pn, err)
		}

		return nil
	})
}
