package job

import "github.com/xescugc/qid/qid/utils"

type Job struct {
	ID   uint32     `json:"id"`
	Name string     `json:"name" hcl:"name,label"`
	Get  []GetStep  `json:"gets" hcl:"get,block"`
	Task []TaskStep `json:"tasks" hcl:"task,block"`
}

type GetStep struct {
	Type    string   `json:"type" hcl:"type,label"`
	Name    string   `json:"name" hcl:"name,label"`
	Passed  []string `json:"passed" hcl:"passed,optional"`
	Trigger bool     `json:"trigger" hcl:"trigger,optional"`
}

func (g *GetStep) ResourceCanonical() string {
	return utils.ResourceCanonical(g.Type, g.Name)
}

type TaskStep struct {
	Name string     `json:"name" hcl:"name,label"`
	Run  RunCommand `json:"run" hcl:"run,block"`
}

type RunCommand struct {
	Path string   `json:"path" hcl:"path"`
	Args []string `json:"args" hcl:"args,optional"`
}
