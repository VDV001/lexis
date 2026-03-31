package infra

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
)

type PostgresUserRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepo(pool *pgxpool.Pool) *PostgresUserRepo {
	return &PostgresUserRepo{pool: pool}
}

func (r *PostgresUserRepo) Create(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, display_name, avatar_url, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.AvatarURL, user.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrEmailTaken
		}
		return err
	}
	return nil
}

func (r *PostgresUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, avatar_url, created_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return scanUser(row)
}

func (r *PostgresUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, avatar_url, created_at, deleted_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`, email,
	)
	return scanUser(row)
}

func (r *PostgresUserRepo) Update(ctx context.Context, user *domain.User) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET display_name = $1, avatar_url = $2
		 WHERE id = $3 AND deleted_at IS NULL`,
		user.DisplayName, user.AvatarURL, user.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
