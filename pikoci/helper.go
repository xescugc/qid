package pikoci

import (
	"context"
	"fmt"
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/xescugc/pikoci/pikoci/job"
	"github.com/xescugc/pikoci/pikoci/pipeline"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
	"github.com/xescugc/pikoci/pikoci/utils"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// hclGetStep is the HCL-decoded get step with per-step hooks.
type hclGetStep struct {
	Type    string   `json:"type" hcl:"type,label"`
	Name    string   `json:"name" hcl:"name,label"`
	Passed  []string `json:"passed" hcl:"passed,optional"`
	Trigger bool     `json:"trigger" hcl:"trigger,optional"`

	OnSuccess []utils.RunnerCommand `json:"on_success" hcl:"on_success,block"`
	OnFailure []utils.RunnerCommand `json:"on_failure" hcl:"on_failure,block"`
	Ensure    []utils.RunnerCommand `json:"ensure" hcl:"ensure,block"`
}

// hclTaskStep is the HCL-decoded task step with per-step hooks.
type hclTaskStep struct {
	Name string              `json:"name" hcl:"name,label"`
	Run  utils.RunnerCommand `json:"run" hcl:"run,block"`

	OnSuccess []utils.RunnerCommand `json:"on_success" hcl:"on_success,block"`
	OnFailure []utils.RunnerCommand `json:"on_failure" hcl:"on_failure,block"`
	Ensure    []utils.RunnerCommand `json:"ensure" hcl:"ensure,block"`
}

// hclPutStep is the HCL-decoded put step.
type hclPutStep struct {
	Type string `hcl:"type,label"`
	Name string `hcl:"name,label"`

	OnSuccess []utils.RunnerCommand `hcl:"on_success,block"`
	OnFailure []utils.RunnerCommand `hcl:"on_failure,block"`
	Ensure    []utils.RunnerCommand `hcl:"ensure,block"`

	Params map[string]string `hcl:",remain"`
}

// hclJob is the intermediate HCL-decoded job with separate get/task/put arrays.
type hclJob struct {
	Name string       `hcl:"name,label"`
	Get  []hclGetStep `hcl:"get,block"`
	Task []hclTaskStep `hcl:"task,block"`
	Put  []hclPutStep `hcl:"put,block"`

	OnSuccess []utils.RunnerCommand `hcl:"on_success,block"`
	OnFailure []utils.RunnerCommand `hcl:"on_failure,block"`
	Ensure    []utils.RunnerCommand `hcl:"ensure,block"`
}

// hclPipeline is the intermediate HCL-decoded pipeline.
type hclPipeline struct {
	Name          string                 `json:"name"`
	Jobs          []hclJob               `hcl:"job,block"`
	Resources     []resource.Resource    `hcl:"resource,block"`
	ResourceTypes []restype.ResourceType `hcl:"resource_type,block"`
	Runners       []runner.Runner        `hcl:"runner,block"`
	Remain        hcl.Body               `hcl:",remain"`
}

func (q *PikoCI) readPipeline(ctx context.Context, rpp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error) {
	ectx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"string": cty.StringVal("string"),
			"number": cty.StringVal("number"),
			"bool":   cty.StringVal("bool"),
		},
	}
	var pvars pipeline.Variables
	err := hclsimple.Decode("pipeline.hcl", rpp, ectx, &pvars)
	if err != nil {
		return nil, fmt.Errorf("failed to Decode Pipeline config: %w", err)
	}

	ecvars := make(map[string]cty.Value)
	for _, v := range pvars.Variables {
		switch v.Type {
		case "string":
			if mv, ok := vars[v.Name]; ok {
				s, ok := mv.(string)
				if !ok {
					return nil, fmt.Errorf("variable %q configured with invalid type type, expected 'string'", v.Name)
				}
				ecvars[v.Name] = cty.StringVal(s)
			} else {
				a, ok := v.Default.(*hcl.Attribute)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'string'", v.Name)
				}
				ctyv, _ := a.Expr.Value(ectx)
				var s string
				err = gocty.FromCtyValue(ctyv, &s)
				if err != nil {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'string'", v.Name)
				}
				ecvars[v.Name] = cty.StringVal(s)
			}
		case "number":
			if mv, ok := vars[v.Name]; ok {
				n, ok := mv.(float64)
				if !ok {
					return nil, fmt.Errorf("variable %q configured with invalid type type, expected 'number'", v.Name)
				}
				ecvars[v.Name] = cty.NumberVal(big.NewFloat(n))
			} else {
				a, ok := v.Default.(*hcl.Attribute)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'number'", v.Name)
				}
				ctyv, _ := a.Expr.Value(ectx)
				var n float64
				err = gocty.FromCtyValue(ctyv, &n)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'number'", v.Name)
				}
				ecvars[v.Name] = cty.NumberVal(big.NewFloat(n))
			}
		case "bool":
			if mv, ok := vars[v.Name]; ok {
				b, ok := mv.(bool)
				if !ok {
					return nil, fmt.Errorf("variable %q configured with invalid type type, expected 'bool'", v.Name)
				}
				ecvars[v.Name] = cty.BoolVal(b)
			} else {
				a, ok := v.Default.(*hcl.Attribute)
				if !ok {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'bool'", v.Name)
				}
				ctyv, _ := a.Expr.Value(ectx)
				var b bool
				err = gocty.FromCtyValue(ctyv, &b)
				if err != nil {
					return nil, fmt.Errorf("variable %q has an invalid default type, expected 'bool'", v.Name)
				}
				ecvars[v.Name] = cty.BoolVal(b)
			}
		}
	}
	ectx = &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(ecvars),
		},
	}

	var hp hclPipeline
	err = hclsimple.Decode("pipeline.hcl", rpp, ectx, &hp)
	if err != nil {
		for _, e := range err.(hcl.Diagnostics).Errs() {
			spew.Dump(e)
		}
		return nil, fmt.Errorf("failed to Decode Pipeline config: %w", err)
	}

	// Parse the raw HCL to determine block ordering within each job.
	jobPlans, err := parseJobPlans(rpp, hp.Jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job plans: %w", err)
	}

	pp := pipeline.Pipeline{
		Resources:     hp.Resources,
		ResourceTypes: hp.ResourceTypes,
		Runners:       hp.Runners,
	}

	for _, hj := range hp.Jobs {
		j := job.Job{
			Name:      hj.Name,
			Plan:      jobPlans[hj.Name],
			OnSuccess: hj.OnSuccess,
			OnFailure: hj.OnFailure,
			Ensure:    hj.Ensure,
		}
		pp.Jobs = append(pp.Jobs, j)
	}

	for i, r := range pp.Resources {
		pp.Resources[i].Canonical = utils.ResourceCanonical(r.Type, r.Name)
	}
	return &pp, nil
}

// parseJobPlans walks the raw HCL AST to extract get/task/put blocks in source
// order for each job, then builds ordered PlanStep slices using the decoded data.
func parseJobPlans(rpp []byte, hclJobs []hclJob) (map[string][]job.PlanStep, error) {
	file, diags := hclsyntax.ParseConfig(rpp, "pipeline.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, diags
	}

	body := file.Body.(*hclsyntax.Body)
	result := make(map[string][]job.PlanStep)

	jobIndex := 0
	for _, block := range body.Blocks {
		if block.Type != "job" {
			continue
		}
		if jobIndex >= len(hclJobs) {
			break
		}
		hj := hclJobs[jobIndex]
		jobIndex++

		var plan []job.PlanStep
		getIdx, taskIdx, putIdx := 0, 0, 0

		for _, innerBlock := range block.Body.Blocks {
			switch innerBlock.Type {
			case "get":
				if getIdx >= len(hj.Get) {
					continue
				}
				g := hj.Get[getIdx]
				getIdx++
				plan = append(plan, job.PlanStep{
					Type: job.StepTypeGet,
					Get: &job.GetStep{
						Type:    g.Type,
						Name:    g.Name,
						Passed:  g.Passed,
						Trigger: g.Trigger,
					},
					OnSuccess: g.OnSuccess,
					OnFailure: g.OnFailure,
					Ensure:    g.Ensure,
				})
			case "task":
				if taskIdx >= len(hj.Task) {
					continue
				}
				t := hj.Task[taskIdx]
				taskIdx++
				plan = append(plan, job.PlanStep{
					Type: job.StepTypeTask,
					Task: &job.TaskStep{
						Name: t.Name,
						Run:  t.Run,
					},
					OnSuccess: t.OnSuccess,
					OnFailure: t.OnFailure,
					Ensure:    t.Ensure,
				})
			case "put":
				if putIdx >= len(hj.Put) {
					continue
				}
				p := hj.Put[putIdx]
				putIdx++
				plan = append(plan, job.PlanStep{
					Type: job.StepTypePut,
					Put: &job.PutStep{
						Type:   p.Type,
						Name:   p.Name,
						Params: p.Params,
					},
					OnSuccess: p.OnSuccess,
					OnFailure: p.OnFailure,
					Ensure:    p.Ensure,
				})
			}
		}

		result[hj.Name] = plan
	}

	return result, nil
}
