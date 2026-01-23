package migrations

// V5Duration adds the hooks for job and builds
var V5Duration = Migration{
	Name: "Duration",
	SQL: `
		ALTER TABLE builds 
			ADD started_at TIMESTAMP;

		ALTER TABLE builds 
			ADD duration INT;
	`,
}
