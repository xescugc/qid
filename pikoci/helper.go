package pikoci

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/xescugc/pikoci/pikoci/job"
	"github.com/xescugc/pikoci/pikoci/pipeline"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
	"github.com/xescugc/pikoci/pikoci/sectype"
	"github.com/xescugc/pikoci/pikoci/source"
	"github.com/xescugc/pikoci/pikoci/utils"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/gocty"
)

// hclGetStep is the HCL-decoded get step with per-step hooks.
type hclGetStep struct {
	Type    string   `json:"type" hcl:"type,label"`
	Name    string   `json:"name" hcl:"name,label"`
	Passed  []string `json:"passed" hcl:"passed,optional"`
	Trigger  bool     `json:"trigger" hcl:"trigger,optional"`
	Timeout  string   `json:"timeout" hcl:"timeout,optional"`
	Attempts int      `json:"attempts" hcl:"attempts,optional"`
	Secrets  map[string]string `json:"secrets" hcl:"secrets,optional"`

	OnSuccess []utils.RunnerCommand `json:"on_success" hcl:"on_success,block"`
	OnFailure []utils.RunnerCommand `json:"on_failure" hcl:"on_failure,block"`
	Ensure    []utils.RunnerCommand `json:"ensure" hcl:"ensure,block"`
}

// hclTaskStep is the HCL-decoded task step with per-step hooks.
type hclTaskStep struct {
	Name     string              `json:"name" hcl:"name,label"`
	Timeout  string              `json:"timeout" hcl:"timeout,optional"`
	Attempts int                 `json:"attempts" hcl:"attempts,optional"`
	Secrets  map[string]string   `json:"secrets" hcl:"secrets,optional"`
	Run      utils.RunnerCommand `json:"run" hcl:"run,block"`

	OnSuccess []utils.RunnerCommand `json:"on_success" hcl:"on_success,block"`
	OnFailure []utils.RunnerCommand `json:"on_failure" hcl:"on_failure,block"`
	Ensure    []utils.RunnerCommand `json:"ensure" hcl:"ensure,block"`
}

// hclPutStep is the HCL-decoded put step.
type hclPutStep struct {
	Type     string   `hcl:"type,label"`
	Name     string   `hcl:"name,label"`
	Timeout  string   `hcl:"timeout,optional"`
	Attempts int      `hcl:"attempts,optional"`
	Secrets  map[string]string `hcl:"secrets,optional"`

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

// hclResourceType is an intermediate struct that allows optional check/pull/push blocks
// when source is provided.
type hclResourceType struct {
	Name   string   `json:"name" hcl:"name,label"`
	Source string   `json:"source,omitempty" hcl:"source,optional"`
	Params []string `json:"params" hcl:"params,optional"`

	Check []utils.RunnerCommand `json:"check" hcl:"check,block"`
	Pull  []utils.RunnerCommand `json:"pull" hcl:"pull,block"`
	Push  []utils.RunnerCommand `json:"push" hcl:"push,block"`
}

func (hrt hclResourceType) toResourceType() restype.ResourceType {
	rt := restype.ResourceType{
		Name:   hrt.Name,
		Source: hrt.Source,
		Params: hrt.Params,
	}
	if len(hrt.Check) > 0 {
		rt.Check = hrt.Check[0]
	}
	if len(hrt.Pull) > 0 {
		rt.Pull = hrt.Pull[0]
	}
	if len(hrt.Push) > 0 {
		rt.Push = hrt.Push[0]
	}
	return rt
}

// hclRunnerDef is an intermediate struct that allows optional run block
// when source is provided.
type hclRunnerDef struct {
	Name   string             `json:"name" hcl:"name,label"`
	Source string             `json:"source,omitempty" hcl:"source,optional"`
	Run    []utils.RunCommand `json:"run" hcl:"run,block"`
}

func (hrd hclRunnerDef) toRunner() runner.Runner {
	ru := runner.Runner{
		Name:   hrd.Name,
		Source: hrd.Source,
	}
	if len(hrd.Run) > 0 {
		ru.Run = hrd.Run[0]
	}
	return ru
}

// hclSecretType is an intermediate struct that allows optional get block
// when source is provided. Config attributes (address, token, etc.) are
// captured via Remain.
type hclSecretType struct {
	Name   string   `json:"name" hcl:"name,label"`
	Source string   `json:"source,omitempty" hcl:"source,optional"`
	Params []string `json:"params" hcl:"params,optional"`

	Get    []utils.RunnerCommand `json:"get" hcl:"get,block"`
	Remain hcl.Body              `hcl:",remain"`
}

func (hst hclSecretType) toSecretType() sectype.SecretType {
	st := sectype.SecretType{
		Name:   hst.Name,
		Source: hst.Source,
		Params: hst.Params,
	}
	if len(hst.Get) > 0 {
		st.Get = hst.Get[0]
	}
	return st
}

// hclPipeline is the intermediate HCL-decoded pipeline.
type hclPipeline struct {
	Name          string              `json:"name"`
	Jobs          []hclJob            `hcl:"job,block"`
	Resources     []resource.Resource `hcl:"resource,block"`
	ResourceTypes []hclResourceType   `hcl:"resource_type,block"`
	Runners       []hclRunnerDef      `hcl:"runner,block"`
	SecretTypes   []hclSecretType     `hcl:"secret_type,block"`
	Remain        hcl.Body            `hcl:",remain"`
}

func hclFunctions() map[string]function.Function {
	return map[string]function.Function{
		// String
		"chomp":      stdlib.ChompFunc,
		"format":     stdlib.FormatFunc,
		"formatlist": stdlib.FormatListFunc,
		"indent":     stdlib.IndentFunc,
		"join":       stdlib.JoinFunc,
		"lower":      stdlib.LowerFunc,
		"replace":    stdlib.ReplaceFunc,
		"split":      stdlib.SplitFunc,
		"substr":     stdlib.SubstrFunc,
		"title":      stdlib.TitleFunc,
		"trim":       stdlib.TrimFunc,
		"trimprefix": stdlib.TrimPrefixFunc,
		"trimsuffix": stdlib.TrimSuffixFunc,
		"trimspace":  stdlib.TrimSpaceFunc,
		"upper":      stdlib.UpperFunc,
		// Collection
		"concat":   stdlib.ConcatFunc,
		"contains": stdlib.ContainsFunc,
		"distinct": stdlib.DistinctFunc,
		"flatten":  stdlib.FlattenFunc,
		"keys":     stdlib.KeysFunc,
		"length":   stdlib.LengthFunc,
		"lookup":   stdlib.LookupFunc,
		"merge":    stdlib.MergeFunc,
		"reverse":  stdlib.ReverseListFunc,
		"sort":     stdlib.SortFunc,
		"values":   stdlib.ValuesFunc,
		// Numeric
		"abs":   stdlib.AbsoluteFunc,
		"ceil":  stdlib.CeilFunc,
		"floor": stdlib.FloorFunc,
		"max":   stdlib.MaxFunc,
		"min":   stdlib.MinFunc,
		// Encoding
		"jsonencode": stdlib.JSONEncodeFunc,
		"jsondecode": stdlib.JSONDecodeFunc,
		"csvdecode":  stdlib.CSVDecodeFunc,
		// Regex
		"regex":        stdlib.RegexFunc,
		"regexall":     stdlib.RegexAllFunc,
		"regexreplace": stdlib.RegexReplaceFunc,
	}
}

func (q *PikoCI) readPipeline(ctx context.Context, rpp []byte, vars map[string]interface{}) (*pipeline.Pipeline, error) {
	funcs := hclFunctions()
	ectx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"string": cty.StringVal("string"),
			"number": cty.StringVal("number"),
			"bool":   cty.StringVal("bool"),
		},
		Functions: funcs,
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
		Functions: funcs,
	}

	var hp hclPipeline
	err = hclsimple.Decode("pipeline.hcl", rpp, ectx, &hp)
	if err != nil {
		for _, e := range err.(hcl.Diagnostics).Errs() {
			spew.Dump(e)
		}
		return nil, fmt.Errorf("failed to Decode Pipeline config: %w", err)
	}

	// Convert intermediate types and resolve sources
	var resourceTypes []restype.ResourceType
	for _, hrt := range hp.ResourceTypes {
		if hrt.Source != "" {
			hasInline := len(hrt.Check) > 0 || len(hrt.Pull) > 0 || len(hrt.Push) > 0
			if hasInline {
				return nil, fmt.Errorf("resource_type %q has both source and inline commands, which is not allowed", hrt.Name)
			}
			resolved, err := source.ResolveResourceType(ctx, hrt.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve source for resource_type %q: %w", hrt.Name, err)
			}
			resolved.Name = hrt.Name
			resolved.Source = hrt.Source
			resourceTypes = append(resourceTypes, *resolved)
		} else {
			resourceTypes = append(resourceTypes, hrt.toResourceType())
		}
	}

	var runners []runner.Runner
	for _, hrd := range hp.Runners {
		if hrd.Source != "" {
			hasInline := len(hrd.Run) > 0
			if hasInline {
				return nil, fmt.Errorf("runner %q has both source and inline commands, which is not allowed", hrd.Name)
			}
			resolved, err := source.ResolveRunner(ctx, hrd.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve source for runner %q: %w", hrd.Name, err)
			}
			resolved.Name = hrd.Name
			resolved.Source = hrd.Source
			runners = append(runners, *resolved)
		} else {
			runners = append(runners, hrd.toRunner())
		}
	}

	// Parse secret_type config attributes from the raw HCL AST.
	// Known fields (name, source, params, get) are handled by hclsimple.
	// Any extra attributes are config values (address, token, etc.).
	knownSecretTypeAttrs := map[string]bool{"source": true, "params": true}
	secretTypeConfigs := make(map[int]map[string]string)
	{
		file, diags := hclsyntax.ParseConfig(rpp, "pipeline.hcl", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse pipeline HCL: %s", diags.Error())
		}
		stIdx := 0
		for _, block := range file.Body.(*hclsyntax.Body).Blocks {
			if block.Type != "secret_type" {
				continue
			}
			config := make(map[string]string)
			for name, attr := range block.Body.Attributes {
				if knownSecretTypeAttrs[name] {
					continue
				}
				val, vdiags := attr.Expr.Value(ectx)
				if vdiags.HasErrors() {
					return nil, fmt.Errorf("failed to evaluate secret_type config %q: %s", name, vdiags.Error())
				}
				config[name] = val.AsString()
			}
			if len(config) > 0 {
				secretTypeConfigs[stIdx] = config
			}
			stIdx++
		}
	}

	var secretTypes []sectype.SecretType
	for i, hst := range hp.SecretTypes {
		if hst.Source != "" {
			hasInline := len(hst.Get) > 0
			if hasInline {
				return nil, fmt.Errorf("secret_type %q has both source and inline commands, which is not allowed", hst.Name)
			}
			resolved, err := source.ResolveSecretType(ctx, hst.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve source for secret_type %q: %w", hst.Name, err)
			}
			resolved.Name = hst.Name
			resolved.Source = hst.Source
			if cfg, ok := secretTypeConfigs[i]; ok {
				resolved.Config = cfg
			}
			secretTypes = append(secretTypes, *resolved)
		} else {
			st := hst.toSecretType()
			if cfg, ok := secretTypeConfigs[i]; ok {
				st.Config = cfg
			}
			secretTypes = append(secretTypes, st)
		}
	}

	// Parse the raw HCL to determine block ordering within each job.
	jobPlans, err := parseJobPlans(rpp, hp.Jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job plans: %w", err)
	}

	pp := pipeline.Pipeline{
		Resources:     hp.Resources,
		ResourceTypes: resourceTypes,
		Runners:       runners,
		SecretTypes:   secretTypes,
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
				var timeout time.Duration
				if g.Timeout != "" {
					var err error
					timeout, err = time.ParseDuration(g.Timeout)
					if err != nil {
						return nil, fmt.Errorf("invalid timeout %q on get step %q: %w", g.Timeout, g.Name, err)
					}
				}
				if g.Attempts < 0 {
					return nil, fmt.Errorf("invalid attempts %d on get step %q: must be >= 0", g.Attempts, g.Name)
				}
				plan = append(plan, job.PlanStep{
					Type:     job.StepTypeGet,
					Timeout:  timeout,
					Attempts: g.Attempts,
					Secrets:  g.Secrets,
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
				var timeout time.Duration
				if t.Timeout != "" {
					var err error
					timeout, err = time.ParseDuration(t.Timeout)
					if err != nil {
						return nil, fmt.Errorf("invalid timeout %q on task step %q: %w", t.Timeout, t.Name, err)
					}
				}
				if t.Attempts < 0 {
					return nil, fmt.Errorf("invalid attempts %d on task step %q: must be >= 0", t.Attempts, t.Name)
				}
				plan = append(plan, job.PlanStep{
					Type:     job.StepTypeTask,
					Timeout:  timeout,
					Attempts: t.Attempts,
					Secrets:  t.Secrets,
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
				var timeout time.Duration
				if p.Timeout != "" {
					var err error
					timeout, err = time.ParseDuration(p.Timeout)
					if err != nil {
						return nil, fmt.Errorf("invalid timeout %q on put step %q: %w", p.Timeout, p.Name, err)
					}
				}
				if p.Attempts < 0 {
					return nil, fmt.Errorf("invalid attempts %d on put step %q: must be >= 0", p.Attempts, p.Name)
				}
				plan = append(plan, job.PlanStep{
					Type:     job.StepTypePut,
					Timeout:  timeout,
					Attempts: p.Attempts,
					Secrets:  p.Secrets,
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
