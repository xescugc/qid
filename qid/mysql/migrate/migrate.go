package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lopezator/migrator"
	"github.com/xescugc/qid/qid/mysql"
	"github.com/xescugc/qid/qid/mysql/migrate/migrations"
)

// Migrate runs the migrations on the provided db
func Migrate(db *sql.DB, system string) error {
	ms := make([]interface{}, 0, len(migrations.Migrations))
	for _, m := range migrations.Migrations {
		val := m
		ms = append(ms, &migrator.Migration{
			Name: val.Name,
			Func: func(tx *sql.Tx) error {
				s := adaptSQL(val.SQL, system)
				if _, err := tx.Exec(s); err != nil {
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

func adaptSQL(sql, system string) string {
	switch system {
	case mysql.Mem, mysql.SQLite:
		sql = strings.Replace(sql, "SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';", "", 1)
		sql = strings.ReplaceAll(sql, "id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,", "id INTEGER PRIMARY KEY,")
	case mysql.PostgreSQL:
		sql = strings.Replace(sql, "SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';", "", 1)
		sql = strings.ReplaceAll(sql, "id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,", "id SERIAL PRIMARY KEY,")
		sql = strings.ReplaceAll(sql, "INT UNSIGNED NOT NULL", "INTEGER NOT NULL")
		sql = strings.ReplaceAll(sql, "INT UNSIGNED", "INTEGER")
		// Replace backtick-quoted identifiers with double-quote-quoted ones
		sql = strings.ReplaceAll(sql, "`type`", `"type"`)
		sql = strings.ReplaceAll(sql, "`check`", `"check"`)
		// PostgreSQL doesn't support RENAME COLUMN with the same syntax in older versions,
		// but ALTER TABLE ... RENAME COLUMN is standard and works in PG 9.6+
	case mysql.MySQL:
		// No changes needed for MySQL/MariaDB
	}
	return sql
}
