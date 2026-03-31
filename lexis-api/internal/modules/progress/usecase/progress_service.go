package usecase

import (
	"context"
	"time"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	progressDomain "github.com/lexis-app/lexis-api/internal/modules/progress/domain"
	vocabDomain "github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

type ProgressService struct {
	rounds   progressDomain.RoundRepository
	sessions progressDomain.SessionRepository
	goals    progressDomain.GoalRepository
	words    vocabDomain.WordRepository
	snaps    vocabDomain.SnapshotRepository
	settings authDomain.SettingsRepository
}

func NewProgressService(
	rounds progressDomain.RoundRepository,
	sessions progressDomain.SessionRepository,
	goals progressDomain.GoalRepository,
	words vocabDomain.WordRepository,
	snaps vocabDomain.SnapshotRepository,
	settings authDomain.SettingsRepository,
) *ProgressService {
	return &ProgressService{rounds: rounds, sessions: sessions, goals: goals, words: words, snaps: snaps, settings: settings}
}

func (s *ProgressService) GetSummary(ctx context.Context, userID string) (*progressDomain.ProgressSummary, error) {
	total, correct, err := s.rounds.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	streak, err := s.rounds.GetStreak(ctx, userID)
	if err != nil {
		return nil, err
	}

	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	totalWords, _, _, _, err := s.words.CountByStatus(ctx, userID, settings.TargetLanguage)
	if err != nil {
		return nil, err
	}

	var accuracy float64
	if total > 0 {
		accuracy = float64(correct) / float64(total) * 100
	}

	return &progressDomain.ProgressSummary{
		TotalRounds:   total,
		CorrectRounds: correct,
		Accuracy:      accuracy,
		Streak:        streak,
		TotalWords:    totalWords,
	}, nil
}

type VocabStats struct {
	Total     int `json:"total"`
	Confident int `json:"confident"`
	Uncertain int `json:"uncertain"`
	Unknown   int `json:"unknown"`
}

func (s *ProgressService) GetVocabulary(ctx context.Context, userID string) (*VocabStats, error) {
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	total, confident, uncertain, unknown, err := s.words.CountByStatus(ctx, userID, settings.TargetLanguage)
	if err != nil {
		return nil, err
	}
	return &VocabStats{Total: total, Confident: confident, Uncertain: uncertain, Unknown: unknown}, nil
}

type VocabCurveResponse struct {
	Goal    int                        `json:"goal"`
	Current VocabStats                 `json:"current"`
	Daily   []vocabDomain.DailySnapshot `json:"daily_snapshots"`
}

func (s *ProgressService) GetVocabCurve(ctx context.Context, userID string) (*VocabCurveResponse, error) {
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	current, err := s.GetVocabulary(ctx, userID)
	if err != nil {
		return nil, err
	}

	to := time.Now().UTC()
	from := to.AddDate(0, 0, -30)
	snapshots, err := s.snaps.GetByDateRange(ctx, userID, settings.TargetLanguage, from, to)
	if err != nil {
		return nil, err
	}

	return &VocabCurveResponse{
		Goal:    settings.VocabGoal,
		Current: *current,
		Daily:   snapshots,
	}, nil
}

func (s *ProgressService) GetGoals(ctx context.Context, userID string) ([]progressDomain.Goal, error) {
	return s.goals.ListByUser(ctx, userID)
}

func (s *ProgressService) GetErrors(ctx context.Context, userID string) ([]progressDomain.ErrorCategory, error) {
	return s.rounds.GetErrorCounts(ctx, userID)
}

func (s *ProgressService) GetSessions(ctx context.Context, userID string, limit, offset int) ([]progressDomain.Session, error) {
	return s.sessions.ListByUser(ctx, userID, limit, offset)
}

func (s *ProgressService) GetSession(ctx context.Context, sessionID string) (*progressDomain.Session, error) {
	return s.sessions.GetByID(ctx, sessionID)
}
