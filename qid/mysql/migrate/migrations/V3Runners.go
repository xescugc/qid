package migrations

// V3Runners adds the hooks for job and builds
var V3Runners = Migration{
	Name: "Runners",
	SQL: `
		CREATE TABLE IF NOT EXISTS runners (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				run TEXT,

				pipeline_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__name UNIQUE ( pipeline_id, name ),

				CONSTRAINT fk__runners__pipelines
						FOREIGN KEY (pipeline_id) REFERENCES pipelines (id)
						ON DELETE CASCADE
		);
	`,
}
