package migrations

// V0Initial is the first migration
var V0Initial = Migration{
	Name: "Initial",
	SQL: `
		SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

		CREATE TABLE IF NOT EXISTS pipelines (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				raw TEXT,

				CONSTRAINT uq__name UNIQUE ( name )
		);

		CREATE TABLE IF NOT EXISTS jobs (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				get TEXT,
				task TEXT,

				pipeline_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__name UNIQUE ( pipeline_id, name ),

				CONSTRAINT fk__jobs__pipelines
						FOREIGN KEY (pipeline_id) REFERENCES pipelines (id)
						ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS builds (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				get TEXT,
				task TEXT,
				status VARCHAR(255),
				error TEXT,

				job_id INT UNSIGNED NOT NULL,

				CONSTRAINT fk__builds__jobs
						FOREIGN KEY (job_id) REFERENCES jobs (id)
						ON DELETE CASCADE
		);
			
		CREATE TABLE IF NOT EXISTS resources (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				` + "`type`" + ` VARCHAR(255),
				inputs TEXT,
				logs TEXT,
				canonical VARCHAR(255),

				pipeline_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__pipeline__canonical UNIQUE ( pipeline_id, canonical ),

				CONSTRAINT fk__resources__pipelines
						FOREIGN KEY (pipeline_id) REFERENCES pipelines (id)
						ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS resource_versions (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				hash VARCHAR(255),

				resource_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__hash UNIQUE ( resource_id, hash ),

				CONSTRAINT fk__resource_versions__resources
						FOREIGN KEY (resource_id) REFERENCES resources (id)
						ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS resource_types (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				` + "`check`" + ` TEXT,
				pull TEXT,
				push TEXT,
				inputs TEXT,

				pipeline_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__name UNIQUE ( pipeline_id, name ),

				CONSTRAINT fk__resource_types__pipelines
						FOREIGN KEY (pipeline_id) REFERENCES pipelines (id)
						ON DELETE CASCADE
		);
	`,
}
