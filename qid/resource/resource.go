package resource

type Resource struct {
	ID        uint32 `json:"id"`
	Type      string `json:"type" hcl:"type,label"`
	Name      string `json:"name" hcl:"name,label"`
	Canonical string `json:"canonical"`

	Inputs map[string]string `json:"inputs" hcl:",remain"`

	Logs string `json:"logs"`
}

type Version struct {
	ID   uint32 `json:"id"`
	Hash string `json:"hash"`
}
