package sectype

import "github.com/xescugc/pikoci/pikoci/utils"

type SecretType struct {
	ID     uint32              `json:"id"`
	Name   string              `json:"name" hcl:"name,label"`
	Source string              `json:"source,omitempty" hcl:"source,optional"`
	Params []string            `json:"params" hcl:"params,optional"`
	Get    utils.RunnerCommand `json:"get" hcl:"get,block"`
}
