package restype

import "github.com/xescugc/qid/qid/utils"

type ResourceType struct {
	ID     uint32   `json:"id"`
	Name   string   `json:"name" hcl:"name,label"`
	Params []string `json:"params" hcl:"params"`

	Check utils.RunnerCommand `json:"check" hcl:"check,block"`
	Pull  utils.RunnerCommand `json:"pull" hcl:"pull,block"`
	Push  utils.RunnerCommand `json:"push" hcl:"push,block"`
}
