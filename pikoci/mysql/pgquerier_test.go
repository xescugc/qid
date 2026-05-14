package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRewritePlaceholders(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no placeholders",
			input: "SELECT * FROM users",
			want:  "SELECT * FROM users",
		},
		{
			name:  "single placeholder",
			input: "SELECT * FROM users WHERE id = ?",
			want:  "SELECT * FROM users WHERE id = $1",
		},
		{
			name:  "multiple placeholders",
			input: "INSERT INTO users (name, age) VALUES (?, ?)",
			want:  "INSERT INTO users (name, age) VALUES ($1, $2)",
		},
		{
			name:  "many placeholders",
			input: "INSERT INTO t (a,b,c,d,e,f,g,h,i,j,k) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
			want:  "INSERT INTO t (a,b,c,d,e,f,g,h,i,j,k) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
		},
		{
			name:  "placeholder inside quotes not replaced",
			input: "SELECT * FROM users WHERE name = '?' AND id = ?",
			want:  "SELECT * FROM users WHERE name = '?' AND id = $1",
		},
		{
			name:  "backticks converted to double quotes",
			input: "SELECT `type`, `check` FROM resource_types WHERE id = ?",
			want:  `SELECT "type", "check" FROM resource_types WHERE id = $1`,
		},
		{
			name:  "backticks inside single quotes not replaced",
			input: "SELECT * FROM t WHERE name = '`test`'",
			want:  "SELECT * FROM t WHERE name = '`test`'",
		},
		{
			name:  "complex query with subquery",
			input: "UPDATE resources AS r SET name = ?, `type` = ? FROM (SELECT r.id FROM resources AS r WHERE r.canonical = ?) AS rr WHERE rr.id = r.id",
			want:  `UPDATE resources AS r SET name = $1, "type" = $2 FROM (SELECT r.id FROM resources AS r WHERE r.canonical = $3) AS rr WHERE rr.id = r.id`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rewritePlaceholders(tt.input))
		})
	}
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "1", itoa(1))
	assert.Equal(t, "9", itoa(9))
	assert.Equal(t, "10", itoa(10))
	assert.Equal(t, "99", itoa(99))
	assert.Equal(t, "100", itoa(100))
}
