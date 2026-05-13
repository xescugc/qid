package pipeline_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xescugc/qid/qid/pipeline"
	"github.com/xescugc/qid/qid/resource"
	"github.com/xescugc/qid/qid/restype"
	"github.com/xescugc/qid/qid/runner"
)

func TestPipeline_ResourceType(t *testing.T) {
	pp := &pipeline.Pipeline{
		ResourceTypes: []restype.ResourceType{
			{Name: "git"},
			{Name: "docker"},
		},
	}

	t.Run("finds existing resource type", func(t *testing.T) {
		rt, ok := pp.ResourceType("git")
		assert.True(t, ok)
		assert.Equal(t, "git", rt.Name)
	})

	t.Run("finds second resource type", func(t *testing.T) {
		rt, ok := pp.ResourceType("docker")
		assert.True(t, ok)
		assert.Equal(t, "docker", rt.Name)
	})

	t.Run("returns built-in cron type", func(t *testing.T) {
		rt, ok := pp.ResourceType("cron")
		assert.True(t, ok)
		assert.Equal(t, "cron", rt.Name)
		assert.Equal(t, "exec", rt.Check.Runner)
	})

	t.Run("returns false for unknown type", func(t *testing.T) {
		_, ok := pp.ResourceType("unknown")
		assert.False(t, ok)
	})
}

func TestPipeline_Resource(t *testing.T) {
	pp := &pipeline.Pipeline{
		Resources: []resource.Resource{
			{Canonical: "git.my-repo", Name: "my-repo"},
			{Canonical: "cron.timer", Name: "timer"},
		},
	}

	t.Run("finds existing resource", func(t *testing.T) {
		r, ok := pp.Resource("git.my-repo")
		assert.True(t, ok)
		assert.Equal(t, "my-repo", r.Name)
	})

	t.Run("returns false for unknown resource", func(t *testing.T) {
		_, ok := pp.Resource("nonexistent")
		assert.False(t, ok)
	})
}

func TestPipeline_Runner(t *testing.T) {
	pp := &pipeline.Pipeline{
		Runners: []runner.Runner{
			{Name: "docker"},
		},
	}

	t.Run("finds existing runner", func(t *testing.T) {
		r, ok := pp.Runner("docker")
		assert.True(t, ok)
		assert.Equal(t, "docker", r.Name)
	})

	t.Run("returns built-in exec runner", func(t *testing.T) {
		r, ok := pp.Runner("exec")
		assert.True(t, ok)
		assert.Equal(t, "exec", r.Name)
		assert.Equal(t, "$path", r.Run.Path)
		assert.Equal(t, []string{"$args"}, r.Run.Args)
	})

	t.Run("returns false for unknown runner", func(t *testing.T) {
		_, ok := pp.Runner("unknown")
		assert.False(t, ok)
	})
}
