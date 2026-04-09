package infra

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

type PostgresWordRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresWordRepo(pool *pgxpool.Pool) *PostgresWordRepo {
	return &PostgresWordRepo{pool: pool}
}

func (r *PostgresWordRepo) Upsert(ctx context.Context, word *domain.Word) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO vocabulary_words (id, user_id, word, language, status, ease_factor, next_review, context, last_seen)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (user_id, word, language) DO UPDATE SET
		     status      = EXCLUDED.status,
		     ease_factor = EXCLUDED.ease_factor,
		     next_review = EXCLUDED.next_review,
		     context     = EXCLUDED.context,
		     last_seen   = EXCLUDED.last_seen`,
		word.ID, word.UserID, word.Word, word.Language,
		word.Status, word.EaseFactor, word.NextReview, word.Context, word.LastSeen,
	)
	return err
}

func (r *PostgresWordRepo) UpsertBatch(ctx context.Context, words []*domain.Word) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, word := range words {
		_, err := tx.Exec(ctx,
			`INSERT INTO vocabulary_words (id, user_id, word, language, status, ease_factor, next_review, context, last_seen)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (user_id, word, language) DO UPDATE SET
			     context   = EXCLUDED.context,
			     last_seen = EXCLUDED.last_seen`,
			word.ID, word.UserID, word.Word, word.Language,
			word.Status, word.EaseFactor, word.NextReview, word.Context, word.LastSeen,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PostgresWordRepo) GetByUserAndWord(ctx context.Context, userID, word, language string) (*domain.Word, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, word, language, status, ease_factor, next_review, context, last_seen
		 FROM vocabulary_words
		 WHERE user_id = $1 AND word = $2 AND language = $3`,
		userID, word, language,
	)
	return scanWord(row)
}

func (r *PostgresWordRepo) ListByUser(ctx context.Context, userID, language string, limit, offset int) ([]domain.Word, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, word, language, status, ease_factor, next_review, context, last_seen
		 FROM vocabulary_words
		 WHERE user_id = $1 AND language = $2
		 ORDER BY last_seen DESC
		 LIMIT $3 OFFSET $4`,
		userID, language, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectWords(rows)
}

func (r *PostgresWordRepo) CountByStatus(ctx context.Context, userID, language string) (total, confident, uncertain, unknown int, err error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM vocabulary_words
		 WHERE user_id = $1 AND language = $2
		 GROUP BY status`,
		userID, language,
	)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err = rows.Scan(&status, &count); err != nil {
			return 0, 0, 0, 0, err
		}
		total += count
		switch domain.VocabStatus(status) {
		case domain.StatusConfident:
			confident = count
		case domain.StatusUncertain:
			uncertain = count
		case domain.StatusUnknown:
			unknown = count
		}
	}

	return total, confident, uncertain, unknown, rows.Err()
}

func (r *PostgresWordRepo) GetDueForReview(ctx context.Context, userID, language string, limit int) ([]domain.Word, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, word, language, status, ease_factor, next_review, context, last_seen
		 FROM vocabulary_words
		 WHERE user_id = $1 AND language = $2
		   AND next_review <= now()
		   AND status != 'confident'
		 ORDER BY next_review
		 LIMIT $3`,
		userID, language, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectWords(rows)
}

func (r *PostgresWordRepo) Delete(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM vocabulary_words WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresWordRepo) UpdateStatus(ctx context.Context, id, userID string, status domain.VocabStatus) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE vocabulary_words SET status = $1, last_seen = now() WHERE id = $2 AND user_id = $3`,
		status, id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresWordRepo) ListDistinctUserLanguages(ctx context.Context) ([]domain.UserLanguage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT user_id, language
		FROM vocabulary_words
		WHERE user_id IN (SELECT id FROM users WHERE deleted_at IS NULL)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairs := make([]domain.UserLanguage, 0)
	for rows.Next() {
		var ul domain.UserLanguage
		if err := rows.Scan(&ul.UserID, &ul.Language); err != nil {
			return nil, err
		}
		pairs = append(pairs, ul)
	}
	return pairs, rows.Err()
}

func scanWord(row pgx.Row) (*domain.Word, error) {
	var w domain.Word
	err := row.Scan(
		&w.ID, &w.UserID, &w.Word, &w.Language,
		&w.Status, &w.EaseFactor, &w.NextReview, &w.Context, &w.LastSeen,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func collectWords(rows pgx.Rows) ([]domain.Word, error) {
	words := make([]domain.Word, 0)
	for rows.Next() {
		var w domain.Word
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.Word, &w.Language,
			&w.Status, &w.EaseFactor, &w.NextReview, &w.Context, &w.LastSeen,
		); err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, rows.Err()
}
