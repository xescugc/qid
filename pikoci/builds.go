package pikoci

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/xescugc/pikoci/pikoci/build"
	"github.com/xescugc/pikoci/pikoci/utils"
)

func (q *PikoCI) CreateJobBuild(ctx context.Context, tc, pn, jn string, b build.Build) (*build.Build, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", jn)
	}

	id, buildNumber, err := q.Builds.Create(ctx, tc, pn, jn, b)
	if err != nil {
		return nil, fmt.Errorf("failed to Create Build: %w", err)
	}

	b.ID = id
	b.BuildNumber = buildNumber

	return &b, nil
}

func (q *PikoCI) ListJobBuilds(ctx context.Context, tc, pn, jn string) ([]*build.Build, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", jn)
	}

	builds, err := q.Builds.Filter(ctx, tc, pn, jn)
	if err != nil {
		return nil, fmt.Errorf("failed to list Builds: %w", err)
	}

	slices.Reverse(builds)

	return builds, nil
}

func (q *PikoCI) GetJobBuild(ctx context.Context, tc, pn, jn string, buildNumber string) (*build.Build, error) {
	if !utils.ValidateCanonical(tc) {
		return nil, fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return nil, fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return nil, fmt.Errorf("invalid Job Name format %q", jn)
	}

	b, err := q.Builds.Find(ctx, tc, pn, jn, buildNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to Find Build: %w", err)
	}
	return b, nil
}

func (q *PikoCI) CancelJobBuild(ctx context.Context, tc, pn, jn string, buildNumber string) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", jn)
	}

	b, err := q.Builds.Find(ctx, tc, pn, jn, buildNumber)
	if err != nil {
		return fmt.Errorf("failed to Find Build: %w", err)
	}
	if b.Status != build.Started {
		return fmt.Errorf("build %s is not running (status: %s)", buildNumber, b.Status)
	}
	b.Status = build.Cancelled
	b.Duration = time.Since(b.StartedAt)
	return q.Builds.Update(ctx, tc, pn, jn, buildNumber, *b)
}

func (q *PikoCI) UpdateJobBuild(ctx context.Context, tc, pn, jn string, buildNumber string, b build.Build) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", jn)
	}

	// Prevent worker from overwriting a cancelled build back to a non-terminal status
	existing, err := q.Builds.Find(ctx, tc, pn, jn, buildNumber)
	if err == nil && existing.Status == build.Cancelled {
		b.Status = build.Cancelled
	}

	if b.Status != build.Started && b.Duration == 0 {
		b.Duration = time.Since(b.StartedAt)
	}

	if err = q.Builds.Update(ctx, tc, pn, jn, buildNumber, b); err != nil {
		return fmt.Errorf("failed to Update Build: %w", err)
	}

	return nil
}

func (q *PikoCI) InsertBuildGetVersion(ctx context.Context, tc, pn, jn string, buildID uint32, stepName string, versionID uint32) error {
	return q.Builds.InsertGetVersion(ctx, tc, pn, jn, buildID, stepName, versionID)
}

func (q *PikoCI) DeleteJobBuild(ctx context.Context, tc, pn, jn string, buildNumber string) error {
	if !utils.ValidateCanonical(tc) {
		return fmt.Errorf("invalid Team Canonical format %q", tc)
	} else if !utils.ValidateCanonical(pn) {
		return fmt.Errorf("invalid Pipeline Name format %q", pn)
	} else if !utils.ValidateCanonical(jn) {
		return fmt.Errorf("invalid Job Name format %q", jn)
	}

	err := q.Builds.Delete(ctx, tc, pn, jn, buildNumber)
	if err != nil {
		return fmt.Errorf("failed to Delete Build: %w", err)
	}

	return nil
}
