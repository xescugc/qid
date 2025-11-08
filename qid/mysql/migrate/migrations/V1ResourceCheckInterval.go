package migrations

// V1ResourceCheckInterval adds the check_interval and last_check to resources
var V1ResourceCheckInterval = Migration{
	Name: "ResourceCheckInterval",
	SQL: `
		ALTER TABLE resources
			ADD check_interval VARCHAR(255);
		ALTER TABLE resources
			ADD last_check TIMESTAMP;
	`,
}
