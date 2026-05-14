package job_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/pikoci/pikoci/job"
)

func TestGetStep_ResourceCanonical(t *testing.T) {
	g := &job.GetStep{Type: "git", Name: "my-repo"}
	assert.Equal(t, "git.my-repo", g.ResourceCanonical())

	g = &job.GetStep{Type: "cron", Name: "timer"}
	assert.Equal(t, "cron.timer", g.ResourceCanonical())
}
