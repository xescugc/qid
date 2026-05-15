package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/xescugc/pikoci/pikoci/builtin"
	"github.com/xescugc/pikoci/pikoci/restype"
	"github.com/xescugc/pikoci/pikoci/runner"
)

const (
	pikoPrefix = "pikoci://"
	baseURL    = "https://raw.githubusercontent.com/xescugc/pikoci/master/pikoci/builtin"
)

type hclResourceType struct {
	ResourceTypes []restype.ResourceType `hcl:"resource_type,block"`
}

type hclRunner struct {
	Runners []runner.Runner `hcl:"runner,block"`
}

func ResolveResourceType(ctx context.Context, src string) (*restype.ResourceType, error) {
	data, err := resolveHCL(ctx, src, "resource_types")
	if err != nil {
		return nil, err
	}

	var hrt hclResourceType
	err = hclsimple.Decode("source.hcl", data, nil, &hrt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode resource type from source %q: %w", src, err)
	}
	if len(hrt.ResourceTypes) == 0 {
		return nil, fmt.Errorf("no resource_type block found in source %q", src)
	}
	return &hrt.ResourceTypes[0], nil
}

func ResolveRunner(ctx context.Context, src string) (*runner.Runner, error) {
	data, err := resolveHCL(ctx, src, "runners")
	if err != nil {
		return nil, err
	}

	var hr hclRunner
	err = hclsimple.Decode("source.hcl", data, nil, &hr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode runner from source %q: %w", src, err)
	}
	if len(hr.Runners) == 0 {
		return nil, fmt.Errorf("no runner block found in source %q", src)
	}
	return &hr.Runners[0], nil
}

func resolveHCL(ctx context.Context, src, kind string) ([]byte, error) {
	if strings.HasPrefix(src, pikoPrefix) {
		name := strings.TrimPrefix(src, pikoPrefix)
		// Try embedded built-in first
		switch kind {
		case "resource_types":
			if data, ok := builtin.ResourceTypeHCL(name); ok {
				return data, nil
			}
		case "runners":
			if data, ok := builtin.RunnerHCL(name); ok {
				return data, nil
			}
		}
		// Fall back to GitHub raw URL
		url := fmt.Sprintf("%s/%s/%s.hcl", baseURL, kind, name)
		return fetchURL(ctx, url)
	}

	if strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://") {
		return fetchURL(ctx, src)
	}

	return nil, fmt.Errorf("unsupported source URL scheme: %q", src)
}

func fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %q: %w", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %q: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %q: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %q: %w", url, err)
	}
	return data, nil
}
