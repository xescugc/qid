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
}

func newDBPipeline(o pipeline.Pipeline) dbPipeline {
	return dbPipeline{
		Name: toNullString(o.Name),
	}
}

func (dbp *dbPipeline) toDomainEntity() *pipeline.Pipeline {
	return &pipeline.Pipeline{
		ID:   uint32(dbp.ID.Int64),
		Name: dbp.Name.String,
	}
}

func (r *PipelineRepository) Create(ctx context.Context, p pipeline.Pipeline) (uint32, error) {
	dbp := newDBPipeline(p)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO pipelines(name)
		VALUES (?)
	`, dbp.Name)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *PipelineRepository) Find(ctx context.Context, pn string) (*pipeline.Pipeline, error) {
	row := r.querier.QueryRowContext(ctx, `
		SELECT p.id, p.name
		FROM pipelines AS p
		WHERE p.name = ?
	`, pn)

	p, err := scanPipeline(row)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Pipeline: %w", err)
	}

	return p, nil
}

func (r *PipelineRepository) Filter(ctx context.Context) ([]*pipeline.Pipeline, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT p.id, p.name
		FROM pipelines AS p
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to filter Pipelines: %w", err)
	}

	ps, err := scanPipelines(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Pipeline: %w", err)
	}

	return ps, nil
}

func (r *PipelineRepository) Delete(ctx context.Context, pn string) error {
	res, err := r.querier.ExecContext(ctx, `
		DELETE
		FROM pipelines AS p
		WHERE p.name = ?
	`, pn)
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
