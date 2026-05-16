package pipeline

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/xescugc/pikoci/pikoci/builtin"
	"github.com/xescugc/pikoci/pikoci/job"
	"github.com/xescugc/pikoci/pikoci/resource"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
	"github.com/xescugc/pikoci/pikoci/secret"
	"github.com/xescugc/pikoci/pikoci/sectype"
)

type Pipeline struct {
	ID            uint32                 `json:"id"`
	Name          string                 `json:"name"`
	Public        bool                   `json:"public"`
	Jobs          []job.Job              `json:"jobs" hcl:"job,block"`
	Resources     []resource.Resource    `json:"resources" hcl:"resource,block"`
	ResourceTypes []restype.ResourceType `json:"resource_types" hcl:"resource_type,block"`
	Runners       []runner.Runner        `json:"runners" hcl:"runner,block"`
	SecretTypes   []sectype.SecretType   `json:"secret_types" hcl:"secret_type,block"`
	Secrets       []secret.Secret        `json:"secrets" hcl:"secret,block"`
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

func (pp *Pipeline) Secret(sCan string) (secret.Secret, bool) {
	for _, s := range pp.Secrets {
		if s.Canonical == sCan {
			return s, true
		}
	}

	return secret.Secret{}, false
}
