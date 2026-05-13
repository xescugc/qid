package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cycloidio/sqlr"
	"github.com/xescugc/qid/qid/pipeline"
)

type PipelineRepository struct {
	querier sqlr.Querier
}

func NewPipelineRepository(db sqlr.Querier) *PipelineRepository {
	return &PipelineRepository{
		querier: db,
	}
}

type dbPipeline struct {
	ID   sql.NullInt64
	Name sql.NullString
	Raw  sql.NullString
}

func newDBPipeline(p pipeline.Pipeline) dbPipeline {
	return dbPipeline{
		Name: toNullString(p.Name),
		Raw:  toNullString(string(p.Raw)),
	}
}

func (dbp *dbPipeline) toDomainEntity() *pipeline.Pipeline {
	return &pipeline.Pipeline{
		ID:   uint32(dbp.ID.Int64),
		Name: dbp.Name.String,
		Raw:  []byte(dbp.Raw.String),
	}
}

func (r *PipelineRepository) Create(ctx context.Context, tc string, p pipeline.Pipeline) (uint32, error) {
	dbp := newDBPipeline(p)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO pipelines(name, raw, team_id)
		VALUES (?, ?,
			-- pipeline_id
			(
				SELECT t.id
				FROM teams AS t
				WHERE t.canonical = ?
			))`, dbp.Name, dbp.Raw, tc)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *PipelineRepository) Update(ctx context.Context, tc, pn string, p pipeline.Pipeline) error {
	dbp := newDBPipeline(p)
	res, err := r.querier.ExecContext(ctx, `
		UPDATE pipelines AS p
		SET name = ?, raw = ?
		FROM (
			SELECT p.id
			FROM pipelines AS p
			JOIN teams AS t
				ON p.team_id = t.id
			WHERE t.canonical = ? AND p.name = ?
		) AS pp
		WHERE p.id = pp.id
	`, dbp.Name, dbp.Raw, tc, pn)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

func (r *PipelineRepository) Find(ctx context.Context, tc, pn string) (*pipeline.Pipeline, error) {
	row := r.querier.QueryRowContext(ctx, `
		SELECT p.id, p.name, p.raw
		FROM pipelines AS p
		JOIN teams AS t
			ON p.team_id = t.id
		WHERE t.canonical = ? AND p.name = ?
	`, tc, pn)

	p, err := scanPipeline(row)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Pipeline: %w", err)
	}

	return p, nil
}

func (r *PipelineRepository) Filter(ctx context.Context, tc string) ([]*pipeline.Pipeline, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT p.id, p.name, p.raw
		FROM pipelines AS p
		JOIN teams AS t
			ON p.team_id = t.id
		WHERE t.canonical = ?
	`, tc)
	if err != nil {
		return nil, fmt.Errorf("failed to filter Pipelines: %w", err)
	}

	ps, err := scanPipelines(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Pipeline: %w", err)
	}

	return ps, nil
}

func (r *PipelineRepository) Delete(ctx context.Context, tc, pn string) error {
	res, err := r.querier.ExecContext(ctx, `
		DELETE
		FROM pipelines
		WHERE id IN (
			SELECT p.id
			FROM pipelines AS p
			JOIN teams AS t
				ON p.team_id = t.id
			WHERE t.canonical = ? AND p.name = ?
		)
	`, tc, pn)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to delete the Pipeline: %w", err)
	}

	return nil
}

func scanPipeline(s sqlr.Scanner) (*pipeline.Pipeline, error) {
	var p dbPipeline

	err := s.Scan(
		&p.ID,
		&p.Name,
		&p.Raw,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	return p.toDomainEntity(), nil
}

func scanPipelines(rows *sql.Rows) ([]*pipeline.Pipeline, error) {
	var ps []*pipeline.Pipeline

	for rows.Next() {
		p, err := scanPipeline(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pipeline: %w", err)
		}
		ps = append(ps, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan pipeline: %w", err)
	}
	return ps, nil
}
