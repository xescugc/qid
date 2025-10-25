package migrations

// V0Initial is the first migration
var V1ResourceCheckInterval = Migration{
	Name: "Initial",
	SQL: `
		ALTER TABLE resources
			ADD check_interval VARCHAR(255);
		ALTER TABLE resources
			ADD last_check TIMESTAMP;
	`,
}
