package migrations

// V18Concurrency adds a concurrency column to the jobs table.
var V18Concurrency = Migration{
	Name: "Concurrency",
	SQL:  `ALTER TABLE jobs ADD COLUMN concurrency INTEGER NOT NULL DEFAULT 0;`,
}
