package migrations

// V14Secrets adds secret_types and secrets tables.
var V14Secrets = Migration{
	Name: "Secrets",
	SQL: `
		CREATE TABLE IF NOT EXISTS secret_types (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				source VARCHAR(255),
				params TEXT,
				get TEXT,

				pipeline_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__secret_types__pipeline__name UNIQUE ( pipeline_id, name ),

				CONSTRAINT fk__secret_types__pipelines
						FOREIGN KEY (pipeline_id) REFERENCES pipelines (id)
						ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS secrets (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				` + "`type`" + ` VARCHAR(255),
				canonical VARCHAR(255),
				params TEXT,

				pipeline_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__secrets__pipeline__canonical UNIQUE ( pipeline_id, canonical ),

				CONSTRAINT fk__secrets__pipelines
						FOREIGN KEY (pipeline_id) REFERENCES pipelines (id)
						ON DELETE CASCADE
		);
	`,
}
