package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cycloidio/sqlr"
)

// lastInsertedID extracts the id from the query result.
// If the entity was not created, i.e. id == 0, the
// UnexpectedErrorExternalSystemDB error will be returned.
func lastInsertedID(res sql.Result) (uint32, error) {
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted id: %w", err)
	}

	if id == 0 {
		return 0, fmt.Errorf("the entity was not created")
	}

	return uint32(id), nil
}

// isEntityFound returns whether the SQL query did affect any rows.
func isEntityFound(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected SQL rows: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

// toNullString returns sql.NullString. The string is considered valid if it's not empty.
func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// toNullBool returns sql.NullBool, that is always Valid
func toNullBool(b bool) sql.NullBool {
	return sql.NullBool{Bool: b, Valid: true}
}

// toNullInt64 returns sql.NullInt64. The int is considered valid if it's not equal 0.
func toNullInt64(i int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(i), Valid: i != 0}
}

// toNullTime returns sql.NullTIme. The time is considered valid if it's not equal Zero.
func toNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

func scanCount(s sqlr.Scanner) (int, error) {
	var c int

	err := s.Scan(
		&c,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to scan: %w", err)
	}

	return c, nil
}
