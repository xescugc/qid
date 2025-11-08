package runner

import "github.com/xescugc/qid/qid/utils"

type Runner struct {
	ID   uint32           `json:"id"`
	Name string           `json:"name" hcl:"name,label"`
	Run  utils.RunCommand `json:"run" hcl:"run,block"`
}
