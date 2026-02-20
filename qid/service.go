package qid

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/awalterschulze/gographviz"
	cron "github.com/netresearch/go-cron"
	"github.com/xescugc/qid/qid/build"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/runner"
	"github.com/xescugc/qid/qid/team"
	"github.com/xescugc/qid/qid/user"
	"github.com/xescugc/qid/qid/utils"
	"gocloud.dev/pubsub"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type Service interface {
	UserLogin(ctx context.Context, un, pass string) (*user.User, error)

	GetUser(ctx context.Context, un string) (*user.WithMemberships, error)
	CreateUser(ctx context.Context, u user.User, isHash bool) (*user.User, error)
	ListUsers(ctx context.Context) ([]*user.User, error)

	CreateTeam(ctx context.Context, un string, t team.Team) (*team.WithMembers, error)
	ListTeams(ctx context.Context, un string) ([]*team.WithMembers, error)
	GetTeam(ctx context.Context, un, tc string) (*team.WithMembers, error)
	UpdateTeam(ctx context.Context, un, tc string, t team.Team) (*team.WithMembers, error)
	DeleteTeam(ctx context.Context, un, tc string) error

	CreateTeamMember(ctx context.Context, un, tc string, tm team.Member) (*team.Member, error)
	UpdateTeamMember(ctx context.Context, un, tc, mc string, tm team.Member) (*team.Member, error)
	DeleteTeamMember(ctx context.Context, un, tc, mc string) error

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
	DeleteJobBuild(ctx context.Context, pn, jn string, bID uint32) error
	ListJobBuilds(ctx context.Context, pn, jn string) ([]*build.Build, error)

	GetPipelineResource(ctx context.Context, pn, rCan string) (*resource.Resource, error)
	UpdatePipelineResource(ctx context.Context, pn, rCan string, r resource.Resource) error
	TriggerPipelineResource(ctx context.Context, pn, rCan string) error
	CreateResourceVersion(ctx context.Context, pn, rCan string, v resource.Version) (*resource.Version, error)
	ListResourceVersions(ctx context.Context, pn, rCan string) ([]*resource.Version, error)
}

type Qid struct {
	Topic         queue.Topic
	Users         user.Repository
	Teams         team.Repository
	Pipelines     pipeline.Repository
	Jobs          job.Repository
	Resources     resource.Repository
	ResourceTypes restype.Repository
	Builds        build.Repository
	Runners       runner.Repository
	Ctx           context.Context

	cron   *cron.Cron
	logger log.Logger
}

func New(ctx context.Context, t queue.Topic, ur user.Repository, tr team.Repository, pr pipeline.Repository, jr job.Repository, rr resource.Repository, rt restype.Repository, br build.Repository, rur runner.Repository, l log.Logger) *Qid {
	q := &Qid{
		Topic:         t,
		Users:         ur,
		Teams:         tr,
		Pipelines:     pr,
		Jobs:          jr,
		Resources:     rr,
		ResourceTypes: rt,
		Builds:        br,
		Runners:       rur,
		Ctx:           ctx,
		logger:        l,
		cron:          cron.New(cron.WithContext(ctx)),
	}

	q.cron.Start()

	return q
}

func (q *Qid) newCronResourceFunc(ppName, resCan string) func() {
	return func() {
		level.Info(q.logger).Log("msg", "Checking resource ...", "Pipeline", ppName, "Resource", resCan)
		r, err := q.Resources.Find(q.Ctx, ppName, resCan)
		if err != nil {
			level.Error(q.logger).Log("msg", "failed to find Resource", "error", err.Error())
			return
		}
		m := queue.Body{
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
		_ = q.UpdatePipelineResource(q.Ctx, ppName, resCan, *r)
	}
}

func (q *Qid) UserLogin(ctx context.Context, un, pass string) (*user.User, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	}
	u, err := q.Users.Find(ctx, un)
	if err != nil {
		return nil, fmt.Errorf("failed to Find User: %w", err)
	}

	ok := utils.CheckPasswordHash(pass, u.Password)
	if !ok {
		return nil, fmt.Errorf("Username or Password is wrong")
	}

	return u, nil
}

func (q *Qid) GetUser(ctx context.Context, un string) (*user.WithMemberships, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	}

	um, err := q.Users.FindWithMemberships(ctx, un)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return um, nil
}

func (q *Qid) CreateUser(ctx context.Context, u user.User, isHash bool) (*user.User, error) {
	if !utils.ValidateCanonical(u.Username) {
		return nil, fmt.Errorf("invalid Username format %q", u.Username)
	} else if u.Password == "" {
		return nil, fmt.Errorf("invalid empty Password")
	}

	if !isHash {
		hash, err := utils.HashPassword(u.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash Passowrd: %w", err)
		}
		u.Password = hash
	}

	id, err := q.Users.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("failed to Create User: %w", err)
	}
	u.ID = id

	return &u, nil
}

func (q *Qid) ListUsers(ctx context.Context) ([]*user.User, error) {
	us, err := q.Users.Filter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to Find User: %w", err)
	}

	return us, nil
}

func (q *Qid) CreateTeam(ctx context.Context, un string, t team.Team) (*team.WithMembers, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	} else if t.Name == "" {
		return nil, fmt.Errorf("Team Name is required")
	}

	t.Canonical = utils.Canonicalize(t.Name)

	id, err := q.Teams.Create(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("failed to create Team: %w", err)
	}
	t.ID = id

	err = q.Teams.CreateMember(ctx, t.Canonical, team.Member{
		Admin: true,
		User: user.User{
			Username: un,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Team Member: %w", err)
	}

	twm, err := q.Teams.Find(ctx, t.Canonical)
	if err != nil {
		return nil, fmt.Errorf("failed to find Team: %w", err)
	}

	return twm, nil
}

func (q *Qid) ListTeams(ctx context.Context, un string) ([]*team.WithMembers, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	}

	teams, err := q.Teams.Filter(ctx, un)
	if err != nil {
		return nil, fmt.Errorf("failed to list Teams: %w", err)
	}

	return teams, nil
}

func (q *Qid) GetTeam(ctx context.Context, un, tc string) (*team.WithMembers, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	} else if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	}

	t, err := q.Teams.Find(ctx, tc)
	if err != nil {
		return nil, fmt.Errorf("failed to get Team: %w", err)
	}

	return t, nil
}

func (q *Qid) UpdateTeam(ctx context.Context, un, tc string, t team.Team) (*team.WithMembers, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	} else if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	}

	t.Canonical = utils.Canonicalize(t.Name)

	err := q.Teams.Update(ctx, tc, t)
	if err != nil {
		return nil, fmt.Errorf("failed to update Team: %w", err)
	}

	twm, err := q.Teams.Find(ctx, t.Canonical)
	if err != nil {
		return nil, fmt.Errorf("failed to find Team: %w", err)
	}

	return twm, nil
}

func (q *Qid) DeleteTeam(ctx context.Context, un, tc string) error {
	if !utils.ValidateCanonical(un) {
		return fmt.Errorf("invalid Username format %q", un)
	} else if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	}

	err := q.Teams.Delete(ctx, tc)
	if err != nil {
		return fmt.Errorf("failed to delete Team: %w", err)
	}

	return nil
}

func (q *Qid) CreateTeamMember(ctx context.Context, un, tc string, tm team.Member) (*team.Member, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	} else if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(tm.User.Username) {
		return nil, fmt.Errorf("invalid Team Member Username format %q", tm.User.Username)
	}

	err := q.Teams.CreateMember(ctx, tc, tm)
	if err != nil {
		return nil, fmt.Errorf("failed to create member: %w", err)
	}

	rtm, err := q.Teams.FindMember(ctx, tc, tm.User.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to find member: %w", err)
	}

	return rtm, nil
}

func (q *Qid) UpdateTeamMember(ctx context.Context, un, tc, mu string, tm team.Member) (*team.Member, error) {
	if !utils.ValidateCanonical(un) {
		return nil, fmt.Errorf("invalid Username format %q", un)
	} else if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(mu) {
		return nil, fmt.Errorf("invalid Team Member Username format %q", mu)
	}

	err := q.Teams.UpdateMember(ctx, tc, mu, tm)
	if err != nil {
		return nil, fmt.Errorf("failed to create member: %w", err)
	}

	rtm, err := q.Teams.FindMember(ctx, tc, mu)
	if err != nil {
		return nil, fmt.Errorf("failed to find member: %w", err)
	}

	return rtm, nil
}

func (q *Qid) DeleteTeamMember(ctx context.Context, un, tc, mc string) error {
	if !utils.ValidateCanonical(un) {
		return fmt.Errorf("invalid Username format %q", un)
	} else if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(mc) {
		return fmt.Errorf("invalid Team Member Username format %q", tc)
	}

	err := q.Teams.DeleteMember(ctx, tc, mc)
	if err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}

	return nil
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
		spec := r.CheckInterval
		if spec == "" {
			spec = "@every 1m"
		}
		eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(pp.Name, r.Canonical))
		if err != nil {
			return fmt.Errorf("failed to add Cron Func %q: %w", r.Name, err)
		}
		r.CronID = uint64(eid)
		_, err = q.Resources.Create(ctx, pn, r)
		if err != nil {
			q.cron.Remove(eid)
			return fmt.Errorf("failed to create Resource %q: %w", r.Name, err)
		}
	}

	for _, ru := range pp.Runners {
		if !utils.ValidateCanonical(ru.Name) {
			return fmt.Errorf("invalid Runner Name format %q", ru.Name)
		}
		_, err = q.Runners.Create(ctx, pn, ru)
		if err != nil {
			return fmt.Errorf("failed to create Runner %q: %w", ru.Name, err)
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

	dbrs := make(map[string]resource.Resource)
	for _, r := range dbpp.Resources {
		dbrs[r.Name] = r
	}
	for _, r := range pp.Resources {
		if !utils.ValidateCanonical(r.Name) {
			return fmt.Errorf("invalid Resource Name format %q", r.Name)
		}
		if dbr, ok := dbrs[r.Name]; ok {
			delete(dbrs, r.Name)
			if dbr.CheckInterval != r.CheckInterval {
				q.cron.Remove(cron.EntryID(dbr.CronID))
				spec := r.CheckInterval
				if spec == "" {
					spec = "@every 1m"
				}
				eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(pp.Name, r.Canonical))
				if err != nil {
					return fmt.Errorf("failed to add Cron Func %q: %w", r.Canonical, err)
				}
				r.CronID = uint64(eid)
			}
			err = q.Resources.Update(ctx, pn, r.Canonical, r)
			if err != nil {
				return fmt.Errorf("failed to update Resource %q: %w", r.Canonical, err)
			}
		} else {
			q.cron.Remove(cron.EntryID(dbr.CronID))
			spec := r.CheckInterval
			if spec == "" {
				spec = "@every 1m"
			}
			eid, err := q.cron.AddFunc(spec, q.newCronResourceFunc(pp.Name, r.Canonical))
			if err != nil {
				return fmt.Errorf("failed to add Cron Func %q: %w", r.Canonical, err)
			}
			r.CronID = uint64(eid)
			_, err = q.Resources.Create(ctx, pn, r)
			if err != nil {
				return fmt.Errorf("failed to create Resource %q: %w", r.Canonical, err)
			}
		}
	}
	for rn := range dbrs {
		err = q.Resources.Delete(ctx, pn, rn)
		if err != nil {
			return fmt.Errorf("failed to delete Resource %q: %w", rn, err)
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
			err = q.Runners.Update(ctx, pn, ru.Name, ru)
			if err != nil {
				return fmt.Errorf("failed to update Runner %q: %w", ru.Name, err)
			}
		} else {
			_, err = q.Runners.Create(ctx, pn, ru)
			if err != nil {
				return fmt.Errorf("failed to create Runner %q: %w", ru.Name, err)
			}
		}
	}
	for run := range dbru {
		err = q.Runners.Delete(ctx, pn, run)
		if err != nil {
			return fmt.Errorf("failed to delete Runner %q: %w", run, err)
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

		runners, err := q.Runners.Filter(ctx, pp.Name)
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

	runners, err := q.Runners.Filter(ctx, pn)
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
	resources, err := q.Resources.Filter(ctx, pn)
	if err != nil {
		return fmt.Errorf("failed to get Resources from Pipeline %q: %w", pn, err)
	}

	// Removes all the Cron Resources
	for _, res := range resources {
		q.cron.Remove(cron.EntryID(res.CronID))
	}

	err = q.Pipelines.Delete(ctx, pn)
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

	if b.Status != build.Started && b.Duration == 0 {
		b.Duration = time.Now().Sub(b.StartedAt)
	}

	err := q.Builds.Update(ctx, pn, jn, bID, b)
	if err != nil {
		return fmt.Errorf("failed to Update Build: %w", err)
	}

	return nil
}

func (q *Qid) DeleteJobBuild(ctx context.Context, pn, jn string, bID uint32) error {
	if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", pn)
	}

	err := q.Builds.Delete(ctx, pn, jn, bID)
	if err != nil {
		return fmt.Errorf("failed to Delete Build: %w", err)
	}

	return nil
}

func (q *Qid) CreateResourceVersion(ctx context.Context, pn, rCan string, v resource.Version) (*resource.Version, error) {
	if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateResourceCanonical(rCan) {
		return nil, fmt.Errorf("invalid Resource Canonical format %q", rCan)
	}

	id, err := q.Resources.CreateVersion(ctx, pn, rCan, v)
	if err != nil {
		return nil, fmt.Errorf("failed to Create Resource Version: %w", err)
	}

	v.ID = id

	return &v, nil
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
