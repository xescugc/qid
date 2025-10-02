package restype

type ResourceType struct {
	ID     uint32   `json:"id"`
	Name   string   `json:"name" hcl:"name,label"`
	Inputs []string `json:"inputs" hcl:"inputs"`
	Check  Run      `json:"check" hcl:"check,block"`
	Pull   Run      `json:"pull" hcl:"pull,block"`
	Push   Run      `json:"push" hcl:"push,block"`
}

type Run struct {
	Path string   `json:"path" hcl:"path"`
	Args []string `json:"args" hcl:"args"`
}
