package migrations

// V16BuildGetVersions adds the build_get_versions table to track which
// resource version each get step consumed. Used by the scheduler to
// determine when downstream jobs with `passed` constraints are ready.
var V16BuildGetVersions = Migration{
	Name: "BuildGetVersions",
	SQL: `
		CREATE TABLE build_get_versions (
			build_id INT NOT NULL,
			step_name VARCHAR(255) NOT NULL,
			version_id INT NOT NULL,
			PRIMARY KEY (build_id, step_name),
			FOREIGN KEY (build_id) REFERENCES builds(id) ON DELETE CASCADE
		);
		CREATE INDEX idx_bgv_version ON build_get_versions(version_id);
	`,
}
