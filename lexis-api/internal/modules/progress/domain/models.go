package domain

import "time"

type ProgressSummary struct {
	TotalRounds   int     `json:"total_rounds"`
	CorrectRounds int     `json:"correct_rounds"`
	Accuracy      float64 `json:"accuracy"`
	Streak        int     `json:"streak"`
	TotalWords    int     `json:"total_words"`
}

type ErrorCategory struct {
	ErrorType string `json:"error_type"`
	Count     int    `json:"count"`
}

type Session struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Mode         string     `json:"mode"`
	Language     string     `json:"language"`
	Level        string     `json:"level"`
	AIModel      string     `json:"ai_model"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	RoundCount   int        `json:"round_count"`
	CorrectCount int        `json:"correct_count"`
}

type Round struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	UserID        string    `json:"user_id"`
	Mode          string    `json:"mode"`
	IsCorrect     bool      `json:"is_correct"`
	ErrorType     *string   `json:"error_type,omitempty"`
	Question      string    `json:"question"`
	UserAnswer    string    `json:"user_answer"`
	CorrectAnswer *string   `json:"correct_answer,omitempty"`
	Explanation   *string   `json:"explanation,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Goal struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Language  string    `json:"language"`
	Progress  int       `json:"progress"`
	Color     string    `json:"color"`
	IsSystem  bool      `json:"is_system"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateGoalProgress implements the logic from spec:
// No error: all goals +3%
// Has error: weakest goal +8%, other goals decay by -1
func UpdateGoalProgress(goals []Goal, hasError bool) []Goal {
	if len(goals) == 0 {
		return goals
	}
	if hasError {
		minIdx := 0
		for i, g := range goals {
			if g.Progress < goals[minIdx].Progress {
				minIdx = i
			}
		}
		goals[minIdx].Progress += 8
		if goals[minIdx].Progress > 100 {
			goals[minIdx].Progress = 100
		}
		goals[minIdx].Color = ColorForProgress(goals[minIdx].Progress)
		// Decay other goals slightly on error
		for i := range goals {
			if i != minIdx {
				goals[i].Progress -= 1
				if goals[i].Progress < 0 {
					goals[i].Progress = 0
				}
				goals[i].Color = ColorForProgress(goals[i].Progress)
			}
		}
	} else {
		for i := range goals {
			goals[i].Progress += 3
			if goals[i].Progress > 100 {
				goals[i].Progress = 100
			}
			goals[i].Color = ColorForProgress(goals[i].Progress)
		}
	}
	return goals
}

func ColorForProgress(pct int) string {
	if pct >= 70 {
		return "green"
	}
	if pct >= 40 {
		return "amber"
	}
	return "red"
}
