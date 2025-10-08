package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lopezator/migrator"
	"github.com/xescugc/qid/qid/mysql/migrate/migrations"
)

// Migrate runs the migrations on the provided db
func Migrate(db *sql.DB, isSQLite bool) error {
	ms := make([]interface{}, 0, len(migrations.Migrations))
	for _, m := range migrations.Migrations {
		val := m
		ms = append(ms, &migrator.Migration{
			Name: val.Name,
			Func: func(tx *sql.Tx) error {
				if isSQLite {
					val.SQL = strings.Replace(val.SQL, "SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';", "", 1)
					val.SQL = strings.ReplaceAll(val.SQL, "id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,", "id INTEGER PRIMARY KEY,")
				}
				if _, err := tx.Exec(val.SQL); err != nil {
					return err
				}
				return nil
			},
		})
	}

	m, err := migrator.New(migrator.Migrations(ms...))
	if err != nil {
		return fmt.Errorf("error while creating the migration: %w", err)
	}

	if err := m.Migrate(db); err != nil {
		return fmt.Errorf("error while migrating: %w", err)
	}

	return nil
}
