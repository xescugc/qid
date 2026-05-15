package restype

import "github.com/xescugc/pikoci/pikoci/utils"

type ResourceType struct {
	ID     uint32   `json:"id"`
	Name   string   `json:"name" hcl:"name,label"`
	Source string   `json:"source,omitempty" hcl:"source,optional"`
	Params []string `json:"params" hcl:"params,optional"`

	Check utils.RunnerCommand `json:"check" hcl:"check,block"`
	Pull  utils.RunnerCommand `json:"pull" hcl:"pull,block"`
	Push  utils.RunnerCommand `json:"push" hcl:"push,block"`
}
