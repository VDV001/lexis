package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWord_Review_CorrectAnswer(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}
	w.Review(5) // perfect
	assert.Equal(t, StatusConfident, w.Status)
	assert.True(t, w.NextReview.After(time.Now()))
	assert.GreaterOrEqual(t, w.EaseFactor, 2.5)
}

func TestWord_Review_WrongAnswer(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusConfident}
	w.Review(1) // wrong
	assert.Equal(t, StatusUnknown, w.Status)
	assert.Less(t, w.EaseFactor, 2.5)
}

func TestWord_Review_Quality3_Uncertain(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}
	w.Review(3)
	assert.Equal(t, StatusUncertain, w.Status)
	// Should schedule review ~1 day from now
	expectedMin := time.Now().Add(23 * time.Hour)
	assert.True(t, w.NextReview.After(expectedMin))
}

func TestWord_Review_Quality4_Uncertain(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}
	w.Review(4)
	assert.Equal(t, StatusUncertain, w.Status)
	// 6 * 2.5 = 15 days
	expectedMin := time.Now().Add(14 * 24 * time.Hour)
	assert.True(t, w.NextReview.After(expectedMin))
}

func TestWord_Review_EaseFactorFloor(t *testing.T) {
	w := &Word{EaseFactor: 1.3, Status: StatusUnknown}
	w.Review(0) // worst quality
	// Ease factor should not drop below 1.3
	assert.GreaterOrEqual(t, w.EaseFactor, 1.3)
}

func TestWord_Review_QualityClamped(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}
	w.Review(-1) // below range
	assert.Equal(t, StatusUnknown, w.Status)

	w2 := &Word{EaseFactor: 2.5, Status: StatusUnknown}
	w2.Review(10) // above range
	assert.Equal(t, StatusConfident, w2.Status)
}

func TestWord_Review_LastSeenUpdated(t *testing.T) {
	w := &Word{EaseFactor: 2.5}
	before := time.Now()
	w.Review(3)
	assert.True(t, !w.LastSeen.Before(before))
}
