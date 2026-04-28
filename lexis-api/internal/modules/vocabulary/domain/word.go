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

func (s VocabStatus) IsValid() bool {
	switch s {
	case StatusUnknown, StatusUncertain, StatusConfident:
		return true
	}
	return false
}

type Word struct {
	ID         string      `json:"id"`
	UserID     string      `json:"user_id"`
	Word       string      `json:"word"`
	Language   string      `json:"language"`
	Status     VocabStatus `json:"status"`
	EaseFactor float64     `json:"ease_factor"`
	NextReview time.Time   `json:"next_review"`
	Context    string      `json:"context"`
	LastSeen   time.Time   `json:"last_seen"`
}

// Review applies the SM-2 algorithm based on answer quality (0-5).
// 0-2: wrong, 3-5: correct.
func (w *Word) Review(quality int, now time.Time) {
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
		w.Status = StatusUnknown
		w.NextReview = now.Add(1 * time.Minute)
	case quality == 3:
		w.Status = StatusUncertain
		w.NextReview = now.Add(1 * 24 * time.Hour)
	case quality == 4:
		w.Status = StatusUncertain
		days := math.Round(6 * w.EaseFactor)
		w.NextReview = now.Add(time.Duration(days) * 24 * time.Hour)
	default:
		w.Status = StatusConfident
		days := math.Round(6 * w.EaseFactor * w.EaseFactor)
		w.NextReview = now.Add(time.Duration(days) * 24 * time.Hour)
	}

	w.LastSeen = now
}

type DailySnapshot struct {
	UserID       string    `json:"user_id"`
	Language     string    `json:"language"`
	SnapshotDate time.Time `json:"date"`
	TotalWords   int       `json:"total"`
	Confident    int       `json:"confident"`
	Uncertain    int       `json:"uncertain"`
	Unknown      int       `json:"unknown"`
}
