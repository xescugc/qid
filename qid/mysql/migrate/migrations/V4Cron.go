package migrations

// V4Cron adds the hooks for job and builds
var V4Cron = Migration{
	Name: "JobsAndBuilds",
	SQL: `
		ALTER TABLE resources
			ADD cron_id INT;
	`,
}
