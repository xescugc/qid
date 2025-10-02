package resource

type Resource struct {
	ID   uint32 `json:"id"`
	Type string `json:"type" hcl:"type,label"`
	Name string `json:"name" hcl:"name,label"`

	Inputs map[string]string `json:"Inputs" hcl:",remain"`
}

type Version struct {
	ID   uint32 `json:"id"`
	Hash string `json:"hash"`
}
