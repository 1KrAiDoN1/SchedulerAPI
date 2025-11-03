package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type JobsRepository struct {
	pool *pgxpool.Pool
}

func NewJobsRepository(pool *pgxpool.Pool) *JobsRepository {
	return &JobsRepository{
		pool: pool,
	}
}
