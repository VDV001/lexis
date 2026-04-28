package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDailySnapshot(t *testing.T) {
	userID := "user-123"
	language := "en"
	snapshotDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	total := 100
	confident := 40
	uncertain := 35
	unknown := 25

	snap := NewDailySnapshot(userID, language, snapshotDate, total, confident, uncertain, unknown)

	assert.Equal(t, userID, snap.UserID)
	assert.Equal(t, language, snap.Language)
	assert.Equal(t, snapshotDate, snap.SnapshotDate)
	assert.Equal(t, total, snap.TotalWords)
	assert.Equal(t, confident, snap.Confident)
	assert.Equal(t, uncertain, snap.Uncertain)
	assert.Equal(t, unknown, snap.Unknown)
}
