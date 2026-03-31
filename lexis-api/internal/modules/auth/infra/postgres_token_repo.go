package infra

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

type PostgresTokenRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresTokenRepo(pool *pgxpool.Pool) *PostgresTokenRepo {
	return &PostgresTokenRepo{pool: pool}
}

func (r *PostgresTokenRepo) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, user_agent, ip_address, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.UserAgent, token.IPAddress, token.CreatedAt,
	)
	return err
}

func (r *PostgresTokenRepo) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, user_agent, ip_address, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, hash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.UserAgent, &t.IPAddress, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *PostgresTokenRepo) RevokeByHash(ctx context.Context, hash string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`, hash,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTokenNotFound
	}
	return nil
}

func (r *PostgresTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID,
	)
	return err
}
