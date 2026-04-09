package infra

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/progress/domain"
)

type PostgresGoalRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresGoalRepo(pool *pgxpool.Pool) *PostgresGoalRepo {
	return &PostgresGoalRepo{pool: pool}
}

func (r *PostgresGoalRepo) ListByUser(ctx context.Context, userID string) ([]domain.Goal, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, language, progress, color, is_system, updated_at
		 FROM goals WHERE user_id = $1 ORDER BY name`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goals := make([]domain.Goal, 0)
	for rows.Next() {
		var g domain.Goal
		if err := rows.Scan(
			&g.ID, &g.UserID, &g.Name, &g.Language, &g.Progress,
			&g.Color, &g.IsSystem, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func (r *PostgresGoalRepo) CreateDefaults(ctx context.Context, userID, language string) error {
	defaults := []struct {
		name     string
		progress int
	}{
		{"Present/Past Simple", 20},
		{"Tech vocabulary", 8},
		{"Phrasal verbs", 12},
		{"Articles", 35},
		{"Prepositions", 5},
	}

	now := time.Now()
	for _, d := range defaults {
		color := domain.ColorForProgress(d.progress)
		_, err := r.pool.Exec(ctx,
			`INSERT INTO goals (user_id, name, language, progress, color, is_system, updated_at)
			 VALUES ($1, $2, $3, $4, $5, true, $6)`,
			userID, d.name, language, d.progress, color, now,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresGoalRepo) UpdateBatch(ctx context.Context, goals []domain.Goal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, g := range goals {
		_, err := tx.Exec(ctx,
			`UPDATE goals SET progress = $1, color = $2, updated_at = $3 WHERE id = $4`,
			g.Progress, g.Color, time.Now(), g.ID,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
