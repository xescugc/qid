package mysql

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToNullString(t *testing.T) {
	ns := toNullString("hello")
	assert.True(t, ns.Valid)
	assert.Equal(t, "hello", ns.String)

	ns = toNullString("")
	assert.False(t, ns.Valid)
	assert.Equal(t, "", ns.String)
}

func TestToNullBool(t *testing.T) {
	nb := toNullBool(true)
	assert.True(t, nb.Valid)
	assert.True(t, nb.Bool)

	nb = toNullBool(false)
	assert.True(t, nb.Valid)
	assert.False(t, nb.Bool)
}

func TestToNullInt64(t *testing.T) {
	ni := toNullInt64(42)
	assert.True(t, ni.Valid)
	assert.Equal(t, int64(42), ni.Int64)

	ni = toNullInt64(0)
	assert.False(t, ni.Valid)
	assert.Equal(t, int64(0), ni.Int64)
}

func TestToNullTime(t *testing.T) {
	now := time.Now()
	nt := toNullTime(now)
	assert.True(t, nt.Valid)
	assert.Equal(t, now, nt.Time)

	nt = toNullTime(time.Time{})
	assert.False(t, nt.Valid)
}

func TestIsEntityFound(t *testing.T) {
	err := isEntityFound(mockResult{rowsAffected: 1})
	assert.NoError(t, err)

	err = isEntityFound(mockResult{rowsAffected: 0})
	assert.Error(t, err)
}

func TestLastInsertedID(t *testing.T) {
	id, err := lastInsertedID(mockResult{lastInsertID: 5})
	assert.NoError(t, err)
	assert.Equal(t, uint32(5), id)

	_, err = lastInsertedID(mockResult{lastInsertID: 0})
	assert.Error(t, err)
}

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m mockResult) LastInsertId() (int64, error) { return m.lastInsertID, nil }
func (m mockResult) RowsAffected() (int64, error) { return m.rowsAffected, nil }

var _ sql.Result = mockResult{}
