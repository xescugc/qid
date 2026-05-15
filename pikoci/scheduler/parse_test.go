package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCheckInterval(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{"every 1m", "@every 1m", false},
		{"every 30s", "@every 30s", false},
		{"every 10s", "@every 10s", false},
		{"cron expression", "*/5 * * * *", false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseCheckInterval(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, s)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, s)
			}
		})
	}
}

func TestValidateCheckInterval(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{"valid every 1m", "@every 1m", false},
		{"valid every 30s", "@every 30s", false},
		{"valid every 10s", "@every 10s", false},
		{"too short every 5s", "@every 5s", true},
		{"too short every 1s", "@every 1s", true},
		{"valid cron every 5 min", "*/5 * * * *", false},
		{"valid cron hourly", "0 * * * *", false},
		{"invalid expression", "bad", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCheckInterval(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComputeNextCheck(t *testing.T) {
	from := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	next, err := ComputeNextCheck("@every 1m", from)
	require.NoError(t, err)
	assert.Equal(t, from.Add(1*time.Minute), next)

	next, err = ComputeNextCheck("@every 30s", from)
	require.NoError(t, err)
	assert.Equal(t, from.Add(30*time.Second), next)

	_, err = ComputeNextCheck("invalid", from)
	assert.Error(t, err)
}
