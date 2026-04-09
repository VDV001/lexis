package infra

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/progress/domain"
)

type PostgresRoundRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRoundRepo(pool *pgxpool.Pool) *PostgresRoundRepo {
	return &PostgresRoundRepo{pool: pool}
}

func (r *PostgresRoundRepo) Create(ctx context.Context, round *domain.Round) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rounds (id, session_id, user_id, mode, is_correct, error_type, question, user_answer, correct_answer, explanation, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		round.ID, round.SessionID, round.UserID, round.Mode, round.IsCorrect,
		round.ErrorType, round.Question, round.UserAnswer, round.CorrectAnswer,
		round.Explanation, round.CreatedAt,
	)
	return err
}

func (r *PostgresRoundRepo) CountByUser(ctx context.Context, userID string) (total, correct int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END), 0)
		 FROM rounds WHERE user_id = $1`, userID,
	).Scan(&total, &correct)
	return
}

func (r *PostgresRoundRepo) GetStreak(ctx context.Context, userID string) (int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT is_correct FROM rounds WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1000`, userID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	streak := 0
	for rows.Next() {
		var correct bool
		if err := rows.Scan(&correct); err != nil {
			return 0, err
		}
		if !correct {
			break
		}
		streak++
	}
	return streak, rows.Err()
}

func (r *PostgresRoundRepo) GetErrorCounts(ctx context.Context, userID string) ([]domain.ErrorCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT error_type, COUNT(*) FROM rounds
		 WHERE user_id = $1 AND NOT is_correct AND error_type IS NOT NULL
		 GROUP BY error_type ORDER BY COUNT(*) DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]domain.ErrorCategory, 0)
	for rows.Next() {
		var ec domain.ErrorCategory
		if err := rows.Scan(&ec.ErrorType, &ec.Count); err != nil {
			return nil, err
		}
		categories = append(categories, ec)
	}
	return categories, rows.Err()
}
