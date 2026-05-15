package migrations

// V9OrderedPlan adds plan column to jobs and steps column to builds,
// replacing the separate get/task columns. Old columns are kept for
// backwards compatibility during the transition.
var V9OrderedPlan = Migration{
	Name: "OrderedPlan",
	SQL: `
		ALTER TABLE jobs ADD plan TEXT;
		ALTER TABLE builds ADD steps TEXT;
	`,
}
