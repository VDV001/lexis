package domain

import (
	"math"
	"time"
)

type VocabStatus string

const (
	StatusUnknown   VocabStatus = "unknown"
	StatusUncertain VocabStatus = "uncertain"
	StatusConfident VocabStatus = "confident"
)

type Word struct {
	ID         string
	UserID     string
	Word       string
	Language   string
	Status     VocabStatus
	EaseFactor float64
	NextReview time.Time
	Context    string
	LastSeen   time.Time
}

// Review applies the SM-2 algorithm based on answer quality (0-5).
// 0-2: wrong, 3-5: correct.
func (w *Word) Review(quality int) {
	if quality < 0 {
		quality = 0
	}
	if quality > 5 {
		quality = 5
	}

	// Update ease factor
	w.EaseFactor += 0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02)
	if w.EaseFactor < 1.3 {
		w.EaseFactor = 1.3
	}

	// Update status and interval
	switch {
	case quality < 3:
		// Wrong answer — reset
		w.Status = StatusUnknown
		w.NextReview = time.Now().Add(1 * time.Minute) // review soon
	case quality == 3:
		w.Status = StatusUncertain
		w.NextReview = time.Now().Add(1 * 24 * time.Hour)
	case quality == 4:
		w.Status = StatusUncertain
		days := math.Round(6 * w.EaseFactor)
		w.NextReview = time.Now().Add(time.Duration(days) * 24 * time.Hour)
	default:
		// quality 5 — confident
		w.Status = StatusConfident
		days := math.Round(6 * w.EaseFactor * w.EaseFactor)
		w.NextReview = time.Now().Add(time.Duration(days) * 24 * time.Hour)
	}

	w.LastSeen = time.Now()
}

type DailySnapshot struct {
	UserID       string
	Language     string
	SnapshotDate time.Time
	TotalWords   int
	Confident    int
	Uncertain    int
	Unknown      int
}
