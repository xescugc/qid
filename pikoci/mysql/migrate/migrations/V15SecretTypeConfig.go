package migrations

// V15SecretTypeConfig adds config column to secret_types and drops the secrets table.
// Secrets are now referenced inline at the step level instead of as separate entities.
var V15SecretTypeConfig = Migration{
	Name: "SecretTypeConfig",
	SQL: `
		ALTER TABLE secret_types ADD COLUMN config TEXT;
		DROP TABLE IF EXISTS secrets;
	`,
}
