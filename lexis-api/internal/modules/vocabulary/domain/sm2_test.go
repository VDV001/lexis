package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWord_Review(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		easeFactor     float64
		initialStatus  VocabStatus
		quality        int
		wantStatus     VocabStatus
		wantMinEase    float64
		wantMaxEase    float64
		wantAfter      time.Time
		wantBefore     time.Time
		wantLastSeen   time.Time
	}{
		{
			name:          "quality 5 — confident, ease increases",
			easeFactor:    2.5,
			initialStatus: StatusUnknown,
			quality:       5,
			wantStatus:    StatusConfident,
			wantMinEase:   2.5,
			wantMaxEase:   3.5,
			wantAfter:     now.Add(30 * 24 * time.Hour),
			wantLastSeen:  now,
		},
		{
			name:          "quality 1 — reset to unknown",
			easeFactor:    2.5,
			initialStatus: StatusConfident,
			quality:       1,
			wantStatus:    StatusUnknown,
			wantMinEase:   1.3,
			wantMaxEase:   2.5,
			wantAfter:     now,
			wantBefore:    now.Add(2 * time.Minute),
			wantLastSeen:  now,
		},
		{
			name:          "quality 3 — uncertain, ~1 day",
			easeFactor:    2.5,
			initialStatus: StatusUnknown,
			quality:       3,
			wantStatus:    StatusUncertain,
			wantMinEase:   2.3,
			wantMaxEase:   2.7,
			wantAfter:     now.Add(23 * time.Hour),
			wantBefore:    now.Add(25 * time.Hour),
			wantLastSeen:  now,
		},
		{
			name:          "quality 4 — uncertain, ~15 days",
			easeFactor:    2.5,
			initialStatus: StatusUnknown,
			quality:       4,
			wantStatus:    StatusUncertain,
			wantMinEase:   2.4,
			wantMaxEase:   2.7,
			wantAfter:     now.Add(14 * 24 * time.Hour),
			wantLastSeen:  now,
		},
		{
			name:          "quality 0 — ease floor at 1.3",
			easeFactor:    1.3,
			initialStatus: StatusUnknown,
			quality:       0,
			wantStatus:    StatusUnknown,
			wantMinEase:   1.3,
			wantMaxEase:   1.3,
			wantLastSeen:  now,
		},
		{
			name:          "quality -1 clamped to 0",
			easeFactor:    2.5,
			initialStatus: StatusUnknown,
			quality:       -1,
			wantStatus:    StatusUnknown,
			wantMinEase:   1.3,
			wantMaxEase:   2.5,
			wantLastSeen:  now,
		},
		{
			name:          "quality 10 clamped to 5",
			easeFactor:    2.5,
			initialStatus: StatusUnknown,
			quality:       10,
			wantStatus:    StatusConfident,
			wantMinEase:   2.5,
			wantMaxEase:   3.5,
			wantLastSeen:  now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Word{EaseFactor: tt.easeFactor, Status: tt.initialStatus}
			w.Review(tt.quality, now)

			assert.Equal(t, tt.wantStatus, w.Status)
			assert.GreaterOrEqual(t, w.EaseFactor, tt.wantMinEase)
			assert.LessOrEqual(t, w.EaseFactor, tt.wantMaxEase)
			assert.Equal(t, tt.wantLastSeen, w.LastSeen)

			if !tt.wantAfter.IsZero() {
				assert.True(t, w.NextReview.After(tt.wantAfter) || w.NextReview.Equal(tt.wantAfter),
					"NextReview %v should be after %v", w.NextReview, tt.wantAfter)
			}
			if !tt.wantBefore.IsZero() {
				assert.True(t, w.NextReview.Before(tt.wantBefore),
					"NextReview %v should be before %v", w.NextReview, tt.wantBefore)
			}
		})
	}
}

func TestWord_Review_EaseFloor_RepeatedWrong(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	w := &Word{EaseFactor: 2.5, Status: StatusConfident}

	for i := 0; i < 20; i++ {
		w.Review(0, now)
		assert.GreaterOrEqual(t, w.EaseFactor, 1.3,
			"ease factor dropped below 1.3 on iteration %d", i)
	}
	assert.Equal(t, 1.3, w.EaseFactor)
}

func TestWord_Review_SequentialTransitions(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}

	w.Review(3, now)
	assert.Equal(t, StatusUncertain, w.Status)

	w.Review(5, now)
	assert.Equal(t, StatusConfident, w.Status)
	confidentReview := w.NextReview

	w.Review(1, now)
	assert.Equal(t, StatusUnknown, w.Status)
	assert.True(t, w.NextReview.Before(confidentReview))
}

func TestWord_Review_EaseIncreases_PerfectScores(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	w := &Word{EaseFactor: 2.5, Status: StatusUnknown}

	w.Review(5, now)
	first := w.EaseFactor
	assert.Greater(t, first, 2.5)

	w.Review(5, now)
	assert.Greater(t, w.EaseFactor, first)
}
