package migrations

var V11PublicPipelines = Migration{
	Name: "V11PublicPipelines",
	SQL: `
		ALTER TABLE pipelines ADD COLUMN public BOOLEAN NOT NULL DEFAULT FALSE;
	`,
}
