package repositories

import "github.com/jackc/pgx/v5/pgxpool"

type Repositories struct {
	JobRepositoryInterface
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		JobRepositoryInterface: NewJobRepository(pool),
	}
}
