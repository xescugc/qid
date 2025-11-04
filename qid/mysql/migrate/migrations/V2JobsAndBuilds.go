package migrations

// V2JobsAndBuilds adds the hooks for job and builds
var V2JobsAndBuilds = Migration{
	Name: "Initial",
	SQL: `
		ALTER TABLE builds
			ADD job TEXT;

		ALTER TABLE jobs
			ADD on_success TEXT;

		ALTER TABLE jobs
			ADD on_failure TEXT;

		ALTER TABLE jobs
			ADD ensure TEXT;
	`,
}
