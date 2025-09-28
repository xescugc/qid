package job

type Job struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
	Plan []Step `json:"plan"`
}

type Step struct {
	Task    string     `json:"task"`
	TConfig TaskConfig `json:"task_config"`

	Get     string    `json:"get"`
	GConfig GetConfig `json:"get_config"`
}

type TaskConfig struct {
	Run RunCommand `json:"run"`
}

type RunCommand struct {
	Path string   `json:"path"`
	Args []string `json:"args"`
}

type GetConfig struct {
	Passed  []string `json:"passed"`
	Trigger bool     `json:"trigger"`
}
