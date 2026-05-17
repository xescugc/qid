package pipeline

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/xescugc/pikoci/pikoci/builtin"
	"github.com/zclconf/go-cty/cty"
	"github.com/xescugc/pikoci/pikoci/job"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
	"github.com/xescugc/pikoci/pikoci/sectype"
	"github.com/xescugc/pikoci/pikoci/service"
	"github.com/xescugc/pikoci/pikoci/source"
	"github.com/xescugc/pikoci/pikoci/utils"
)

type Pipeline struct {
	ID            uint32                    `json:"id"`
	Name          string                    `json:"name"`
	Public        bool                      `json:"public"`
	Jobs          []job.Job                 `json:"jobs" hcl:"job,block"`
	Resources     []resource.Resource       `json:"resources" hcl:"resource,block"`
	ResourceTypes []restype.ResourceType    `json:"resource_types" hcl:"resource_type,block"`
	Runners       []runner.Runner           `json:"runners" hcl:"runner,block"`
	SecretTypes   []sectype.SecretType      `json:"secret_types" hcl:"secret_type,block"`
	Services      []service.Service         `json:"services" hcl:"service,block"`
	SecretVars    map[string]VariableSecret `json:"secret_vars,omitempty"`
	Remain        hcl.Body                  `json:"-" hcl:",remain"`
	Raw           []byte                    `json:"raw"`
}

type Variables struct {
	Variables []Variable `json:"variables" hcl:"variable,block"`
	Remain    hcl.Body   `hcl:",remain"`
}
type Variable struct {
	Name    string           `json:"name" hcl:"name,label"`
	Type    string           `json:"type" hcl:"type"`
	Default interface{}      `json:"default" hcl:"default,optional"`
	Secret  *VariableSecret  `json:"secret,omitempty" hcl:"secret,block"`
}

type VariableSecret struct {
	Type string `json:"type" hcl:"type,label"`
	Path string `json:"path" hcl:"path"`
	Key  string `json:"key" hcl:"key"`
}

func (pp *Pipeline) ResourceType(rtn string) (restype.ResourceType, bool) {
	for _, rt := range pp.ResourceTypes {
		if rt.Name == rtn {
			return rt, true
		}
	}

	if brt, ok := builtin.ResourceTypes()[rtn]; ok {
		return brt, true
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

	if bru, ok := builtin.Runners()[run]; ok {
		return bru, true
	}

	return runner.Runner{}, false
}

func (pp *Pipeline) SecretType(stn string) (sectype.SecretType, bool) {
	for _, st := range pp.SecretTypes {
		if st.Name == stn {
			return st, true
		}
	}

	return sectype.SecretType{}, false
}

func (pp *Pipeline) Service(name string) (service.Service, bool) {
	for _, s := range pp.Services {
		if s.Name == name {
			return s, true
		}
	}

	return service.Service{}, false
}

// hclReadyCheckRaw is used for parsing ready_check blocks from raw HCL.
type hclReadyCheckRaw struct {
	Runner   string            `hcl:"runner,label"`
	Args     []string          `hcl:"args,optional"`
	Interval string            `hcl:"interval,optional"`
	Timeout  string            `hcl:"timeout,optional"`
	Params   map[string]string `hcl:",remain"`
}

// hclServiceRaw is used for parsing top-level service blocks from raw HCL.
type hclServiceRaw struct {
	Name       string                `hcl:"name,label"`
	Source     string                `hcl:"source,optional"`
	Params     []string              `hcl:"params,optional"`
	Start      []utils.RunnerCommand `hcl:"start,block"`
	ReadyCheck []hclReadyCheckRaw    `hcl:"ready_check,block"`
	Stop       []utils.RunnerCommand `hcl:"stop,block"`
}

// hclPipelineServices is a minimal struct for parsing only service blocks from raw HCL.
type hclPipelineServices struct {
	Services []hclServiceRaw `hcl:"service,block"`
	Remain   hcl.Body        `hcl:",remain"`
}

// ParseServicesFromRaw parses service definitions from raw pipeline HCL.
// This extracts both top-level service blocks and inline service definitions
// inside job blocks. Used to populate the Services field on pipelines loaded
// from the database, where services are not stored in a separate table.
func ParseServicesFromRaw(ctx context.Context, raw []byte) ([]service.Service, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Parse top-level service blocks
	var hp hclPipelineServices
	err := hclsimple.Decode("pipeline.hcl", raw, nil, &hp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse services from raw HCL: %w", err)
	}

	serviceByName := make(map[string]bool)
	var services []service.Service
	for _, hs := range hp.Services {
		if hs.Source != "" {
			resolved, err := source.ResolveService(ctx, hs.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve source for service %q: %w", hs.Name, err)
			}
			resolved.Name = hs.Name
			resolved.Source = hs.Source
			if hs.Params != nil {
				resolved.Params = hs.Params
			}
			services = append(services, *resolved)
		} else {
			services = append(services, convertHCLService(hs))
		}
		serviceByName[hs.Name] = true
	}

	return services, nil
}

func convertHCLService(hs hclServiceRaw) service.Service {
	s := service.Service{
		Name:   hs.Name,
		Params: hs.Params,
	}
	if len(hs.Start) > 0 {
		s.Start = hs.Start[0]
	}
	if len(hs.Stop) > 0 {
		s.Stop = hs.Stop[0]
	}
	if len(hs.ReadyCheck) > 0 {
		rc := hs.ReadyCheck[0]
		s.ReadyCheck = &service.ReadyCheck{
			RunnerCommand: utils.RunnerCommand{
				Runner: rc.Runner,
				Args:   rc.Args,
				Params: rc.Params,
			},
			Interval: rc.Interval,
			Timeout:  rc.Timeout,
		}
	}
	return s
}

// hclVariableRaw is a minimal struct for parsing variable blocks from raw HCL.
type hclVariableRaw struct {
	Name   string           `hcl:"name,label"`
	Type   string           `hcl:"type"`
	Secret *VariableSecret  `hcl:"secret,block"`
	Remain hcl.Body         `hcl:",remain"`
}

// hclPipelineVariables is a minimal struct for parsing only variable blocks from raw HCL.
type hclPipelineVariables struct {
	Variables []hclVariableRaw `hcl:"variable,block"`
	Remain    hcl.Body         `hcl:",remain"`
}

// ParseSecretVarsFromRaw parses secret-backed variable declarations from raw pipeline HCL.
// Used to populate the SecretVars field on pipelines loaded from the database.
func ParseSecretVarsFromRaw(raw []byte, vars map[string]interface{}) (map[string]VariableSecret, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	ectx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"string": cty.StringVal("string"),
			"number": cty.StringVal("number"),
			"bool":   cty.StringVal("bool"),
		},
	}

	var pv hclPipelineVariables
	err := hclsimple.Decode("pipeline.hcl", raw, ectx, &pv)
	if err != nil {
		return nil, fmt.Errorf("failed to parse variables from raw HCL: %w", err)
	}

	secretVars := make(map[string]VariableSecret)
	for _, v := range pv.Variables {
		if v.Secret != nil {
			// Only include if not overridden by vars file
			if _, overridden := vars[v.Name]; !overridden {
				secretVars[v.Name] = *v.Secret
			}
		}
	}

	if len(secretVars) == 0 {
		return nil, nil
	}
	return secretVars, nil
}

