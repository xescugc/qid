package build

import "time"

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
	// Job are the general logs printed at the end
	Job []Step `json:"job"`

	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration"`
}

type Step struct {
	Name        string        `json:"name"`
	VersionHash string        `json:"version_hash"`
	Logs        string        `json:"logs"`
	Duration    time.Duration `json:"duration"`
}
