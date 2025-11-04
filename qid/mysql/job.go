package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cycloidio/sqlr"
	"github.com/xescugc/qid/qid/job"
)

type JobRepository struct {
	querier sqlr.Querier
}

func NewJobRepository(db sqlr.Querier) *JobRepository {
	return &JobRepository{
		querier: db,
	}
}

type dbJob struct {
	ID        sql.NullInt64
	Name      sql.NullString
	Get       sql.NullString
	Task      sql.NullString
	OnSuccess sql.NullString
	OnFailure sql.NullString
	Ensure    sql.NullString
}

func newDBJob(p job.Job) dbJob {
	g, _ := json.Marshal(p.Get)
	t, _ := json.Marshal(p.Task)
	s, _ := json.Marshal(p.OnSuccess)
	f, _ := json.Marshal(p.OnFailure)
	e, _ := json.Marshal(p.Ensure)
	return dbJob{
		Name:      toNullString(p.Name),
		Get:       toNullString(string(g)),
		Task:      toNullString(string(t)),
		OnSuccess: toNullString(string(s)),
		OnFailure: toNullString(string(f)),
		Ensure:    toNullString(string(e)),
	}
}

func (dbp *dbJob) toDomainEntity() *job.Job {
	j := &job.Job{
		ID:   uint32(dbp.ID.Int64),
		Name: dbp.Name.String,
	}

	_ = json.Unmarshal([]byte(dbp.Get.String), &j.Get)
	_ = json.Unmarshal([]byte(dbp.Task.String), &j.Task)
	_ = json.Unmarshal([]byte(dbp.OnSuccess.String), &j.OnSuccess)
	_ = json.Unmarshal([]byte(dbp.OnFailure.String), &j.OnFailure)
	_ = json.Unmarshal([]byte(dbp.Ensure.String), &j.Ensure)

	return j
}

func (r *JobRepository) Create(ctx context.Context, pn string, j job.Job) (uint32, error) {
	dbj := newDBJob(j)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO jobs(name, get, task, on_success, on_failure, ensure, pipeline_id)
		VALUES (?, ?, ?, ?, ?, ?,
			-- pipeline_id
			(
				SELECT p.id
				FROM pipelines AS p
				WHERE p.name = ?
			))`, dbj.Name, dbj.Get, dbj.Task, dbj.OnSuccess, dbj.OnFailure, dbj.Ensure, pn)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *JobRepository) Update(ctx context.Context, pn, jn string, j job.Job) error {
	dbj := newDBJob(j)
	res, err := r.querier.ExecContext(ctx, `
		UPDATE jobs AS j
		SET name = ?, get = ?, task = ?, on_success = ?, on_failure = ?, ensure = ?
		FROM (
			SELECT j.id
			FROM jobs AS j
			JOIN pipelines AS p
				ON j.pipeline_id = p.id
			WHERE p.name = ? AND j.name = ?
		) AS jj
		WHERE jj.id = j.id
	`, dbj.Name, dbj.Get, dbj.Task, dbj.OnSuccess, dbj.OnFailure, dbj.Ensure, pn, jn)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

func (r *JobRepository) Find(ctx context.Context, pn, jn string) (*job.Job, error) {
	row := r.querier.QueryRowContext(ctx, `
		SELECT j.id, j.name, j.get, j.task, j.on_success, j.on_failure, j.ensure
		FROM jobs AS j
		JOIN pipelines AS p
			ON j.pipeline_id = p.id
		WHERE p.name = ? AND j.name = ?
	`, pn, jn)

	j, err := scanJob(row)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Job: %w", err)
	}

	return j, nil
}

func (r *JobRepository) Filter(ctx context.Context, pn string) ([]*job.Job, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT j.id, j.name, j.get, j.task, j.on_success, j.on_failure, j.ensure
		FROM jobs AS j
		JOIN pipelines AS p
			ON j.pipeline_id = p.id
		WHERE p.name = ?
	`, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to filter jobs: %w", err)
	}

	jobs, err := scanJobs(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to filter jobs: %w", err)
	}

	return jobs, nil
}

func (r *JobRepository) Delete(ctx context.Context, pn, jn string) error {
	res, err := r.querier.ExecContext(ctx, `
		DELETE
		FROM jobs
		WHERE id IN (
			SELECT j.id
			FROM jobs AS j
			JOIN pipelines AS p
				ON j.pipeline_id = p.id
			WHERE p.name = ? AND j.name = ?
		)
	`, pn, jn)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to delete the Job: %w", err)
	}

	return nil
}

func scanJob(s sqlr.Scanner) (*job.Job, error) {
	var j dbJob

	err := s.Scan(
		&j.ID,
		&j.Name,
		&j.Get,
		&j.Task,
		&j.OnSuccess,
		&j.OnFailure,
		&j.Ensure,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	return j.toDomainEntity(), nil
}

func scanJobs(rows *sql.Rows) ([]*job.Job, error) {
	var js []*job.Job

	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		js = append(js, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan job: %w", err)
	}
	return js, nil
}
