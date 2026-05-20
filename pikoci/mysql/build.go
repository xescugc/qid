package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cycloidio/sqlr"
	"github.com/xescugc/pikoci/pikoci/build"
)

type BuildRepository struct {
	querier sqlr.Querier
}

func NewBuildRepository(db sqlr.Querier) *BuildRepository {
	return &BuildRepository{
		querier: db,
	}
}

type dbBuild struct {
	ID        sql.NullInt64
	Steps     sql.NullString
	Job       sql.NullString
	Status    sql.NullString
	Error     sql.NullString
	StartedAt sql.NullTime
	Duration  sql.NullInt64
}

func newDBBuild(b build.Build) dbBuild {
	s, _ := json.Marshal(b.Steps)
	j, _ := json.Marshal(b.Job)
	return dbBuild{
		Steps:     toNullString(string(s)),
		Job:       toNullString(string(j)),
		Status:    toNullString(b.Status.String()),
		Error:     toNullString(b.Error),
		StartedAt: toNullTime(b.StartedAt),
		Duration:  toNullInt64(int(b.Duration)),
	}
}

func (dbb *dbBuild) toDomainEntity() *build.Build {
	s, _ := build.StatusString(dbb.Status.String)
	b := &build.Build{
		ID:        uint32(dbb.ID.Int64),
		Status:    s,
		Error:     dbb.Error.String,
		StartedAt: dbb.StartedAt.Time,
		Duration:  time.Duration(dbb.Duration.Int64),
	}

	_ = json.Unmarshal([]byte(dbb.Steps.String), &b.Steps)
	_ = json.Unmarshal([]byte(dbb.Job.String), &b.Job)

	return b
}

func (r *BuildRepository) Create(ctx context.Context, tc, pn, jn string, b build.Build) (uint32, error) {
	dbb := newDBBuild(b)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO builds(steps, job, status, error, started_at, duration, job_id)
		VALUES (?, ?, ?, ?, ?, ?,
			-- job_id
			(
				SELECT j.id
				FROM jobs AS j
				JOIN pipelines AS p
					ON j.pipeline_id = p.id
				JOIN teams AS t
					ON p.team_id = t.id
				WHERE t.canonical = ? AND p.name = ? AND j.name = ?
			))`, dbb.Steps, dbb.Job, dbb.Status, dbb.Error, dbb.StartedAt, dbb.Duration, tc, pn, jn)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *BuildRepository) Find(ctx context.Context, tc, pn, jn string, bID uint32) (*build.Build, error) {
	row := r.querier.QueryRowContext(ctx, `
		SELECT b.id, b.steps, b.job, b.status, b.error, b.started_at, b.duration
		FROM builds AS b
		JOIN jobs AS j
			ON b.job_id = j.id
		JOIN pipelines AS p
			ON j.pipeline_id = p.id
		JOIN teams AS t
			ON p.team_id = t.id
		WHERE tc.canonical = ? AND p.name = ? AND j.name = ? AND b.id = ?
	`, tc, pn, jn, bID)

	j, err := scanBuild(row)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Build: %w", err)
	}

	return j, nil
}

func (r *BuildRepository) Filter(ctx context.Context, tc, pn, jn string) ([]*build.Build, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT b.id, b.steps, b.job, b.status, b.error, b.started_at, b.duration
		FROM builds AS b
		JOIN jobs AS j
			ON b.job_id = j.id
		JOIN pipelines AS p
			ON j.pipeline_id = p.id
		JOIN teams AS t
			ON p.team_id = t.id
		WHERE t.canonical = ? AND p.name = ? AND j.name = ?
	`, tc, pn, jn)
	if err != nil {
		return nil, fmt.Errorf("failed to filter builds: %w", err)
	}

	builds, err := scanBuilds(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to filter builds: %w", err)
	}

	return builds, nil
}

func (r *BuildRepository) Update(ctx context.Context, tc, pn, jn string, bID uint32, b build.Build) error {
	dbb := newDBBuild(b)
	res, err := r.querier.ExecContext(ctx, `
		UPDATE builds AS b
		SET steps = ?, job = ?, status = ?, error = ?, started_at = ?, duration = ?
		FROM (
			SELECT b.id
			FROM builds AS b
			JOIN jobs AS j
				ON b.job_id = j.id
			JOIN pipelines AS p
				ON j.pipeline_id = p.id
			JOIN teams AS t
				ON p.team_id = t.id
			WHERE t.canonical = ? AND p.name = ? AND j.name = ? AND b.id = ?
		) AS bb
		WHERE bb.id = b.id
	`, dbb.Steps, dbb.Job, dbb.Status, dbb.Error, dbb.StartedAt, dbb.Duration, tc, pn, jn, bID, bID)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to update build: %w", err)
	}

	return nil
}

func (r *BuildRepository) Delete(ctx context.Context, tc, pn, jn string, bID uint32) error {
	res, err := r.querier.ExecContext(ctx, `
		DELETE
		FROM builds
		WHERE id IN (
			SELECT b.id
			FROM builds AS b
			JOIN jobs AS j
				ON b.job_id = j.id
			JOIN pipelines AS p
				ON j.pipeline_id = p.id
			JOIN teams AS t
				ON p.team_id = t.id
			WHERE t.canonical = ? AND p.name = ? AND j.name = ? AND b.id
		)
	`, tc, pn, jn, bID)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to delete the Job: %w", err)
	}

	return nil
}

func (r *BuildRepository) InsertGetVersion(ctx context.Context, tc, pn, jn string, buildID uint32, stepName string, versionID uint32) error {
	_, err := r.querier.ExecContext(ctx, `
		INSERT OR IGNORE INTO build_get_versions(build_id, step_name, version_id)
		VALUES (?, ?, ?)
	`, buildID, stepName, versionID)
	if err != nil {
		return fmt.Errorf("failed to insert build get version: %w", err)
	}
	return nil
}

// FindReadyDownstreamVersion finds the highest version_id that ALL upstream
// jobs succeeded with but the downstream job hasn't built yet (regardless of
// downstream build status). This means if a downstream build consumed a version
// and then failed, that version will NOT be retried — preventing infinite retry
// loops. Manual re-trigger can still be used for failed builds.
func (r *BuildRepository) FindReadyDownstreamVersion(ctx context.Context, tc, pn string, upstreamJobs []string, downstreamJob string, stepName string, upstreamCount int) (uint32, bool, error) {
	// Build the IN clause placeholders
	placeholders := make([]string, len(upstreamJobs))
	args := make([]interface{}, 0, len(upstreamJobs)+5)
	args = append(args, tc, pn)
	for i, j := range upstreamJobs {
		placeholders[i] = "?"
		args = append(args, j)
	}
	args = append(args, stepName)
	// Args for the NOT IN subquery
	args = append(args, downstreamJob, stepName)
	// HAVING count
	args = append(args, upstreamCount)

	query := `
		SELECT bgv.version_id
		FROM build_get_versions bgv
		JOIN builds b ON bgv.build_id = b.id
		JOIN jobs j ON b.job_id = j.id
		JOIN pipelines p ON j.pipeline_id = p.id
		JOIN teams t ON p.team_id = t.id
		WHERE t.canonical = ? AND p.name = ? AND b.status = 'succeeded'
		  AND j.name IN (` + strings.Join(placeholders, ", ") + `)
		  AND bgv.step_name = ?
		  AND bgv.version_id NOT IN (
			  SELECT bgv2.version_id FROM build_get_versions bgv2
			  JOIN builds b2 ON bgv2.build_id = b2.id
			  JOIN jobs j2 ON b2.job_id = j2.id
			  WHERE j2.pipeline_id = p.id AND j2.name = ?
				AND bgv2.step_name = ?
		  )
		GROUP BY bgv.version_id
		HAVING COUNT(DISTINCT j.name) = ?
		ORDER BY bgv.version_id DESC
		LIMIT 1
	`

	var versionID uint32
	err := r.querier.QueryRowContext(ctx, query, args...).Scan(&versionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to find ready downstream version: %w", err)
	}
	return versionID, true, nil
}

func (r *BuildRepository) LastBuildAtByPipeline(ctx context.Context, tc string) (map[uint32]time.Time, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT p.id, MAX(b.started_at)
		FROM builds AS b
		JOIN jobs AS j ON b.job_id = j.id
		JOIN pipelines AS p ON j.pipeline_id = p.id
		JOIN teams AS t ON p.team_id = t.id
		WHERE t.canonical = ?
		GROUP BY p.id
	`, tc)
	if err != nil {
		return nil, fmt.Errorf("failed to query last build timestamps: %w", err)
	}
	defer rows.Close()

	result := make(map[uint32]time.Time)
	for rows.Next() {
		var pipelineID uint32
		var startedAt sql.NullTime
		if err := rows.Scan(&pipelineID, &startedAt); err != nil {
			return nil, fmt.Errorf("failed to scan last build timestamp: %w", err)
		}
		if startedAt.Valid {
			result[pipelineID] = startedAt.Time
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate last build timestamps: %w", err)
	}

	return result, nil
}

func scanBuild(s sqlr.Scanner) (*build.Build, error) {
	var b dbBuild

	err := s.Scan(
		&b.ID,
		&b.Steps,
		&b.Job,
		&b.Status,
		&b.Error,
		&b.StartedAt,
		&b.Duration,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	return b.toDomainEntity(), nil
}

func scanBuilds(rows *sql.Rows) ([]*build.Build, error) {
	var bs []*build.Build

	for rows.Next() {
		b, err := scanBuild(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan build: %w", err)
		}
		bs = append(bs, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan build: %w", err)
	}
	return bs, nil
}
