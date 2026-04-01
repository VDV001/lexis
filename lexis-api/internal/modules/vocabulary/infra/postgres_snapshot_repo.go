package infra

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
)

type PostgresSnapshotRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresSnapshotRepo(pool *pgxpool.Pool) *PostgresSnapshotRepo {
	return &PostgresSnapshotRepo{pool: pool}
}

func (r *PostgresSnapshotRepo) Create(ctx context.Context, snapshot *domain.DailySnapshot) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO vocabulary_daily_snapshots (user_id, language, snapshot_date, total_words, confident, uncertain, unknown)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, language, snapshot_date)
		 DO UPDATE SET total_words = EXCLUDED.total_words, confident = EXCLUDED.confident, uncertain = EXCLUDED.uncertain, unknown = EXCLUDED.unknown`,
		snapshot.UserID, snapshot.Language, snapshot.SnapshotDate,
		snapshot.TotalWords, snapshot.Confident, snapshot.Uncertain, snapshot.Unknown,
	)
	return err
}

func (r *PostgresSnapshotRepo) GetByDateRange(ctx context.Context, userID, language string, from, to time.Time) ([]domain.DailySnapshot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, language, snapshot_date, total_words, confident, uncertain, unknown
		 FROM vocabulary_daily_snapshots
		 WHERE user_id = $1 AND language = $2
		   AND snapshot_date BETWEEN $3 AND $4
		 ORDER BY snapshot_date`,
		userID, language, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []domain.DailySnapshot
	for rows.Next() {
		var s domain.DailySnapshot
		if err := rows.Scan(
			&s.UserID, &s.Language, &s.SnapshotDate,
			&s.TotalWords, &s.Confident, &s.Uncertain, &s.Unknown,
		); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}
