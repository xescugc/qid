package resource

import "time"

type Resource struct {
	ID   uint32 `json:"id"`
	Type string `json:"type" hcl:"type,label"`
	Name string `json:"name" hcl:"name,label"`

	Params        Params `json:"params" hcl:"params,block"`
	CheckInterval string `json:"check_interval" hcl:"check_interval,optional"`

	CronID    uint64    `json:"cron_id"`
	Canonical string    `json:"canonical"`
	Logs      string    `json:"logs"`
	LastCheck time.Time `json:"last_check"`
}

type Params struct {
	Params map[string]string `json:"params" hcl:",remain"`
}

type Version struct {
	ID      uint32                 `json:"id"`
	Version map[string]interface{} `json:"version"`
}
