package pipeline

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
)

type Pipeline struct {
	ID            uint32                 `json:"id"`
	Name          string                 `json:"name"`
	Jobs          []job.Job              `json:"jobs" hcl:"job,block"`
	Resources     []resource.Resource    `json:"resources" hcl:"resource,block"`
	ResourceTypes []restype.ResourceType `json:"resource_types" hcl:"resource_type,block"`
	Remain        hcl.Body               `json:"-" hcl:",remain"`
	Raw           []byte                 `json:"raw"`
}

type Variables struct {
	Variables []Variable `json:"variables" hcl:"variable,block"`
	Remain    hcl.Body   `hcl:",remain"`
}
type Variable struct {
	Name        string      `json:"name" hcl:"name,label"`
	Type        string      `json:"type" hcl:"type"`
	Default     interface{} `json:"default" hcl:"default,optional"`
	Description string      `json:"description" hcl:"description,optional"`
}
