package main

import (
	"context"
	"time"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	progressUsecase "github.com/lexis-app/lexis-api/internal/modules/progress/usecase"
	tutorUsecase "github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	vocabDomain "github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
	vocabUsecase "github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
)

// authSettingsRepo / authUserRepo are the cross-module surface area: anything
// that returns auth-domain types and is shared with another module funnels
// through these adapters. main.go is the only place allowed to know both
// auth/domain and the consuming module's port.
type authSettingsRepo interface {
	GetByUserID(ctx context.Context, userID string) (*authDomain.UserSettings, error)
}

type authUserRepo interface {
	GetByID(ctx context.Context, id string) (*authDomain.User, error)
}

// vocabSettingsAdapter projects auth UserSettings down to the narrow view
// the vocabulary module's usecase package consumes.
type vocabSettingsAdapter struct{ inner authSettingsRepo }

func (a vocabSettingsAdapter) GetByUserID(ctx context.Context, userID string) (*vocabUsecase.UserSettingsView, error) {
	s, err := a.inner.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &vocabUsecase.UserSettingsView{TargetLanguage: s.TargetLanguage}, nil
}

// tutorSettingsAdapter projects auth UserSettings to the tutor module's view.
type tutorSettingsAdapter struct{ inner authSettingsRepo }

func (a tutorSettingsAdapter) GetByUserID(ctx context.Context, userID string) (*tutorUsecase.UserSettingsView, error) {
	s, err := a.inner.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &tutorUsecase.UserSettingsView{
		TargetLanguage:   s.TargetLanguage,
		ProficiencyLevel: s.ProficiencyLevel,
		VocabularyType:   s.VocabularyType,
		AIModel:          s.AIModel,
	}, nil
}

// tutorUserAdapter projects auth User to the tutor module's view.
type tutorUserAdapter struct{ inner authUserRepo }

func (a tutorUserAdapter) GetByID(ctx context.Context, id string) (*tutorUsecase.UserView, error) {
	u, err := a.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &tutorUsecase.UserView{DisplayName: u.DisplayName}, nil
}

// progressSettingsAdapter projects auth UserSettings to the progress view.
type progressSettingsAdapter struct{ inner authSettingsRepo }

func (a progressSettingsAdapter) GetByUserID(ctx context.Context, userID string) (*progressUsecase.UserSettingsView, error) {
	s, err := a.inner.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &progressUsecase.UserSettingsView{
		TargetLanguage: s.TargetLanguage,
		VocabGoal:      s.VocabGoal,
	}, nil
}

// vocabSnapshotsRepo is the upstream port for vocabulary daily snapshots.
type vocabSnapshotsRepo interface {
	GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]vocabDomain.DailySnapshot, error)
}

// progressSnapshotAdapter projects vocabulary's DailySnapshot to the
// progress module's view, decoupling progress/usecase from vocab/domain.
type progressSnapshotAdapter struct{ inner vocabSnapshotsRepo }

func (a progressSnapshotAdapter) GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]progressUsecase.DailySnapshotView, error) {
	src, err := a.inner.GetByDateRange(ctx, userID, language, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]progressUsecase.DailySnapshotView, len(src))
	for i, s := range src {
		out[i] = progressUsecase.DailySnapshotView{
			UserID:       s.UserID,
			Language:     s.Language,
			SnapshotDate: s.SnapshotDate,
			TotalWords:   s.TotalWords,
			Confident:    s.Confident,
			Uncertain:    s.Uncertain,
			Unknown:      s.Unknown,
		}
	}
	return out, nil
}
