package migrations

// V6ResourceVersion adds the hooks for job and builds
var V6ResourceVersion = Migration{
	Name: "ResourceVersion",
	SQL: `
		ALTER TABLE resource_versions
			RENAME COLUMN hash to version;
	`,
}
