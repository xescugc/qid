package build

//go:generate go tool enumer -type=Status -transform=snake -output=status_string.go -json

type Status int

const (
	Succeeded Status = iota
	Failed
	Started
)

// Build represents a run of a Job
type Build struct {
	ID     uint32 `json:"id"`
	Get    []Step `json:"get"`
	Task   []Step `json:"task"`
	Status Status `json:"status"`
	Error  string `json:"error"`
}

type Step struct {
	Name        string `json:"name"`
	VersionHash string `json:"version_hash"`
	Logs        string `json:"logs"`
}
