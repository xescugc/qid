package pipeline

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/runner"
	"github.com/xescugc/qid/qid/utils"
)

type Pipeline struct {
	ID            uint32                 `json:"id"`
	Name          string                 `json:"name"`
	Jobs          []job.Job              `json:"jobs" hcl:"job,block"`
	Resources     []resource.Resource    `json:"resources" hcl:"resource,block"`
	ResourceTypes []restype.ResourceType `json:"resource_types" hcl:"resource_type,block"`
	Runners       []runner.Runner        `json:"runners" hcl:"runner,block"`
	Remain        hcl.Body               `json:"-" hcl:",remain"`
	Raw           []byte                 `json:"raw"`
}

type Variables struct {
	Variables []Variable `json:"variables" hcl:"variable,block"`
	Remain    hcl.Body   `hcl:",remain"`
}
type Variable struct {
	Name    string      `json:"name" hcl:"name,label"`
	Type    string      `json:"type" hcl:"type"`
	Default interface{} `json:"default" hcl:"default,optional"`
}

func (pp *Pipeline) ResourceType(rtn string) (restype.ResourceType, bool) {
	for _, rt := range pp.ResourceTypes {
		if rt.Name == rtn {
			return rt, true
		}
	}

	if rtn == "cron" {
		return restype.ResourceType{
			Name: "cron",
			Check: utils.RunnerCommand{
				Runner: "exec",
				Params: map[string]string{
					"path": "date",
				},
			},
		}, true
	}

	return restype.ResourceType{}, false
}

func (pp *Pipeline) Resource(rCan string) (resource.Resource, bool) {
	for _, r := range pp.Resources {
		if r.Canonical == rCan {
			return r, true
		}
	}

	return resource.Resource{}, false
}

func (pp *Pipeline) Runner(run string) (runner.Runner, bool) {
	for _, ru := range pp.Runners {
		if ru.Name == run {
			return ru, true
		}
	}

	if run == "exec" {
		return runner.Runner{
			Name: "exec",
			Run: utils.RunCommand{
				Path: "$path",
				Args: []string{"$args"},
			},
		}, true
	}

	return runner.Runner{}, false
}
