package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cycloidio/sqlr"
	"github.com/xescugc/pikoci/pikoci/pipeline"
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
	rows, err := r.querier.QueryContext(ctx, pipelineQuery+`
		WHERE t.canonical = ? AND p.name = ?
	`, tc, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to query Pipeline: %w", err)
	}

	pps, err := scanPipelines(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Pipeline: %w", err)
	}

	if len(pps) == 0 {
		return nil, fmt.Errorf("not found")
	}

	return pps[0], nil
}

func (r *PipelineRepository) Filter(ctx context.Context, tc string) ([]*pipeline.Pipeline, error) {
	rows, err := r.querier.QueryContext(ctx, pipelineQuery+`
		WHERE t.canonical = ?
	`, tc)
	if err != nil {
		return nil, fmt.Errorf("failed to query Pipelines: %w", err)
	}

	pps, err := scanPipelines(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Pipelines: %w", err)
	}

	return pps, nil
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

const pipelineQuery = `
	SELECT
		p.id, p.name, p.raw,
		j.id, j.name, j.plan, j.on_success, j.on_failure, j.ensure,
		r.id, r.name, r.type, r.canonical, r.params, r.check_interval, r.cron_id, r.logs, r.last_check,
		rt.id, rt.name, rt.` + "`check`" + `, rt.pull, rt.push, rt.params,
		ru.id, ru.name, ru.run
	FROM pipelines AS p
	JOIN teams AS t ON p.team_id = t.id
	LEFT JOIN jobs AS j ON j.pipeline_id = p.id
	LEFT JOIN resources AS r ON r.pipeline_id = p.id
	LEFT JOIN resource_types AS rt ON rt.pipeline_id = p.id
	LEFT JOIN runners AS ru ON ru.pipeline_id = p.id
`

func scanPipelines(rows *sql.Rows) ([]*pipeline.Pipeline, error) {
	pipelineMap := make(map[uint32]*pipeline.Pipeline)
	var pipelineOrder []uint32

	// Track seen IDs to avoid duplicates from the JOIN
	type seenKey struct {
		pipelineID uint32
		table      string
		id         uint32
	}
	seen := make(map[seenKey]bool)

	for rows.Next() {
		var (
			pp  dbPipeline
			j   dbJob
			r   dbResource
			rt  dbResourceType
			ru  dbRunner
		)

		err := rows.Scan(
			&pp.ID, &pp.Name, &pp.Raw,
			&j.ID, &j.Name, &j.Plan, &j.OnSuccess, &j.OnFailure, &j.Ensure,
			&r.ID, &r.Name, &r.Type, &r.Canonical, &r.Params, &r.CheckInterval, &r.CronID, &r.Logs, &r.LastCheck,
			&rt.ID, &rt.Name, &rt.Check, &rt.Pull, &rt.Push, &rt.Params,
			&ru.ID, &ru.Name, &ru.Run,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}

		ppID := uint32(pp.ID.Int64)
		p, ok := pipelineMap[ppID]
		if !ok {
			p = pp.toDomainEntity()
			pipelineMap[ppID] = p
			pipelineOrder = append(pipelineOrder, ppID)
		}

		if j.ID.Valid {
			k := seenKey{ppID, "j", uint32(j.ID.Int64)}
			if !seen[k] {
				seen[k] = true
				p.Jobs = append(p.Jobs, *j.toDomainEntity())
			}
		}
		if r.ID.Valid {
			k := seenKey{ppID, "r", uint32(r.ID.Int64)}
			if !seen[k] {
				seen[k] = true
				p.Resources = append(p.Resources, *r.toDomainEntity())
			}
		}
		if rt.ID.Valid {
			k := seenKey{ppID, "rt", uint32(rt.ID.Int64)}
			if !seen[k] {
				seen[k] = true
				p.ResourceTypes = append(p.ResourceTypes, *rt.toDomainEntity())
			}
		}
		if ru.ID.Valid {
			k := seenKey{ppID, "ru", uint32(ru.ID.Int64)}
			if !seen[k] {
				seen[k] = true
				p.Runners = append(p.Runners, *ru.toDomainEntity())
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	result := make([]*pipeline.Pipeline, 0, len(pipelineOrder))
	for _, id := range pipelineOrder {
		result = append(result, pipelineMap[id])
	}
	return result, nil
}

