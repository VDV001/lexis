package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

var ErrInvalidStatus = errors.New("invalid vocabulary status")

type VocabService struct {
	words    WordRepository
	settings SettingsReader
}

func NewVocabService(words WordRepository, settings SettingsReader) *VocabService {
	return &VocabService{words: words, settings: settings}
}

type AddWordInput struct {
	UserID   string
	Word     string
	Language string
	Status   domain.VocabStatus
	Context  string
}

func (s *VocabService) AddWord(ctx context.Context, input AddWordInput) (*domain.Word, error) {
	language := input.Language
	if language == "" {
		settings, err := s.settings.GetByUserID(ctx, input.UserID)
		if err != nil {
			return nil, err
		}
		language = settings.TargetLanguage
	}

	status := input.Status
	if status == "" {
		status = domain.StatusUnknown
	}
	if !status.IsValid() {
		return nil, ErrInvalidStatus
	}

	now := time.Now().UTC()
	word, err := domain.NewWord(input.UserID, input.Word, language, input.Context, now)
	if err != nil {
		return nil, err
	}
	if status != domain.StatusUnknown {
		word.Status = status
	}

	if err := s.words.Upsert(ctx, word); err != nil {
		return nil, err
	}
	return word, nil
}

func (s *VocabService) AddDiscoveredWords(ctx context.Context, userID, language string, words []string, wordContext string) error {
	now := time.Now().UTC()
	batch := make([]*domain.Word, 0, len(words))
	for _, w := range words {
		word, err := domain.NewWord(userID, w, language, wordContext, now)
		if err != nil {
			return err
		}
		batch = append(batch, word)
	}
	return s.words.UpsertBatch(ctx, batch)
}

func (s *VocabService) ListWords(ctx context.Context, userID string, limit, offset int) ([]domain.Word, error) {
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.words.ListByUser(ctx, userID, settings.TargetLanguage, limit, offset)
}

func (s *VocabService) DeleteWord(ctx context.Context, wordID, userID string) error {
	return s.words.Delete(ctx, wordID, userID)
}

func (s *VocabService) UpdateStatus(ctx context.Context, wordID, userID string, status domain.VocabStatus) error {
	if !status.IsValid() {
		return ErrInvalidStatus
	}
	return s.words.UpdateStatus(ctx, wordID, userID, status)
}

func (s *VocabService) GetDueForReview(ctx context.Context, userID string, limit int) ([]domain.Word, error) {
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.words.GetDueForReview(ctx, userID, settings.TargetLanguage, limit)
}
