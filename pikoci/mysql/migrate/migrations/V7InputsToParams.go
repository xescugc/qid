package migrations

// V7InputsToParams adds the hooks for job and builds
var V7InputsToParams = Migration{
	Name: "ResourceVersion",
	SQL: `
		ALTER TABLE resources
			RENAME COLUMN inputs to params;

		ALTER TABLE resource_types
			RENAME COLUMN inputs to params;
	`,
}
