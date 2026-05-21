package migrations

// V17BuildNumber adds a build_number column to the builds table.
// Build numbers are sequential per job (e.g. "1", "2", "3") and stored as
// strings to support future retry notation ("123.1", "123.2").
var V17BuildNumber = Migration{
	Name: "BuildNumber",
	SQL: `
		ALTER TABLE builds ADD COLUMN build_number VARCHAR(32) NOT NULL DEFAULT '';

		-- Backfill existing builds with sequential numbers per job, ordered by id.
		-- The integer result of ROW_NUMBER() is implicitly converted to the VARCHAR column.
		UPDATE builds
		SET build_number = (
			SELECT cnt FROM (
				SELECT b2.id, ROW_NUMBER() OVER (PARTITION BY b2.job_id ORDER BY b2.id) AS cnt
				FROM builds AS b2
			) AS numbered
			WHERE numbered.id = builds.id
		);

		CREATE UNIQUE INDEX idx_builds_job_build_number ON builds(job_id, build_number);
	`,
}
