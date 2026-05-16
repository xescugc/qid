package secret

type Secret struct {
	ID        uint32            `json:"id"`
	Type      string            `json:"type" hcl:"type,label"`
	Name      string            `json:"name" hcl:"name,label"`
	Canonical string            `json:"canonical"`
	Params    map[string]string `json:"params,omitempty"`
}
