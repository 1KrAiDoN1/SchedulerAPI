package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"scheduler/internal/domain/entity"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExecutionsRepository struct {
	pool *pgxpool.Pool
}

func NewExecutionsRepository(pool *pgxpool.Pool) *ExecutionsRepository {
	return &ExecutionsRepository{
		pool: pool,
	}
}

const (
	queryCreateExecution = `INSERT INTO executions (id, job_id, worker_id, status, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO UPDATE SET
		status = EXCLUDED.status, finished_at = EXCLUDED.finished_at`

	queryGetExecutionByJobId = `SELECT id, job_id, worker_id, status, started_at, finished_at
		FROM executions WHERE job_id = $1 ORDER BY started_at DESC`
)

func (e *ExecutionsRepository) Create(ctx context.Context, execution *entity.Execution) error {
	var finishedAt sql.NullInt64
	if execution.FinishedAt > 0 {
		finishedAt = sql.NullInt64{
			Int64: execution.FinishedAt,
			Valid: true,
		}
	}

	_, err := e.pool.Exec(ctx, queryCreateExecution,
		execution.ID,
		execution.JobID,
		execution.WorkerID,
		execution.Status,
		execution.StartedAt,
		finishedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

func (e *ExecutionsRepository) GetByJobID(ctx context.Context, jobID string) ([]*entity.Execution, error) {
	rows, err := e.pool.Query(ctx, queryGetExecutionByJobId, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer rows.Close()

	var executions []*entity.Execution
	for rows.Next() {
		var (
			id         string
			jobID      string
			workerID   string
			status     string
			startedAt  int64
			finishedAt sql.NullInt64
		)

		if err := rows.Scan(
			&id,
			&jobID,
			&workerID,
			&status,
			&startedAt,
			&finishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		execution := &entity.Execution{
			ID:        id,
			JobID:     jobID,
			WorkerID:  workerID,
			Status:    status,
			StartedAt: startedAt,
		}

		if finishedAt.Valid {
			execution.FinishedAt = finishedAt.Int64
		}

		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if len(executions) == 0 {
		return []*entity.Execution{}, nil
	}

	return executions, nil
}
