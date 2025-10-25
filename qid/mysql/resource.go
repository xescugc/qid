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
	ID            sql.NullInt64
	Name          sql.NullString
	Type          sql.NullString
	Canonical     sql.NullString
	Inputs        sql.NullString
	Logs          sql.NullString
	CheckInterval sql.NullString
	LastCheck     sql.NullTime
}

type dbResourceVersion struct {
	ID   sql.NullInt64
	Hash sql.NullString
}

func newDBResource(r resource.Resource) dbResource {
	i, _ := json.Marshal(r.Inputs)
	return dbResource{
		Name:          toNullString(r.Name),
		Type:          toNullString(r.Type),
		Canonical:     toNullString(r.Canonical),
		Inputs:        toNullString(string(i)),
		Logs:          toNullString(r.Logs),
		CheckInterval: toNullString(r.CheckInterval),
		LastCheck:     toNullTime(r.LastCheck),
	}
}

func (dbr *dbResource) toDomainEntity() *resource.Resource {
	r := &resource.Resource{
		ID:            uint32(dbr.ID.Int64),
		Name:          dbr.Name.String,
		Type:          dbr.Type.String,
		Canonical:     dbr.Canonical.String,
		Logs:          dbr.Logs.String,
		CheckInterval: dbr.CheckInterval.String,
		LastCheck:     dbr.LastCheck.Time,
	}

	_ = json.Unmarshal([]byte(dbr.Inputs.String), &r.Inputs)

	return r
}

func newDBResourceVersion(v resource.Version) dbResourceVersion {
	return dbResourceVersion{
		Hash: toNullString(v.Hash),
	}
}

func (dbrv *dbResourceVersion) toDomainEntity() *resource.Version {
	v := &resource.Version{
		ID:   uint32(dbrv.ID.Int64),
		Hash: dbrv.Hash.String,
	}

	return v
}

func (r *ResourceRepository) Create(ctx context.Context, pn string, rs resource.Resource) (uint32, error) {
	dbrs := newDBResource(rs)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO resources(name, `+"`type`"+`, canonical, inputs, check_interval, last_check, pipeline_id)
		VALUES (?, ?, ?, ?, ?, ?,
			-- pipeline_id
			(
				SELECT p.id
				FROM pipelines AS p
				WHERE p.name = ?
			))`, dbrs.Name, dbrs.Type, dbrs.Canonical, dbrs.Inputs, dbrs.CheckInterval, dbrs.LastCheck, pn)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *ResourceRepository) Update(ctx context.Context, pn, rCan string, rs resource.Resource) error {
	dbrs := newDBResource(rs)
	res, err := r.querier.ExecContext(ctx, `
		UPDATE resources AS r
		SET name = ?, type = ?, canonical = ?, inputs = ?, check_interval = ?, logs = ?, last_check = ?
		FROM (
			SELECT r.id
			FROM resources AS r
			JOIN pipelines AS p
				ON r.pipeline_id = p.id
			WHERE p.name = ? AND r.canonical = ?
		) AS rr
		WHERE rr.id = r.id
	`, dbrs.Name, dbrs.Type, dbrs.Canonical, dbrs.Inputs, dbrs.CheckInterval, dbrs.Logs, dbrs.LastCheck, pn, rCan)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	return nil
}

func (r *ResourceRepository) Find(ctx context.Context, pn, rCan string) (*resource.Resource, error) {
	row := r.querier.QueryRowContext(ctx, `
		SELECT r.id, r.name, r.type, r.canonical, r.inputs, r.check_interval, r.logs, r.last_check
		FROM resources AS r
		JOIN pipelines AS p
			ON r.pipeline_id = p.id
		WHERE p.name = ? AND r.canonical = ?
	`, pn, rCan)

	rs, err := scanResource(row)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Resource: %w", err)
	}

	return rs, nil
}

func (r *ResourceRepository) Filter(ctx context.Context, pn string) ([]*resource.Resource, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT r.id, r.name, r.type, r.canonical, r.inputs, r.check_interval, r.logs, r.last_check
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

func (r *ResourceRepository) CreateVersion(ctx context.Context, pn, rCan string, rv resource.Version) (uint32, error) {
	dbrv := newDBResourceVersion(rv)
	res, err := r.querier.ExecContext(ctx, `
		INSERT INTO resource_versions(hash, resource_id)
		VALUES (?, 
			-- resource_id
			(
				SELECT r.id
				FROM resources AS r
				JOIN pipelines AS p
					ON r.pipeline_id = p.id
				WHERE p.name = ? AND r.canonical = ?
			))`, dbrv.Hash, pn, rCan)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query: %w", err)
	}

	id, err := lastInsertedID(res)
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	return id, nil
}

func (r *ResourceRepository) FilterVersions(ctx context.Context, pn, rCan string) ([]*resource.Version, error) {
	rows, err := r.querier.QueryContext(ctx, `
		SELECT rv.id, rv.hash
		FROM resource_versions AS rv
		JOIN resources AS r
			ON rv.resource_id = r.id
		JOIN pipelines AS p
			ON r.pipeline_id = p.id
		WHERE p.name = ? AND r.canonical = ?
	`, pn, rCan)
	if err != nil {
		return nil, fmt.Errorf("failed to filter Resources: %w", err)
	}

	rvs, err := scanResourceVersions(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to filter jobs: %w", err)
	}

	return rvs, nil
}

func (r *ResourceRepository) Delete(ctx context.Context, pn, rCan string) error {
	res, err := r.querier.ExecContext(ctx, `
		DELETE r
		FROM resources
		WHERE id IN (
			SELECT r.id
			FROM resources AS r
			JOIN pipelines AS p
				ON r.pipeline_id = p.id
			WHERE p.name = ? AND r.canonical = ?
		)
	`, pn, rCan)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	err = isEntityFound(res)
	if err != nil {
		return fmt.Errorf("failed to delete the Job: %w", err)
	}

	return nil
}

func scanResource(s sqlr.Scanner) (*resource.Resource, error) {
	var r dbResource

	err := s.Scan(
		&r.ID,
		&r.Name,
		&r.Type,
		&r.Canonical,
		&r.Inputs,
		&r.CheckInterval,
		&r.Logs,
		&r.LastCheck,
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

func scanResourceVersion(s sqlr.Scanner) (*resource.Version, error) {
	var rv dbResourceVersion

	err := s.Scan(
		&rv.ID,
		&rv.Hash,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	return rv.toDomainEntity(), nil
}

func scanResourceVersions(rows *sql.Rows) ([]*resource.Version, error) {
	var rvs []*resource.Version

	for rows.Next() {
		rv, err := scanResourceVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource version: %w", err)
		}
		rvs = append(rvs, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan resource version: %w", err)
	}
	return rvs, nil
}
