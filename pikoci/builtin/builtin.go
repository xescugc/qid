package builtin

import (
	"embed"
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
)

//go:embed resource_types/*.hcl
var resourceTypeFS embed.FS

//go:embed runners/*.hcl
var runnerFS embed.FS

type hclResourceType struct {
	ResourceTypes []restype.ResourceType `hcl:"resource_type,block"`
}

type hclRunner struct {
	Runners []runner.Runner `hcl:"runner,block"`
}

var (
	resourceTypes     map[string]restype.ResourceType
	resourceTypesOnce sync.Once

	runners     map[string]runner.Runner
	runnersOnce sync.Once
)

func ResourceTypes() map[string]restype.ResourceType {
	resourceTypesOnce.Do(func() {
		resourceTypes = make(map[string]restype.ResourceType)
		entries, err := resourceTypeFS.ReadDir("resource_types")
		if err != nil {
			panic(fmt.Sprintf("failed to read embedded resource_types: %v", err))
		}
		for _, e := range entries {
			data, err := resourceTypeFS.ReadFile("resource_types/" + e.Name())
			if err != nil {
				panic(fmt.Sprintf("failed to read embedded resource_type %s: %v", e.Name(), err))
			}
			var hrt hclResourceType
			err = hclsimple.Decode(e.Name(), data, nil, &hrt)
			if err != nil {
				panic(fmt.Sprintf("failed to decode embedded resource_type %s: %v", e.Name(), err))
			}
			for _, rt := range hrt.ResourceTypes {
				resourceTypes[rt.Name] = rt
			}
		}
	})
	return resourceTypes
}

func Runners() map[string]runner.Runner {
	runnersOnce.Do(func() {
		runners = make(map[string]runner.Runner)
		entries, err := runnerFS.ReadDir("runners")
		if err != nil {
			panic(fmt.Sprintf("failed to read embedded runners: %v", err))
		}
		for _, e := range entries {
			data, err := runnerFS.ReadFile("runners/" + e.Name())
			if err != nil {
				panic(fmt.Sprintf("failed to read embedded runner %s: %v", e.Name(), err))
			}
			var hr hclRunner
			err = hclsimple.Decode(e.Name(), data, nil, &hr)
			if err != nil {
				panic(fmt.Sprintf("failed to decode embedded runner %s: %v", e.Name(), err))
			}
			for _, ru := range hr.Runners {
				runners[ru.Name] = ru
			}
		}
	})
	return runners
}

// ResourceTypeHCL returns the raw HCL bytes for a built-in resource type, if it exists.
func ResourceTypeHCL(name string) ([]byte, bool) {
	data, err := resourceTypeFS.ReadFile("resource_types/" + name + ".hcl")
	if err != nil {
		return nil, false
	}
	return data, true
}

// RunnerHCL returns the raw HCL bytes for a built-in runner, if it exists.
func RunnerHCL(name string) ([]byte, bool) {
	data, err := runnerFS.ReadFile("runners/" + name + ".hcl")
	if err != nil {
		return nil, false
	}
	return data, true
}
