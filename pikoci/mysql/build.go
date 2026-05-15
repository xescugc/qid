package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
