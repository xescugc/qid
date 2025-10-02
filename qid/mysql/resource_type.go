package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cycloidio/sqlr"
	"github.com/xescugc/qid/qid/restype"
)

type ResourceTypeRepository struct {
	querier sqlr.Querier
}

func NewResourceTypeRepository(db sqlr.Querier) *ResourceTypeRepository {
	return &ResourceTypeRepository{
		querier: db,
	}
}

type dbResourceType struct {
	ID     sql.NullInt64
	Name   sql.NullString
	Inputs sql.NullString
	Check  sql.NullString
	Pull   sql.NullString
	Push   sql.NullString
}

func newDBResourceType(rt restype.ResourceType) dbResourceType {
	i, _ := json.Marshal(rt.Inputs)
	c, _ := json.Marshal(rt.Check)
	pl, _ := json.Marshal(rt.Pull)
	ps, _ := json.Marshal(rt.Push)
	return dbResourceType{
		Name:   toNullString(rt.Name),
		Inputs: toNullString(string(i)),
		Check:  toNullString(string(c)),
		Pull:   toNullString(string(pl)),
		Push:   toNullString(string(ps)),
	}
}

func (dbrt *dbResourceType) toDomainEntity() *restype.ResourceType {
	rt := &restype.ResourceType{
		ID:   uint32(dbrt.ID.Int64),
		Name: dbrt.Name.String,
	}

	_ = json.Unmarshal([]byte(dbrt.Inputs.String), &rt.Inputs)
	_ = json.Unmarshal([]byte(dbrt.Check.String), &rt.Check)
	_ = json.Unmarshal([]byte(dbrt.Pull.String), &rt.Pull)
	_ = json.Unmarshal([]byte(dbrt.Push.String), &rt.Push)

	return rt
}

func (r *ResourceTypeRepository) Create(ctx context.Context, pn string, rt restype.ResourceType) (uint32, error) {
	dbrt := newDBResourceType(rt)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO resource_types(name, `+"`check`"+`, pull, push, inputs, pipeline_id)
		VALUES (?, ?, ?, ?, ?,
			-- pipeline_id
			(
				SELECT p.id
				FROM pipelines AS p
				WHERE p.name = ?
			))`, dbrt.Name, dbrt.Check, dbrt.Pull, dbrt.Push, dbrt.Inputs, pn)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *ResourceTypeRepository) Find(ctx context.Context, pn, rtn string) (*restype.ResourceType, error) {
	row := r.querier.QueryRowContext(ctx, `
		SELECT rt.id, rt.name, rt.check, rt.pull, rt.push, rt.inputs
		FROM resource_types AS rt
		JOIN pipelines AS p
			ON rt.pipeline_id = p.id
		WHERE p.name = ? AND rt.name = ?
	`, pn, rtn)

	rt, err := scanResourceType(row)
	if err != nil {
		return nil, fmt.Errorf("failed to scan ResourceType: %w", err)
	}

	return rt, nil
}

func (r *ResourceTypeRepository) Filter(ctx context.Context, pn string) ([]*restype.ResourceType, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT rt.id, rt.name, rt.check, rt.pull, rt.push, rt.inputs
		FROM resource_types AS rt
		JOIN pipelines AS p
			ON rt.pipeline_id = p.id
		WHERE p.name = ?
	`, pn)
	if err != nil {
		return nil, fmt.Errorf("failed to filter ResourceTypes: %w", err)
	}

	restypes, err := scanResourceTypes(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to filter jobs: %w", err)
	}

	return restypes, nil
}

func scanResourceType(s sqlr.Scanner) (*restype.ResourceType, error) {
	var rt dbResourceType

	err := s.Scan(
		&rt.ID,
		&rt.Name,
		&rt.Check,
		&rt.Pull,
		&rt.Push,
		&rt.Inputs,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	return rt.toDomainEntity(), nil
}

func scanResourceTypes(rows *sql.Rows) ([]*restype.ResourceType, error) {
	var rts []*restype.ResourceType

	for rows.Next() {
		rt, err := scanResourceType(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource: %w", err)
		}
		rts = append(rts, rt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan resource: %w", err)
	}
	return rts, nil
}
