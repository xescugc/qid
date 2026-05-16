package job

import (
	"time"

	"github.com/xescugc/pikoci/pikoci/utils"
)

type StepType string

const (
	StepTypeGet     StepType = "get"
	StepTypeTask    StepType = "task"
	StepTypePut     StepType = "put"
	StepTypeService StepType = "service"
)

type Job struct {
	ID   uint32 `json:"id"`
	Name string `json:"name" hcl:"name,label"`
	Plan []PlanStep `json:"plan"`

	OnSuccess []utils.RunnerCommand `json:"on_success" hcl:"on_success,block"`
	OnFailure []utils.RunnerCommand `json:"on_failure" hcl:"on_failure,block"`
	Ensure    []utils.RunnerCommand `json:"ensure" hcl:"ensure,block"`
}

// GetSteps returns all get steps from the plan in order.
func (j *Job) GetSteps() []GetStep {
	var steps []GetStep
	for _, p := range j.Plan {
		if p.Type == StepTypeGet && p.Get != nil {
			steps = append(steps, *p.Get)
		}
	}
	return steps
}

// PlanGetSteps returns all PlanSteps that are get steps.
func (j *Job) PlanGetSteps() []PlanStep {
	var steps []PlanStep
	for _, p := range j.Plan {
		if p.Type == StepTypeGet {
			steps = append(steps, p)
		}
	}
	return steps
}

type PlanStep struct {
	Type      StepType              `json:"type"`
	Timeout   time.Duration         `json:"timeout,omitempty"`
	Attempts  int                   `json:"attempts,omitempty"`
	Secrets   map[string]string      `json:"secrets,omitempty"`
	Get       *GetStep              `json:"get,omitempty"`
	Task      *TaskStep             `json:"task,omitempty"`
	Put       *PutStep              `json:"put,omitempty"`
	Service   *ServiceStep          `json:"service,omitempty"`
	OnSuccess []utils.RunnerCommand `json:"on_success,omitempty"`
	OnFailure []utils.RunnerCommand `json:"on_failure,omitempty"`
	Ensure    []utils.RunnerCommand `json:"ensure,omitempty"`
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
	Name string              `json:"name" hcl:"name,label"`
	Run  utils.RunnerCommand `json:"run" hcl:"run,block"`
}

type PutStep struct {
	Type   string            `json:"type"`
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
}

func (p *PutStep) ResourceCanonical() string {
	return utils.ResourceCanonical(p.Type, p.Name)
}

type ServiceStep struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
}
