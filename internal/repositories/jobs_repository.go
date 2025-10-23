package repositories

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	// log  *slog.Logger
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{
		pool: pool,
	}
}
