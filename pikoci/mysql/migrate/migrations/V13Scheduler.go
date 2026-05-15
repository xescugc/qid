package migrations

// V13Scheduler replaces in-memory cron with DB-polling scheduler.
// Adds next_check column and drops the cron_id column.
var V13Scheduler = Migration{
	Name: "Scheduler",
	SQL: `
		ALTER TABLE resources ADD COLUMN next_check TIMESTAMP DEFAULT NULL;
		ALTER TABLE resources DROP COLUMN cron_id;
	`,
}
