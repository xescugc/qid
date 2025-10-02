package pipeline

import (
	"github.com/xescugc/qid/qid/job"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
)

type Pipeline struct {
	ID            uint32
	Name          string
	Jobs          []job.Job              `json:"jobs" hcl:"job,block"`
	Resources     []resource.Resource    `json:"resources" hcl:"resource,block"`
	ResourceTypes []restype.ResourceType `json:"resource_types" hcl:"resource_type,block"`
}
