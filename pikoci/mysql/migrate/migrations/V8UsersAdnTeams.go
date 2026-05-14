package migrations

// V8UsersAdnTeams adds the Users and Teams structure
var V8UsersAdnTeams = Migration{
	Name: "UsersAndTeams",
	SQL: `
		CREATE TABLE IF NOT EXISTS users (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				full_name TEXT,
				username VARCHAR(255),
				password VARCHAR(255),
				admin BOOL,

				CONSTRAINT uq__username UNIQUE ( username )
		);
		INSERT INTO users (full_name, username, password, admin)
		VALUES ("Admin", "admin", "$2a$14$FoV/2Z0CRgQyiDJLMcErd.cC/DtWCKMWtxZEaL6HQd/rrtU2DZpAu", true);

		CREATE TABLE IF NOT EXISTS teams (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				canonical VARCHAR(255),

				CONSTRAINT uq__name UNIQUE ( name ),
				CONSTRAINT uq__canonical UNIQUE ( canonical )
		);
		INSERT INTO teams (name, canonical)
		VALUES ("Main", "main");

		CREATE TABLE IF NOT EXISTS teams_users (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,

				user_id INT UNSIGNED NOT NULL,
				team_id INT UNSIGNED NOT NULL,

				admin BOOL,

				CONSTRAINT uq__user_id__team_id UNIQUE ( user_id, team_id ),

				CONSTRAINT fk__teams_users__users
						FOREIGN KEY (user_id) REFERENCES users (id)
						ON DELETE CASCADE,

				CONSTRAINT fk__teams_users__teams
						FOREIGN KEY (team_id) REFERENCES teams (id)
						ON DELETE CASCADE
		);
		INSERT INTO teams_users (user_id, team_id, admin)
		VALUES (1, 1, true);

		CREATE TABLE IF NOT EXISTS pipelines_new (
				id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				raw TEXT,
				team_id INT UNSIGNED NOT NULL,

				CONSTRAINT uq__name__team_id UNIQUE ( name, team_id ),

				CONSTRAINT fk__pipelines__teams
					FOREIGN KEY (team_id) REFERENCES teams (id)
					ON DELETE CASCADE
		);
		
		INSERT INTO pipelines_new (id, name, raw, team_id) SELECT id, name, raw, 1 FROM pipelines;
		DROP TABLE pipelines;
		ALTER TABLE pipelines_new RENAME TO pipelines;
	`,
}
