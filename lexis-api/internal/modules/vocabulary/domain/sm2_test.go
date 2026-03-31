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

func TestWord_Review_EaseFactorFloor_RepeatedWrong(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusConfident}
	// Repeatedly answer wrong — ease factor must never drop below 1.3
	for i := 0; i < 20; i++ {
		w.Review(0)
		assert.GreaterOrEqual(t, w.EaseFactor, 1.3,
			"ease factor dropped below 1.3 on iteration %d", i)
	}
	assert.Equal(t, 1.3, w.EaseFactor)
}

func TestWord_Review_TransitionUnknownToUncertainToConfident(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}

	// Quality 3 → uncertain
	w.Review(3)
	assert.Equal(t, StatusUncertain, w.Status)

	// Quality 5 → confident
	w.Review(5)
	assert.Equal(t, StatusConfident, w.Status)
	assert.True(t, w.NextReview.After(time.Now()))
}

func TestWord_Review_MultipleSequentialReviews(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}

	// First review: quality 4 → uncertain
	w.Review(4)
	assert.Equal(t, StatusUncertain, w.Status)
	firstReview := w.NextReview

	// Second review: quality 5 → confident, further out
	w.Review(5)
	assert.Equal(t, StatusConfident, w.Status)
	assert.True(t, w.NextReview.After(firstReview),
		"confident review should schedule further out than uncertain")

	// Third review: quality 1 → reset to unknown
	w.Review(1)
	assert.Equal(t, StatusUnknown, w.Status)
	assert.True(t, w.NextReview.Before(firstReview),
		"wrong answer should schedule sooner than previous reviews")
}

func TestWord_Review_EaseFactorIncreasesWithPerfectScores(t *testing.T) {
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}
	initial := w.EaseFactor

	w.Review(5)
	assert.Greater(t, w.EaseFactor, initial,
		"ease factor should increase with perfect quality")

	second := w.EaseFactor
	w.Review(5)
	assert.Greater(t, w.EaseFactor, second,
		"ease factor should keep increasing with repeated perfect quality")
}
