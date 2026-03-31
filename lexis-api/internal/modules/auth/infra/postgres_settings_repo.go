package infra

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

type PostgresSettingsRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresSettingsRepo(pool *pgxpool.Pool) *PostgresSettingsRepo {
	return &PostgresSettingsRepo{pool: pool}
}

func (r *PostgresSettingsRepo) GetByUserID(ctx context.Context, userID string) (*domain.UserSettings, error) {
	var s domain.UserSettings
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, target_language, proficiency_level, vocabulary_type, ai_model, vocab_goal, ui_language, updated_at
		 FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&s.UserID, &s.TargetLanguage, &s.ProficiencyLevel, &s.VocabularyType, &s.AIModel, &s.VocabGoal, &s.UILanguage, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *PostgresSettingsRepo) Upsert(ctx context.Context, settings *domain.UserSettings) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, target_language, proficiency_level, vocabulary_type, ai_model, vocab_goal, ui_language, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		 ON CONFLICT (user_id) DO UPDATE SET
		     target_language   = EXCLUDED.target_language,
		     proficiency_level = EXCLUDED.proficiency_level,
		     vocabulary_type   = EXCLUDED.vocabulary_type,
		     ai_model          = EXCLUDED.ai_model,
		     vocab_goal        = EXCLUDED.vocab_goal,
		     ui_language       = EXCLUDED.ui_language,
		     updated_at        = now()`,
		settings.UserID, settings.TargetLanguage, settings.ProficiencyLevel, settings.VocabularyType, settings.AIModel, settings.VocabGoal, settings.UILanguage,
	)
	return err
}
