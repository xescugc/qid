package pipeline

import "github.com/xescugc/qid/qid/job"

type Pipeline struct {
	ID   uint32
	Name string
	Jobs []job.Job `json:"jobs"`
	//Resources []Resource
}
