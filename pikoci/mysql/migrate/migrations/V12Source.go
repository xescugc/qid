package migrations

var V12Source = Migration{
	Name: "V12Source",
	SQL: `
		ALTER TABLE resource_types ADD COLUMN source VARCHAR(512) DEFAULT NULL;
		ALTER TABLE runners ADD COLUMN source VARCHAR(512) DEFAULT NULL;
	`,
}
