package infra

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/progress/domain"
)

type PostgresSessionRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepo(pool *pgxpool.Pool) *PostgresSessionRepo {
	return &PostgresSessionRepo{pool: pool}
}

func (r *PostgresSessionRepo) Create(ctx context.Context, session *domain.Session) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, mode, language, level, ai_model, started_at, ended_at, round_count, correct_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		session.ID, session.UserID, session.Mode, session.Language, session.Level,
		session.AIModel, session.StartedAt, session.EndedAt, session.RoundCount, session.CorrectCount,
	)
	return err
}

func (r *PostgresSessionRepo) GetByID(ctx context.Context, id, userID string) (*domain.Session, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, mode, language, level, ai_model, started_at, ended_at, round_count, correct_count
		 FROM sessions WHERE id = $1 AND user_id = $2`, id, userID,
	)
	return scanSession(row)
}

func (r *PostgresSessionRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Session, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, mode, language, level, ai_model, started_at, ended_at, round_count, correct_count
		 FROM sessions WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Mode, &s.Language, &s.Level,
			&s.AIModel, &s.StartedAt, &s.EndedAt, &s.RoundCount, &s.CorrectCount,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *PostgresSessionRepo) Update(ctx context.Context, session *domain.Session) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE sessions SET ended_at = $1, round_count = $2, correct_count = $3
		 WHERE id = $4`,
		session.EndedAt, session.RoundCount, session.CorrectCount, session.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostgresSessionRepo) IncrementCounters(ctx context.Context, id string, correct bool) error {
	correctInc := 0
	if correct {
		correctInc = 1
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE sessions SET round_count = round_count + 1, correct_count = correct_count + $1
		 WHERE id = $2`,
		correctInc, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanSession(row pgx.Row) (*domain.Session, error) {
	var s domain.Session
	err := row.Scan(
		&s.ID, &s.UserID, &s.Mode, &s.Language, &s.Level,
		&s.AIModel, &s.StartedAt, &s.EndedAt, &s.RoundCount, &s.CorrectCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	return &s, nil
}
