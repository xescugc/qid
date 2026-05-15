package utils

type RunnerCommand struct {
	Runner string            `json:"runner" hcl:"runner,label"`
	Args   []string          `json:"args" hcl:"args,optional"`
	Params map[string]string `json:"params" hcl:",remain"`
}

type RunCommand struct {
	Path string   `json:"path" hcl:"path,optional"`
	Args []string `json:"args" hcl:"args,optional"`
}
