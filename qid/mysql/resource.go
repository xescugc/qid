package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cycloidio/sqlr"
	"github.com/xescugc/qid/qid/resource"
)

type ResourceRepository struct {
	querier sqlr.Querier
}

func NewResourceRepository(db sqlr.Querier) *ResourceRepository {
	return &ResourceRepository{
		querier: db,
	}
}

type dbResource struct {
	ID     sql.NullInt64
	Name   sql.NullString
	Type   sql.NullString
	Inputs sql.NullString
}

func newDBResource(r resource.Resource) dbResource {
	i, _ := json.Marshal(r.Inputs)
	return dbResource{
		Name:   toNullString(r.Name),
		Type:   toNullString(r.Type),
		Inputs: toNullString(string(i)),
	}
}

func (dbr *dbResource) toDomainEntity() *resource.Resource {
	r := &resource.Resource{
		ID:   uint32(dbr.ID.Int64),
		Name: dbr.Name.String,
		Type: dbr.Type.String,
	}

	_ = json.Unmarshal([]byte(dbr.Inputs.String), &r.Inputs)

	return r
}

func (r *ResourceRepository) Create(ctx context.Context, pn string, rs resource.Resource) (uint32, error) {
	dbrs := newDBResource(rs)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO resources(name, `+"`type`"+`, inputs, pipeline_id)
		VALUES (?, ?, ?,
			-- pipeline_id
			(
				SELECT p.id
				FROM pipelines AS p
				WHERE p.name = ?
			))`, dbrs.Name, dbrs.Type, dbrs.Inputs, pn)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *ResourceRepository) Find(ctx context.Context, pn, rn, rt string) (*resource.Resource, error) {
	row := r.querier.QueryRowContext(ctx, `
		SELECT r.id, r.name, r.type, r.inputs
		FROM resources AS r
		JOIN pipelines AS p
			ON r.pipeline_id = p.id
		WHERE p.name = ? AND r.name = ? AND r.type = ?
	`, pn, rn, rt)

	rs, err := scanResource(row)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Resource: %w", err)
	}

	return rs, nil
}

func (r *ResourceRepository) Filter(ctx context.Context, pn string) ([]*resource.Resource, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT r.id, r.name, r.type, r.inputs
		FROM resources AS r
		JOIN pipelines AS p
			ON r.pipeline_id = p.id
		WHERE p.name = ?
	`, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to filter Resources: %w", err)
	}

	resources, err := scanResources(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to filter jobs: %w", err)
	}

	return resources, nil
}

func scanResource(s sqlr.Scanner) (*resource.Resource, error) {
	var r dbResource

	err := s.Scan(
		&r.ID,
		&r.Name,
		&r.Type,
		&r.Inputs,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	return r.toDomainEntity(), nil
}

func scanResources(rows *sql.Rows) ([]*resource.Resource, error) {
	var rs []*resource.Resource

	for rows.Next() {
		r, err := scanResource(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource: %w", err)
		}
		rs = append(rs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan resource: %w", err)
	}
	return rs, nil
}
