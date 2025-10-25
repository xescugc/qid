package resource

import "time"

type Resource struct {
	ID   uint32 `json:"id"`
	Type string `json:"type" hcl:"type,label"`
	Name string `json:"name" hcl:"name,label"`

	Inputs        Inputs `json:"inputs" hcl:"inputs,block"`
	CheckInterval string `json:"check_interval" hcl:"check_interval,optional"`

	Canonical string    `json:"canonical"`
	Logs      string    `json:"logs"`
	LastCheck time.Time `json:"last_check"`
}

type Inputs struct {
	Inputs map[string]string `json:"inputs" hcl:",remain"`
}

type Version struct {
	ID   uint32 `json:"id"`
	Hash string `json:"hash"`
}
