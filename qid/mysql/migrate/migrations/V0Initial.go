package migrations

// V0Initial is the first migration
var V0Initial = Migration{
	Name: "Initial",
	SQL: `
		SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

		CREATE TABLE IF NOT EXISTS pipelines (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),

				CONSTRAINT uq__name UNIQUE ( name )
		);

		CREATE TABLE IF NOT EXISTS jobs (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				plan TEXT,

				pipeline_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__name UNIQUE ( pipeline_id, name ),

				CONSTRAINT fk__jobs__pipelines
						FOREIGN KEY (pipeline_id) REFERENCES pipelines (id)
						ON DELETE CASCADE
		);
	`,
}
