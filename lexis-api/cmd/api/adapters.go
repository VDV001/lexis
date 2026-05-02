package main

import (
	"context"

	authDomain "github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	vocabUsecase "github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
)

// authSettingsRepo is the cross-module surface area: anything that returns
// auth-domain UserSettings and is shared with another module funnels through
// these adapters. main.go is the only place allowed to know both auth/domain
// and the consuming module's port.
type authSettingsRepo interface {
	GetByUserID(ctx context.Context, userID string) (*authDomain.UserSettings, error)
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
